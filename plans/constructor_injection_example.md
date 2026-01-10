# Constructor Injection для CompressionManager - Подробный пример

## Что такое Constructor Injection?

Constructor Injection - это когда зависимости передаются в объект через его конструктор (функцию создания) и сохраняются в полях структуры для дальнейшего использования.

## Текущая структура Handler

Сейчас в [`internal/handler/handlers.go`](internal/handler/handlers.go:24-29):
```go
type Handler struct {
	repo repository.Repository

	//todo переделать!!!
	dbConnectionString string
}

func NewHandler(r repository.Repository) *Handler {
	return &Handler{
		repo: r,
	}
}
```

## Вариант с Constructor Injection для CompressionManager

### Шаг 1: Изменяем структуру Handler

```go
type Handler struct {
	repo               repository.Repository
	compressionManager *CompressionManager  // Добавляем поле
	dbConnectionString string
}
```

### Шаг 2: Изменяем конструктор NewHandler

```go
func NewHandler(r repository.Repository, cm *CompressionManager) *Handler {
	return &Handler{
		repo:               r,
		compressionManager: cm,  // Сохраняем зависимость
	}
}
```

### Шаг 3: Добавляем метод для получения middleware

```go
// GzipMiddleware возвращает middleware с использованием CompressionManager из Handler
func (h *Handler) GzipMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Используем compressionManager из Handler
			if !h.compressionManager.IsCompressionEnabled() {
				handler.ServeHTTP(w, r)
				return
			}

			ow := w

			// Проверяем поддержку gzip клиентом
			acceptEncoding := r.Header.Get("Accept-Encoding")
			supportsGzip := strings.Contains(acceptEncoding, "gzip")

			if supportsGzip {
				cw := NewOptimizedCompressWriter(w, h.compressionManager)
				ow = cw
				defer cw.Close()
			}

			// Проверяем сжатие тела запроса
			contentEncoding := r.Header.Get("Content-Encoding")
			sendsGzip := strings.Contains(contentEncoding, "gzip")

			if sendsGzip {
				cr, err := NewOptimizedCompressReader(r.Body, h.compressionManager.GetReaderPool())
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				r.Body = cr
				defer cr.Close()
			}

			handler.ServeHTTP(ow, r)
		}
	}
}
```

### Шаг 4: Изменения в main.go

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

    // Создаём Handler с CompressionManager через constructor injection
    handlerv = handler.NewHandler(storage, compressionManager)

    // ... остальной код ...

    // Используем middleware из Handler
    r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(handlerv.GzipMiddleware()(handlerv.UpdateJSON)))))
    r.Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(handlerv.GzipMiddleware()(handlerv.UpdateJSON)))))
    r.Post(`/updates`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(handlerv.GzipMiddleware()(handlerv.UpdatesJSON)))))
    r.Post(`/updates/`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(handlerv.GzipMiddleware()(handlerv.UpdatesJSON)))))

    r.Post(`/value/`, hasher.HashMiddleware(hlog.WithLogging(handlerv.GzipMiddleware()(handlerv.GetValueJSON))))
    r.Post(`/value`, hasher.HashMiddleware(hlog.WithLogging(handlerv.GzipMiddleware()(handlerv.GetValueJSON))))
    r.Get(`/`, hlog.WithLogging(handlerv.GzipMiddleware()(handlerv.GetMain)))
}
```

### Шаг 5: Удаляем глобальные переменные

В [`internal/handler/compressor_optimized.go`](internal/handler/compressor_optimized.go) удаляем:
```go
// УДАЛИТЬ ЭТИ СТРОКИ:
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once

func GetCompressionManager() *CompressionManager {
    // Вся эта функция больше не нужна
}
```

## Преимущества Constructor Injection в этом случае

### 1. Все зависимости в одном месте
```go
type Handler struct {
    repo               repository.Repository
    compressionManager *CompressionManager
    // Другие зависимости...
}
```

### 2. Легко тестировать
```go
func TestHandler() {
    // Создаём моки для всех зависимостей
    mockRepo := &MockRepository{}
    mockCompressionManager := &MockCompressionManager{}
    
    // Создаём Handler с моками
    handler := NewHandler(mockRepo, mockCompressionManager)
    
    // Тестируем
    middleware := handler.GzipMiddleware()
    // ...
}
```

### 3. Явные зависимости
```go
// Сразу видно, что Handler зависит от CompressionManager
func NewHandler(r repository.Repository, cm *CompressionManager) *Handler
```

### 4. Инкапсуляция логики
```go
// Вся логика сжатия инкапсулирована в Handler
func (h *Handler) GzipMiddleware() func(http.HandlerFunc) http.HandlerFunc
```

## Сравнение с другими вариантами

### Вариант 1 (параметризованный middleware):
```go
// Нужно передавать manager каждый раз
OptimizedGzipMiddleware(compressionManager)(handler.UpdateJSON)
```

### Constructor Injection:
```go
// Manager уже внутри Handler
handlerv.GzipMiddleware()(handlerv.UpdateJSON)
```

## Потенциальные проблемы

1. **Больше ответственности у Handler** - теперь он отвечает и за сжатие
2. **Жёсткая связь** - Handler всегда зависит от CompressionManager
3. **Сложнее изменить зависимость** - нужно создавать новый Handler

## Альтернатива: Композиция

Если не хочется нагружать Handler, можно создать отдельную структуру:

```go
type ServerComponents struct {
    handler            *Handler
    compressionManager *CompressionManager
    logger             *Logger
    auditor            *Auditor
}

func NewServerComponents(storage repository.Repository) *ServerComponents {
    compressionManager := NewCompressionManager(config)
    handler := NewHandler(storage, compressionManager)
    
    return &ServerComponents{
        handler:            handler,
        compressionManager: compressionManager,
    }
}

func (sc *ServerComponents) SetupRoutes(r chi.Router) {
    r.Post("/update", sc.handler.GzipMiddleware()(sc.handler.UpdateJSON))
}
```

Constructor Injection - это хороший вариант, если CompressionManager логически связан с Handler и используется во многих местах Handler.