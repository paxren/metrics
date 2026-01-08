# Руководство по внедрению оптимизации сжатия

## Обзор

Это руководство описывает пошаговый процесс внедрения оптимизации сжатия с использованием пулов объектов в ваш проект.

## Предварительные требования

1. Go 1.19 или выше
2. Существующая реализация сжатия в `internal/handler/compressor.go`
3. Тестовое окружение для проверки производительности

## Шаг 1: Создание файлов

### 1.1 Создайте файл конфигурации

```bash
touch internal/handler/compression_config.go
```

Содержимое файла `compression_config.go`:
```go
package handler

import (
    "github.com/caarlos0/env/v11"
)

// CompressionConfig содержит настройки для системы сжатия
type CompressionConfig struct {
    WriterPoolSize    int `env:"COMPRESSION_WRITER_POOL_SIZE" envDefault:"100"`
    ReaderPoolSize    int `env:"COMPRESSION_READER_POOL_SIZE" envDefault:"50"`
    CompressionLevel  int `env:"COMPRESSION_LEVEL" envDefault:"6"`
    EnableCompression bool `env:"ENABLE_COMPRESSION" envDefault:"true"`
    MinContentSize    int `env:"COMPRESSION_MIN_CONTENT_SIZE" envDefault:"1024"`
}

// ParseCompressionConfig парсит конфигурацию из переменных окружения
func ParseCompressionConfig() (*CompressionConfig, error) {
    cfg := &CompressionConfig{}
    err := env.Parse(cfg)
    if err != nil {
        return nil, err
    }
    
    if cfg.CompressionLevel < 1 || cfg.CompressionLevel > 9 {
        cfg.CompressionLevel = 6
    }
    
    return cfg, nil
}
```

### 1.2 Создайте файл пулов

```bash
touch internal/handler/compression_pools.go
```

Содержимое файла `compression_pools.go`:
```go
package handler

import (
    "compress/gzip"
    "io"
    "sync"
)

// WriterPool управляет пулом gzip.Writer
type WriterPool struct {
    pool sync.Pool
}

// NewWriterPool создает новый пул для gzip.Writer
func NewWriterPool() *WriterPool {
    return &WriterPool{
        pool: sync.Pool{
            New: func() interface{} {
                return gzip.NewWriter(io.Discard)
            },
        },
    }
}

// Get получает gzip.Writer из пула
func (p *WriterPool) Get(w io.Writer) *gzip.Writer {
    zw := p.pool.Get().(*gzip.Writer)
    zw.Reset(w)
    return zw
}

// Put возвращает gzip.Writer в пул
func (p *WriterPool) Put(zw *gzip.Writer) {
    if zw != nil {
        zw.Close()
        p.pool.Put(zw)
    }
}

// ReaderPool управляет пулом gzip.Reader
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

// Get получает gzip.Reader из пула
func (p *ReaderPool) Get(r io.Reader) (*gzip.Reader, error) {
    zr := p.pool.Get().(*gzip.Reader)
    err := zr.Reset(r)
    if err != nil {
        p.pool.Put(zr)
        return nil, err
    }
    return zr, nil
}

// Put возвращает gzip.Reader в пул
func (p *ReaderPool) Put(zr *gzip.Reader) {
    if zr != nil {
        zr.Close()
        p.pool.Put(zr)
    }
}

// CompressionManager управляет пулами и конфигурацией
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

// ShouldCompress проверяет, нужно ли сжимать контент
func (cm *CompressionManager) ShouldCompress(contentType string, contentLength int) bool {
    if !cm.config.EnableCompression {
        return false
    }
    
    if contentLength > 0 && contentLength < cm.config.MinContentSize {
        return false
    }
    
    return contentType == "application/json" || 
           contentType == "text/html" || 
           contentType == "text/plain" ||
           contentType == "text/css" ||
           contentType == "application/javascript"
}
```

### 1.3 Создайте оптимизированный компрессор

```bash
touch internal/handler/compressor_optimized.go
```

Содержимое файла `compressor_optimized.go`:
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
    mu              sync.Mutex
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

func (c *OptimizedCompressWriter) Header() http.Header {
    return c.w.Header()
}

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
        c.w.Header().Del("Content-Length")
    }
    
    c.w.WriteHeader(statusCode)
}

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

func (c *OptimizedCompressReader) Read(p []byte) (n int, err error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.closed {
        return 0, io.EOF
    }
    
    return c.zr.Read(p)
}

func (c *OptimizedCompressReader) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.closed {
        return nil
    }
    
    c.closed = true
    
    if c.zr != nil {
        c.pool.Put(c.zr)
        c.zr = nil
    }
    
    return c.r.Close()
}

// Глобальный менеджер сжатия
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once

// GetCompressionManager возвращает глобальный менеджер сжатия
func GetCompressionManager() *CompressionManager {
    compressionManagerOnce.Do(func() {
        config, err := ParseCompressionConfig()
        if err != nil {
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
        
        if !manager.config.EnableCompression {
            h.ServeHTTP(w, r)
            return
        }
        
        ow := w
        
        acceptEncoding := r.Header.Get("Accept-Encoding")
        supportsGzip := strings.Contains(acceptEncoding, "gzip")
        
        if supportsGzip {
            cw := NewOptimizedCompressWriter(w, manager)
            ow = cw
            defer cw.Close()
        }
        
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

## Шаг 2: Создание тестов

### 2.1 Создайте файл тестов

```bash
touch internal/handler/compression_pools_test.go
```

Содержимое файла `compression_pools_test.go`:
```go
package handler

import (
    "bytes"
    "compress/gzip"
    "io"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestWriterPool(t *testing.T) {
    pool := NewWriterPool()
    buf := &bytes.Buffer{}
    
    // Получаем writer
    zw1 := pool.Get(buf)
    assert.NotNil(t, zw1)
    
    // Пишем данные
    n, err := zw1.Write([]byte("test data"))
    assert.NoError(t, err)
    assert.Equal(t, 9, n)
    
    // Возвращаем в пул
    pool.Put(zw1)
    
    // Получаем снова
    buf2 := &bytes.Buffer{}
    zw2 := pool.Get(buf2)
    assert.Equal(t, zw1, zw2) // Должен быть тот же объект
}

func TestReaderPool(t *testing.T) {
    pool := NewReaderPool()
    
    // Создаем сжатые данные
    var buf bytes.Buffer
    zw := gzip.NewWriter(&buf)
    zw.Write([]byte("test data"))
    zw.Close()
    
    // Получаем reader
    zr1, err := pool.Get(&buf)
    assert.NoError(t, err)
    assert.NotNil(t, zr1)
    
    // Читаем данные
    result := make([]byte, 100)
    n, err := zr1.Read(result)
    assert.NoError(t, err)
    assert.Equal(t, "test data", string(result[:n]))
    
    // Возвращаем в пул
    pool.Put(zr1)
    
    // Получаем снова
    buf2 := bytes.NewBuffer(buf.Bytes())
    zr2, err := pool.Get(&buf2)
    assert.NoError(t, err)
    assert.Equal(t, zr1, zr2) // Должен быть тот же объект
}

func TestCompressionManager(t *testing.T) {
    config := &CompressionConfig{
        EnableCompression: true,
        MinContentSize:    100,
    }
    manager := NewCompressionManager(config)
    
    // Тест ShouldCompress
    assert.True(t, manager.ShouldCompress("application/json", 200))
    assert.False(t, manager.ShouldCompress("application/json", 50)) // Слишком маленький
    assert.False(t, manager.ShouldCompress("image/png", 200))      // Неподдерживаемый тип
    
    // Тест получения пулов
    assert.NotNil(t, manager.GetWriterPool())
    assert.NotNil(t, manager.GetReaderPool())
}

func BenchmarkCompressionWithPool(b *testing.B) {
    pool := NewWriterPool()
    data := make([]byte, 1024)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf := &bytes.Buffer{}
        zw := pool.Get(buf)
        zw.Write(data)
        zw.Close()
        pool.Put(zw)
    }
}

func BenchmarkCompressionWithoutPool(b *testing.B) {
    data := make([]byte, 1024)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        buf := &bytes.Buffer{}
        zw := gzip.NewWriter(buf)
        zw.Write(data)
        zw.Close()
    }
}
```

## Шаг 3: Интеграция в основной код

### 3.1 Обновление main.go

Замените все вызовы `handler.GzipMiddleware` на `handler.OptimizedGzipMiddleware`:

```go
// Было:
r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(handler.GzipMiddleware(handlerv.UpdateJSON)))))

// Стало:
r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(handler.OptimizedGzipMiddleware(handlerv.UpdateJSON)))))
```

### 3.2 Добавление переменных окружения

Добавьте в `.env` файл или установите переменные окружения:

```bash
# Включение сжатия
ENABLE_COMPRESSION=true

# Минимальный размер контента для сжатия (байты)
COMPRESSION_MIN_CONTENT_SIZE=1024

# Уровень сжатия (1-9)
COMPRESSION_LEVEL=6
```

## Шаг 4: Тестирование

### 4.1 Запуск тестов

```bash
# Запуск всех тестов
go test ./internal/handler/

# Запуск бенчмарков
go test -bench=. ./internal/handler/

# Запуск с покрытием
go test -cover ./internal/handler/
```

### 4.2 Проверка производительности

1. Запустите сервер с новой реализацией
2. Создайте нагрузку с помощью агента
3. Соберите профили:
   ```bash
   go tool pprof -output cpu_optimized.prof http://localhost:8080/debug/pprof/profile?seconds=30
   go tool pprof -output heap_optimized.prof http://localhost:8080/debug/pprof/heap
   ```
4. Сравните с предыдущими профилями

## Шаг 5: Мониторинг

### 5.1 Ключевые метрики

- Аллокации памяти (должны уменьшиться на 70-80%)
- Время ответа (должно уменьшиться на 10-20%)
- Пропускная способность (должна увеличиться на 15-25%)

### 5.2 Проверка в продакшене

1. Внедрите через feature flag
2. Мониторьте метрики в течение недели
3. Постепенно увеличивайте процент трафика

## Шаг 6: Откат (если необходимо)

Если возникнут проблемы, можно быстро откатиться:

1. Замените `OptimizedGzipMiddleware` обратно на `GzipMiddleware`
2. Перезапустите сервис
3. Проанализируйте проблемы

## Ожидаемые результаты

После внедрения оптимизации вы должны увидеть:

- **Снижение аллокаций памяти**: с ~3GB до ~600-900MB
- **Уменьшение использования CPU**: с 1.40% до 0.8-1.0%
- **Улучшение времени ответа**: на 10-20%
- **Увеличение пропускной способности**: на 15-25%

## Частые проблемы и решения

### Проблема: Утечки памяти

**Решение**: Убедитесь, что все объекты возвращаются в пул через defer

### Проблема: Состояние гонки

**Решение**: Используйте мьютексы в OptimizedCompressWriter/Reader

### Проблема: Неправильная реинициализация

**Решение**: Всегда вызывайте Reset() для gzip.Writer/Reader

### Проблема: Сжатие не работает

**Решение**: Проверьте переменные окружения и заголовки запросов