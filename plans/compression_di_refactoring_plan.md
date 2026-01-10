# План рефакторинга: От глобального Singleton к Dependency Injection

## Анализ текущей архитектуры

### Текущая реализация с глобальным singleton

В файле [`internal/handler/compressor_optimized.go`](internal/handler/compressor_optimized.go:155-174) реализован глобальный singleton:

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
```

### Зависимости от globalCompressionManager

1. **Middleware `OptimizedGzipMiddleware`** (строки 176-214):
   ```go
   func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
       return func(w http.ResponseWriter, r *http.Request) {
           manager := GetCompressionManager() // Использование глобального singleton
           // ...
       }
   }
   ```

2. **Использование в main.go**:
   - Middleware применяется к нескольким эндпоинтам в [`cmd/server/main.go`](cmd/server/main.go:173-183)

## Проблемы текущего подхода

1. **Сложность тестирования** - невозможно легко подменить CompressionManager на мок
2. **Глобальное состояние** - все тесты делят один и тот же экземпляр
3. **Скрытые зависимости** - middleware зависит от глобальной переменной
4. **Жёсткая связь** - сложно изменить конфигурацию для разных окружений

## Варианты рефакторинга с использованием Dependency Injection

### Вариант 1: Параметризованный middleware (рекомендуемый)

**Изменение в compressor_optimized.go:**
```go
// OptimizedGzipMiddleware middleware с поддержкой пулов и DI
func OptimizedGzipMiddleware(manager *CompressionManager) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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
}
```

**Изменения в main.go:**
```go
// Создание менеджера сжатия
compressionConfig, err := handler.ParseCompressionConfig()
if err != nil {
    // Используем конфигурацию по умолчанию при ошибке
    compressionConfig = &handler.CompressionConfig{
        EnableCompression: true,
        CompressionLevel:  6,
        MinContentSize:    1024,
    }
}
compressionManager := handler.NewCompressionManager(compressionConfig)

// Применение middleware с DI
r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(handler.OptimizedGzipMiddleware(compressionManager)(handlerv.UpdateJSON)))))
```

### Вариант 2: Структура с зависимостями

**Создание структуры ServerDependencies:**
```go
type ServerDependencies struct {
    CompressionManager *CompressionManager
    // Другие зависимости...
}

func NewServerDependencies(config *ServerConfig) (*ServerDependencies, error) {
    compressionConfig, err := ParseCompressionConfig()
    if err != nil {
        compressionConfig = &CompressionConfig{
            EnableCompression: true,
            CompressionLevel:  6,
            MinContentSize:    1024,
        }
    }
    
    return &ServerDependencies{
        CompressionManager: NewCompressionManager(compressionConfig),
    }, nil
}
```

### Вариант 3: Интерфейс для абстракции

**Создание интерфейса:**
```go
type CompressionService interface {
    IsCompressionEnabled() bool
    ShouldCompress(contentType string, contentLength int) bool
    GetWriterPool() *WriterPool
    GetReaderPool() *ReaderPool
}

// OptimizedGzipMiddleware с интерфейсом
func OptimizedGzipMiddleware(service CompressionService) func(http.HandlerFunc) http.HandlerFunc {
    // Реализация с использованием интерфейса
}
```

## План миграции от singleton к DI

### Этап 1: Подготовка
1. Создать новую версию middleware с параметром CompressionManager
2. Оставить старую версию для обратной совместимости
3. Добавить комментарии об устаревании старой версии

### Этап 2: Обновление main.go
1. Создать экземпляр CompressionManager в main.go
2. Передать его в middleware при регистрации роутов
3. Убедиться, что все эндпоинты используют новую версию

### Этап 3: Обновление тестов
1. Модифицировать тесты для создания тестового CompressionManager
2. Использовать моки для тестирования middleware
3. Убедиться в изоляции тестов

### Этап 4: Удаление старого кода
1. Удалить глобальные переменные globalCompressionManager и compressionManagerOnce
2. Удалить функцию GetCompressionManager()
3. Удалить старую версию middleware

## Оценка влияния изменений

### Положительные эффекты
1. **Улучшение тестируемости** - возможность подмены зависимостей
2. **Чёткие зависимости** - явная передача зависимостей
3. **Гибкость конфигурации** - разные конфигурации для разных окружений
4. **Изоляция** - каждый тест работает со своим экземпляром

### Риски и сложности
1. **Изменение API** - нужно обновить все места использования middleware
2. **Обратная совместимость** - возможные проблемы при обновлении
3. **Объём изменений** - затрагивает несколько файлов

### Необходимые изменения в файлах
1. **internal/handler/compressor_optimized.go**:
   - Изменить сигнатуру OptimizedGzipMiddleware
   - Удалить глобальные переменные
   - Удалить GetCompressionManager()

2. **cmd/server/main.go**:
   - Создать CompressionManager
   - Обновить все вызовы OptimizedGzipMiddleware

3. **Тесты**:
   - Обновить тесты middleware
   - Создать тестовые реализации CompressionManager

## Рекомендации

1. **Использовать Вариант 1** как наиболее простой и эффективный
2. **Проводить миграцию поэтапно** для минимизации рисков
3. **Добавить тесты** для новой реализации перед удалением старой
4. **Сохранить обратную совместимость** на время переходного периода

Этот рефакторинг значительно улучшит архитектуру приложения и сделает код более тестируемым и поддерживаемым.