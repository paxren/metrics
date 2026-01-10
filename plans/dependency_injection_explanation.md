# Dependency Injection (Внедрение зависимостей) - Объяснение

## Что такое Dependency Injection?

**Dependency Injection (DI)** - это паттерн проектирования, при котором зависимости объекта передаются ему извне, а не создаются внутри самого объекта.

## Простой пример

### Плохой подход (без DI):
```go
type Car struct {
    engine Engine
}

func (c *Car) Start() {
    // Двигатель создаётся внутри объекта
    c.engine = NewEngine() 
    c.engine.Start()
}
```

### Хороший подход (с DI):
```go
type Car struct {
    engine Engine
}

// Двигатель передаётся извне через конструктор
func NewCar(engine Engine) *Car {
    return &Car{
        engine: engine,
    }
}

func (c *Car) Start() {
    c.engine.Start()
}
```

## В вашем проекте: Singleton vs DI

### Текущий код (Singleton):
```go
// Глобальная переменная - BAD PRACTICE
var globalCompressionManager *CompressionManager

func GetCompressionManager() *CompressionManager {
    // Создаётся глобальный экземпляр
    if globalCompressionManager == nil {
        globalCompressionManager = NewCompressionManager(config)
    }
    return globalCompressionManager
}

func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Используем глобальную переменную - СКРЫТАЯ ЗАВИСИМОСТЬ
        manager := GetCompressionManager()
        // ...
    }
}
```

### Как будет с DI:
```go
// НЕТ глобальных переменных

// Middleware принимает зависимость как параметр
func OptimizedGzipMiddleware(manager *CompressionManager) func(http.HandlerFunc) http.HandlerFunc {
    return func(h http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // Используем переданный менеджер - ЯВНАЯ ЗАВИСИМОСТЬ
            if !manager.IsCompressionEnabled() {
                h.ServeHTTP(w, r)
                return
            }
            // ...
        }
    }
}

// В main.go создаём менеджер и передаём его
func main() {
    // Создаём менеджер сжатия
    config := ParseCompressionConfig()
    compressionManager := NewCompressionManager(config)
    
    // Передаём его в middleware
    middleware := OptimizedGzipMiddleware(compressionManager)
    
    // Применяем middleware
    r.Post("/update", middleware(handler.UpdateJSON))
}
```

## Почему DI лучше для тестирования?

### Тестирование с Singleton (сложно):
```go
func TestMiddleware() {
    // НЕЛЬЗЯ подменить CompressionManager на мок!
    // Всегда используется реальный глобальный экземпляр
    // Тесты зависят друг от друга
}
```

### Тестирование с DI (легко):
```go
func TestMiddleware() {
    // Создаём мок-менеджер для тестов
    mockManager := &MockCompressionManager{
        enabled: false, // Настраиваем поведение для теста
    }
    
    // Передаём мок в middleware
    middleware := OptimizedGzipMiddleware(mockManager)
    
    // Тестируем с изолированной зависимостью
    // Каждый тест имеет свой экземпляр
}
```

## Виды Dependency Injection

### 1. Constructor Injection (внедрение через конструктор) - РЕКОМЕНДУЕТСЯ
```go
func NewService(repo Repository, logger Logger) *Service {
    return &Service{
        repo:   repo,
        logger: logger,
    }
}
```

### 2. Method Injection (внедрение через метод)
```go
func (s *Service) Process(data Data, repo Repository) {
    // repo передаётся в конкретный метод
}
```

### 3. Property Injection (внедрение через поле) - НЕ РЕКОМЕНДУЕТСЯ
```go
type Service struct {
    repo Repository // Устанавливается извне
}
```

## Преимущества DI

1. **Тестируемость** - легко подменять зависимости моками
2. **Изоляция** - каждый компонент независим
3. **Гибкость** - можно менять реализации без изменения кода
4. **Читаемость** - явные зависимости видны в сигнатуре
5. **Переиспользование** - компоненты можно использовать в разных контекстах

## Недостатки DI

1. **Больше кода** - нужно передавать зависимости
2. **Сложность конфигурации** - нужно настраивать зависимости
3. **Больше файлов** - часто создаются файлы конфигурации DI

## Практический пример для вашего проекта

### Что нужно изменить:

1. **Убрать глобальные переменные:**
```go
// УДАЛИТЬ:
var globalCompressionManager *CompressionManager
var compressionManagerOnce sync.Once
```

2. **Изменить middleware:**
```go
// БЫЛО:
func OptimizedGzipMiddleware(h http.HandlerFunc) http.HandlerFunc

// СТАЛО:
func OptimizedGzipMiddleware(manager *CompressionManager) func(http.HandlerFunc) http.HandlerFunc
```

3. **Изменить main.go:**
```go
// ДОБАВИТЬ:
compressionConfig, _ := handler.ParseCompressionConfig()
compressionManager := handler.NewCompressionManager(compressionConfig)

// ИЗМЕНИТЬ:
r.Post("/update", handler.OptimizedGzipMiddleware(compressionManager)(handler.UpdateJSON))
```

## Итог

Dependency Injection - это просто передача зависимостей извне вместо создания их внутри. Это делает код более тестируемым, гибким и понятным.