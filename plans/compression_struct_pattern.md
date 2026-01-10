# Рефакторинг OptimizedGzipMiddleware в структуру с методом

## Анализ существующих паттернов

Посмотрев на существующий код, я вижу, что у вас уже есть отличные примеры:

### 1. Logger (структура + метод)
```go
type Logger struct {
    logger *zap.Logger
    sugar  *zap.SugaredLogger
}

func (l Logger) WithLogging(h http.HandlerFunc) http.HandlerFunc {
    // логика middleware
}
```

### 2. Hasher (структура + метод)
```go
type hasher struct {
    hashKey      string
    hashKeyBytes []byte
}

func (hs *hasher) HashMiddleware(h http.HandlerFunc) http.HandlerFunc {
    // логика middleware
}
```

### 3. Auditor (структура + метод)
```go
type Auditor struct {
    observers []audit.Observer
}

func (a *Auditor) WithAudit(h http.HandlerFunc) http.HandlerFunc {
    // логика middleware
}
```

## Предлагаемый рефакторинг для Compression

### Шаг 1: Создаём структуру Compressor

```go
type Compressor struct {
    manager *CompressionManager
}

func NewCompressor(manager *CompressionManager) *Compressor {
    return &Compressor{
        manager: manager,
    }
}
```

### Шаг 2: Переносим логику в метод

```go
func (c *Compressor) OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !c.manager.IsCompressionEnabled() {
            h.ServeHTTP(w, r)
            return
        }

        ow := w

        // Проверяем поддержку gzip клиентом
        acceptEncoding := r.Header.Get("Accept-Encoding")
        supportsGzip := strings.Contains(acceptEncoding, "gzip")

        if supportsGzip {
            cw := NewOptimizedCompressWriter(w, c.manager)
            ow = cw
            defer cw.Close()
        }

        // Проверяем сжатие тела запроса
        contentEncoding := r.Header.Get("Content-Encoding")
        sendsGzip := strings.Contains(contentEncoding, "gzip")

        if sendsGzip {
            cr, err := NewOptimizedCompressReader(r.Body, c.manager.GetReaderPool())
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

### Шаг 3: Изменения в main.go

```go
func main() {
    // ... существующий код ...

    // Создаём менеджер сжатия
    compressionConfig, err := handler.ParseCompressionConfig()
    if err != nil {
        compressionConfig = &handler.CompressionConfig{
            EnableCompression: true,
            CompressionLevel:  6,
            MinContentSize:    1024,
        }
    }
    compressionManager := handler.NewCompressionManager(compressionConfig)
    
    // Создаём компрессор
    compressor := handler.NewCompressor(compressionManager)

    // ... существующий код ...

    // Применяем middleware (теперь как у других)
    r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(compressor.OptimizedGzipMiddleware(handlerv.UpdateJSON)))))
    r.Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(compressor.OptimizedGzipMiddleware(handlerv.UpdateJSON)))))
    r.Post(`/updates`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(handlerv.UpdatesJSON)))))
    r.Post(`/updates/`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(handlerv.UpdatesJSON)))))

    r.Post(`/value/`, hasher.HashMiddleware(hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetValueJSON))))
    r.Post(`/value`, hasher.HashMiddleware(hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetValueJSON))))
    r.Get(`/`, hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetMain)))
}
```

### Шаг 4: Удаляем глобальные переменные

```go
// УДАЛИТЬ из compressor_optimized.go:
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once

func GetCompressionManager() *CompressionManager {
    // Вся эта функция больше не нужна
}
```

## Преимущества этого подхода

### 1. Единообразие с остальным кодом
Теперь все middleware следуют одному паттерну:
- `hlog.WithLogging()`
- `hasher.HashMiddleware()`
- `auditor.WithAudit()`
- `compressor.OptimizedGzipMiddleware()`

### 2. Чёткое разделение ответственности
```go
type Compressor struct {
    manager *CompressionManager  // Только управление сжатием
}
```

### 3. Лёгкость тестирования
```go
func TestCompressor() {
    mockManager := &MockCompressionManager{}
    compressor := NewCompressor(mockManager)
    
    middleware := compressor.OptimizedGzipMiddleware(testHandler)
    // Тестируем middleware
}
```

### 4. Инкапсуляция зависимостей
CompressionManager инкапсулирован внутри Compressor, как и должно быть.

### 5. Гибкость
Можно легко добавить дополнительные методы в Compressor:
```go
func (c *Compressor) CompressOnly(h http.HandlerFunc) http.HandlerFunc {
    // Только сжатие ответа
}

func (c *Compressor) DecompressOnly(h http.HandlerFunc) http.HandlerFunc {
    // Только декомпрессия запроса
}
```

## Сравнение с текущим кодом

### Было (плохо):
```go
// Глобальные переменные
var globalCompressionManager *CompressionManager

// Функция-синглтон
func GetCompressionManager() *CompressionManager { ... }

// Функция-middleware
func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
    manager := GetCompressionManager()  // Скрытая зависимость
    // ...
}
```

### Стало (хорошо):
```go
// Структура с зависимостью
type Compressor struct {
    manager *CompressionManager
}

// Конструктор с DI
func NewCompressor(manager *CompressionManager) *Compressor { ... }

// Метод-middleware
func (c *Compressor) OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
    // Явное использование c.manager
    // ...
}
```

## Итог

Этот подход полностью соответствует существующей архитектуре вашего проекта, устраняет глобальные переменные и делает код более тестируемым и поддерживаемым. Паттерн "структура + метод" уже успешно используется у вас для Logger, Hasher и Auditor.