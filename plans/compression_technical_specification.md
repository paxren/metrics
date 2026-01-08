# Техническая спецификация: Оптимизация сжатия с использованием пулов объектов

## Обзор

Документ описывает техническую реализацию оптимизации системы сжатия HTTP ответов и запросов с использованием пулов объектов для снижения аллокаций памяти и улучшения производительности.

## Архитектура

### Компоненты

1. **WriterPool** - пул объектов `gzip.Writer`
2. **ReaderPool** - пул объектов `gzip.Reader`
3. **CompressionManager** - менеджер пулов и конфигурации
4. **OptimizedCompressWriter** - оптимизированный компрессор с поддержкой пулов
5. **OptimizedCompressReader** - оптимизированный декомпрессор с поддержкой пулов

### Структура файлов

```
internal/handler/
├── compressor.go          # Текущая реализация
├── compressor_optimized.go # Новая оптимизированная реализация
├── compression_pools.go    # Реализация пулов
├── compression_config.go   # Конфигурация
└── compressor_test.go      # Тесты
```

## Детальная реализация

### 1. Пул для gzip.Writer (compression_pools.go)

```go
package handler

import (
    "compress/gzip"
    "io"
    "sync"
)

// WriterPool управляет пулом gzip.Writer для переиспользования
type WriterPool struct {
    pool sync.Pool
}

// NewWriterPool создает новый пул для gzip.Writer
func NewWriterPool() *WriterPool {
    return &WriterPool{
        pool: sync.Pool{
            New: func() interface{} {
                // Создаем writer с io.Discard как заглушку
                // Реальный writer будет установлен через Reset()
                return gzip.NewWriter(io.Discard)
            },
        },
    }
}

// Get получает gzip.Writer из пула и настраивает его для работы с w
func (p *WriterPool) Get(w io.Writer) *gzip.Writer {
    zw := p.pool.Get().(*gzip.Writer)
    zw.Reset(w)
    return zw
}

// Put возвращает gzip.Writer в пул после использования
func (p *WriterPool) Put(zw *gzip.Writer) {
    if zw != nil {
        zw.Close() // Закрываем для сброса буферов
        p.pool.Put(zw)
    }
}
```

### 2. Пул для gzip.Reader (compression_pools.go)

```go
// ReaderPool управляет пулом gzip.Reader для переиспользования
type ReaderPool struct {
    pool sync.Pool
}

// NewReaderPool создает новый пул для gzip.Reader
func NewReaderPool() *ReaderPool {
    return &ReaderPool{
        pool: sync.Pool{
            New: func() interface{} {
                return new(gzip.Reader)
            },
        },
    }
}

// Get получает gzip.Reader из пула и настраивает его для работы с r
func (p *ReaderPool) Get(r io.Reader) (*gzip.Reader, error) {
    zr := p.pool.Get().(*gzip.Reader)
    err := zr.Reset(r)
    if err != nil {
        // При ошибке возвращаем в пул и возвращаем ошибку
        p.pool.Put(zr)
        return nil, err
    }
    return zr, nil
}

// Put возвращает gzip.Reader в пул после использования
func (p *ReaderPool) Put(zr *gzip.Reader) {
    if zr != nil {
        zr.Close() // Закрываем для очистки
        p.pool.Put(zr)
    }
}
```

### 3. Конфигурация (compression_config.go)

```go
package handler

import (
    "github.com/caarlos0/env/v11"
)

// CompressionConfig содержит настройки для системы сжатия
type CompressionConfig struct {
    // WriterPoolSize максимальный размер пула писателей (экспериментальный)
    WriterPoolSize int `env:"COMPRESSION_WRITER_POOL_SIZE" envDefault:"100"`
    
    // ReaderPoolSize максимальный размер пула читателей (экспериментальный)
    ReaderPoolSize int `env:"COMPRESSION_READER_POOL_SIZE" envDefault:"50"`
    
    // CompressionLevel уровень сжатия gzip (1-9, 6 по умолчанию)
    CompressionLevel int `env:"COMPRESSION_LEVEL" envDefault:"6"`
    
    // EnableCompression включение/выключение сжатия
    EnableCompression bool `env:"ENABLE_COMPRESSION" envDefault:"true"`
    
    // MinContentSize минимальный размер контента для сжатия в байтах
    MinContentSize int `env:"COMPRESSION_MIN_CONTENT_SIZE" envDefault:"1024"`
}

// ParseConfig парсит конфигурацию из переменных окружения
func ParseCompressionConfig() (*CompressionConfig, error) {
    cfg := &CompressionConfig{}
    err := env.Parse(cfg)
    if err != nil {
        return nil, err
    }
    
    // Валидация уровня сжатия
    if cfg.CompressionLevel < 1 || cfg.CompressionLevel > 9 {
        cfg.CompressionLevel = 6
    }
    
    return cfg, nil
}
```

### 4. Менеджер сжатия (compression_pools.go)

```go
// CompressionManager управляет всеми пулами и конфигурацией
type CompressionManager struct {
    writerPool *WriterPool
    readerPool *ReaderPool
    config     *CompressionConfig
}

// NewCompressionManager создает новый менеджер сжатия
func NewCompressionManager(config *CompressionConfig) *CompressionManager {
    return &CompressionManager{
        writerPool: NewWriterPool(),
        readerPool: NewReaderPool(),
        config:     config,
    }
}

// GetWriterPool возвращает пул писателей
func (cm *CompressionManager) GetWriterPool() *WriterPool {
    return cm.writerPool
}

// GetReaderPool возвращает пул читателей
func (cm *CompressionManager) GetReaderPool() *ReaderPool {
    return cm.readerPool
}

// GetConfig возвращает конфигурацию
func (cm *CompressionManager) GetConfig() *CompressionConfig {
    return cm.config
}

// IsCompressionEnabled проверяет включено ли сжатие
func (cm *CompressionManager) IsCompressionEnabled() bool {
    return cm.config.EnableCompression
}

// ShouldCompress проверяет, нужно ли сжимать контент
func (cm *CompressionManager) ShouldCompress(contentType string, contentLength int) bool {
    if !cm.config.EnableCompression {
        return false
    }
    
    if contentLength > 0 && contentLength < cm.config.MinContentSize {
        return false
    }
    
    // Проверяем тип контента
    return contentType == "application/json" || 
           contentType == "text/html" || 
           contentType == "text/plain" ||
           contentType == "text/css" ||
           contentType == "application/javascript"
}
```

### 5. Оптимизированный компрессор (compressor_optimized.go)

```go
package handler

import (
    "compress/gzip"
    "fmt"
    "io"
    "net/http"
    "strings"
    "sync"
)

// OptimizedCompressWriter реализует http.ResponseWriter с поддержкой пулов
type OptimizedCompressWriter struct {
    w               http.ResponseWriter
    zw              *gzip.Writer
    pool            *WriterPool
    manager         *CompressionManager
    needCompress    bool
    needCompressSet bool
    mu              sync.Mutex // Защита от гонок
}

// NewOptimizedCompressWriter создает новый оптимизированный компрессор
func NewOptimizedCompressWriter(w http.ResponseWriter, manager *CompressionManager) *OptimizedCompressWriter {
    return &OptimizedCompressWriter{
        w:               w,
        zw:              nil,
        pool:            manager.GetWriterPool(),
        manager:         manager,
        needCompress:    false,
        needCompressSet: false,
    }
}

// Header возвращает заголовки ответа
func (c *OptimizedCompressWriter) Header() http.Header {
    return c.w.Header()
}

// Write записывает данные с сжатием если необходимо
func (c *OptimizedCompressWriter) Write(p []byte) (int, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if !c.needCompressSet {
        fmt.Println("ERROR: needCompress not set before Write")
    }
    
    if c.needCompress {
        if c.zw == nil {
            c.zw = c.pool.Get(c.w)
        }
        return c.zw.Write(p)
    }
    return c.w.Write(p)
}

// WriteHeader устанавливает заголовки и определяет нужно ли сжатие
func (c *OptimizedCompressWriter) WriteHeader(statusCode int) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    contentType := c.w.Header().Get("Content-Type")
    contentLength := 0
    if lengthStr := c.w.Header().Get("Content-Length"); lengthStr != "" {
        fmt.Sscanf(lengthStr, "%d", &contentLength)
    }
    
    c.needCompress = statusCode < 300 && 
                     c.manager.ShouldCompress(contentType, contentLength)
    c.needCompressSet = true
    
    if c.needCompress {
        c.w.Header().Set("Content-Encoding", "gzip")
        c.w.Header().Del("Content-Length") // Длина изменится после сжатия
    }
    
    fmt.Printf("Compression enabled: %v, ContentType: %s, StatusCode: %d\n", 
        c.needCompress, contentType, statusCode)
    
    c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и возвращает его в пул
func (c *OptimizedCompressWriter) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.zw != nil {
        err := c.zw.Close()
        c.pool.Put(c.zw)
        c.zw = nil
        return err
    }
    return nil
}

// OptimizedCompressReader реализует io.ReadCloser с поддержкой пулов
type OptimizedCompressReader struct {
    r      io.ReadCloser
    zr     *gzip.Reader
    pool   *ReaderPool
    closed bool
    mu     sync.Mutex
}

// NewOptimizedCompressReader создает новый оптимизированный читатель
func NewOptimizedCompressReader(r io.ReadCloser, pool *ReaderPool) (*OptimizedCompressReader, error) {
    zr, err := pool.Get(r)
    if err != nil {
        return nil, err
    }
    
    return &OptimizedCompressReader{
        r:      r,
        zr:     zr,
        pool:   pool,
        closed: false,
    }, nil
}

// Read читает данные с декомпрессией
func (c *OptimizedCompressReader) Read(p []byte) (n int, err error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.closed {
        return 0, io.EOF
    }
    
    return c.zr.Read(p)
}

// Close закрывает читатели и возвращает gzip.Reader в пул
func (c *OptimizedCompressReader) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.closed {
        return nil
    }
    
    c.closed = true
    
    // Возвращаем gzip.Reader в пул
    if c.zr != nil {
        c.pool.Put(c.zr)
        c.zr = nil
    }
    
    // Закрываем исходный reader
    return c.r.Close()
}
```

### 6. Оптимизированный middleware (compressor_optimized.go)

```go
// Глобальный менеджер сжатия
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once

// GetCompressionManager возвращает глобальный менеджер сжатия (singleton)
func GetCompressionManager() *CompressionManager {
    compressionManagerOnce.Do(func() {
        config, err := ParseCompressionConfig()
        if err != nil {
            // Используем конфигурацию по умолчанию при ошибке
            config = &CompressionConfig{
                EnableCompression: true,
                CompressionLevel:  6,
                MinContentSize:    1024,
            }
        }
        globalCompressionManager = NewCompressionManager(config)
    })
    return globalCompressionManager
}

// OptimizedGzipMiddleware middleware с поддержкой пулов
func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        manager := GetCompressionManager()
        
        if !manager.IsCompressionEnabled() {
            h.ServeHTTP(w, r)
            return
        }
        
        ow := w
        
        // Проверяем поддержку gzip клиентом
        acceptEncoding := r.Header.Get("Accept-Encoding")
        supportsGzip := strings.Contains(acceptEncoding, "gzip")
        
        if supportsGzip {
            cw := NewOptimizedCompressWriter(w, manager)
            ow = cw
            defer cw.Close()
        }
        
        // Проверяем сжатие тела запроса
        contentEncoding := r.Header.Get("Content-Encoding")
        sendsGzip := strings.Contains(contentEncoding, "gzip")
        
        if sendsGzip {
            cr, err := NewOptimizedCompressReader(r.Body, manager.GetReaderPool())
            if err != nil {
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
            r.Body = cr
            defer cr.Close()
        }
        
        h.ServeHTTP(ow, r)
    }
}
```

## Интеграция

### Замена в main.go

```go
// Заменить:
handler.GzipMiddleware(handlerv.UpdateJSON)

// На:
handler.OptimizedGzipMiddleware(handlerv.UpdateJSON)
```

### Обратная совместимость

Старая реализация остается в `compressor.go` для возможности отката.

## Тестирование

### Юнит-тесты для пулов

```go
func TestWriterPool(t *testing.T) {
    pool := NewWriterPool()
    buf := &bytes.Buffer{}
    
    // Получаем writer
    zw1 := pool.Get(buf)
    zw1.Write([]byte("test"))
    zw1.Close()
    
    // Возвращаем в пул
    pool.Put(zw1)
    
    // Получаем снова (должен быть тот же объект)
    zw2 := pool.Get(buf)
    assert.Equal(t, zw1, zw2)
}
```

### Бенчмарки

```go
func BenchmarkCompression(b *testing.B) {
    data := make([]byte, 1024)
    rand.Read(data)
    
    b.Run("Original", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            buf := &bytes.Buffer{}
            zw := gzip.NewWriter(buf)
            zw.Write(data)
            zw.Close()
        }
    })
    
    b.Run("WithPool", func(b *testing.B) {
        pool := NewWriterPool()
        for i := 0; i < b.N; i++ {
            buf := &bytes.Buffer{}
            zw := pool.Get(buf)
            zw.Write(data)
            zw.Close()
            pool.Put(zw)
        }
    })
}
```

## Метрики и мониторинг

### Метрики для сбора

1. **Pool Hit Rate**: `(pool.Gets - pool.Puts) / pool.Gets`
2. **Pool Size**: Текущий размер пула
3. **Compression Ratio**: `original_size / compressed_size`
4. **Compression Time**: Время на сжатие
5. **Memory Allocations**: Количество аллокаций

### Интеграция с метриками

```go
type MetricsCollector struct {
    poolHits   prometheus.Counter
    poolMisses prometheus.Counter
    compTime   prometheus.Histogram
}

func (p *WriterPool) GetWithMetrics(w io.Writer) *gzip.Writer {
    // Сбор метрик
    zw := p.pool.Get().(*gzip.Writer)
    zw.Reset(w)
    return zw
}
```

## План развертывания

1. **Фаза 1**: Реализация пулов и базовой инфраструктуры
2. **Фаза 2**: Создание оптимизированного middleware
3. **Фаза 3**: Тестирование и бенчмаркинг
4. **Фаза 4**: Постепенное внедрение через feature flag
5. **Фаза 5**: Полное переключение и мониторинг

## Ожидаемые результаты

- Снижение аллокаций памяти на 70-80%
- Уменьшение времени ответа на 10-20%
- Снижение давления на GC
- Увеличение пропускной способности на 15-25%