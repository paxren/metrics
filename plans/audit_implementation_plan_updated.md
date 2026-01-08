# План реализации функциональности аудита запросов с использованием паттерна "Наблюдатель"

## Анализ требований

### Задача:
Реализовать функциональность аудита запросов в сервисе сбора метрик с использованием паттерна "Наблюдатель".

### Требования:
1. Добавить параметры конфигурации:
   - `--audit-file` / `AUDIT_FILE` - путь к файлу для логов аудита
   - `--audit-url` / `AUDIT_URL` - URL для отправки логов аудита

2. Формат события аудита:
```json
{
  "ts": 12345678, // unix timestamp события
  "metrics": ["Alloc", "Frees", ...], // наименование полученных метрик
  "ip_address": "192.168.0.42" // IP адрес входящего запроса
}
```

3. Аудит должен выполняться после успешной обработки метрик

## Архитектурное решение

### Компоненты:
1. **AuditEvent** - модель события аудита
2. **AuditObserver** - интерфейс наблюдателя
3. **FileAuditObserver** - наблюдатель для записи в файл
4. **URLAuditObserver** - наблюдатель для отправки на удалённый сервер
5. **AuditMiddleware** - единый middleware для аудита запросов
6. **Расширение конфигурации сервера** - добавление новых параметров

### Схема взаимодействия:
```
HTTP-запрос → AuditMiddleware → Обработчик → Ответ
                     ↓
              Извлечение метрик из JSON
                     ↓
              Восстановление тела запроса
                     ↓
              Выполнение основного обработчика
                     ↓
              Проверка статуса ответа
                     ↓
              Создание AuditEvent
                     ↓
              Уведомление наблюдателей
                     ↓
        FileAuditObserver → Запись в файл
        URLAuditObserver → POST-запрос на URL
```

## Детальная реализация

### 1. Расширение конфигурации сервера

**Файл:** `internal/config/server.go`

Добавить в структуры `ServerConfigEnv` и `ServerConfig` новые поля:
```go
type ServerConfigEnv struct {
    // существующие поля...
    AuditFile string `env:"AUDIT_FILE"`
    AuditURL  string `env:"AUDIT_URL"`
}

type ServerConfig struct {
    // существующие поля...
    AuditFile string
    AuditURL  string
    
    // параметры для флагов
    paramAuditFile string
    paramAuditURL  string
}
```

Добавить инициализацию флагов в методе `Init()` и обработку в методе `Parse()`.

### 2. Модель события аудита

**Файл:** `internal/models/audit.go`

```go
package models

import "time"

type AuditEvent struct {
    TS        int64    `json:"ts"`
    Metrics   []string `json:"metrics"`
    IPAddress string   `json:"ip_address"`
}

func NewAuditEvent(metrics []string, ipAddress string) *AuditEvent {
    return &AuditEvent{
        TS:        time.Now().Unix(),
        Metrics:   metrics,
        IPAddress: ipAddress,
    }
}
```

### 3. Интерфейс наблюдателя

**Файл:** `internal/audit/observer.go`

```go
package audit

import "github.com/paxren/metrics/internal/models"

type Observer interface {
    Notify(event *models.AuditEvent) error
}
```

### 4. Наблюдатель для файлового аудита

**Файл:** `internal/audit/file_observer.go`

```go
package audit

import (
    "encoding/json"
    "os"
    "sync"

    "github.com/paxren/metrics/internal/models"
)

type FileObserver struct {
    filePath string
    mutex    sync.Mutex
}

func NewFileObserver(filePath string) *FileObserver {
    return &FileObserver{
        filePath: filePath,
    }
}

func (f *FileObserver) Notify(event *models.AuditEvent) error {
    f.mutex.Lock()
    defer f.mutex.Unlock()
    
    file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()
    
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    _, err = file.Write(append(data, '\n'))
    return err
}
```

### 5. Наблюдатель для сетевого аудита

**Файл:** `internal/audit/url_observer.go`

```go
package audit

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"

    "github.com/paxren/metrics/internal/models"
)

type URLObserver struct {
    url    string
    client *http.Client
}

func NewURLObserver(url string) *URLObserver {
    return &URLObserver{
        url: url,
        client: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

func (u *URLObserver) Notify(event *models.AuditEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    resp, err := u.client.Post(u.url, "application/json", bytes.NewBuffer(data))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

### 6. Middleware для аудита

**Файл:** `internal/handler/audit_middleware.go`

```go
package handler

import (
    "bytes"
    "encoding/json"
    "io"
    "net"
    "net/http"
    "strings"

    "github.com/paxren/metrics/internal/audit"
    "github.com/paxren/metrics/internal/models"
)

// responseWriter - обёртка для отслеживания статуса ответа
type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

// AuditMiddleware создаёт middleware для аудита запросов
func AuditMiddleware(observers []audit.Observer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Извлекаем метрики из JSON-запроса, если применимо
            var metrics []string
            var bodyBytes []byte
            
            // Сохраняем тело запроса для последующего восстановления
            if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
                var err error
                bodyBytes, err = io.ReadAll(r.Body)
                if err == nil {
                    // Восстанавливаем тело запроса для следующих обработчиков
                    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
                    
                    // Извлекаем метрики из тела запроса
                    metrics = extractMetricsFromJSON(bodyBytes, r.URL.Path)
                }
            }
            
            // Создаём обёртку для ResponseWriter
            wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
            
            // Выполняем основной обработчик
            next.ServeHTTP(wrapped, r)
            
            // Если запрос успешный (статус 2xx) и есть метрики для аудита
            if wrapped.status >= 200 && wrapped.status < 300 && len(metrics) > 0 {
                // Создаём событие аудита
                event := models.NewAuditEvent(metrics, getIPFromRequest(r))
                
                // Уведомляем наблюдателей
                for _, observer := range observers {
                    observer.Notify(event) // Игнорируем ошибки, чтобы не прерывать обработку
                }
            }
        })
    }
}

// extractMetricsFromJSON извлекает названия метрик из JSON-тела запроса
func extractMetricsFromJSON(bodyBytes []byte, path string) []string {
    var metrics []string
    
    // Для одиночной метрики (/update, /update/)
    if strings.HasSuffix(path, "/update") || strings.HasSuffix(path, "/update/") {
        var metric models.Metrics
        if err := json.Unmarshal(bodyBytes, &metric); err == nil && metric.ID != "" {
            metrics = append(metrics, metric.ID)
        }
    }
    
    // Для пакета метрик (/updates, /updates/)
    if strings.HasSuffix(path, "/updates") || strings.HasSuffix(path, "/updates/") {
        var metricModels []models.Metrics
        if err := json.Unmarshal(bodyBytes, &metricModels); err == nil {
            for _, m := range metricModels {
                if m.ID != "" {
                    metrics = append(metrics, m.ID)
                }
            }
        }
    }
    
    return metrics
}

// getIPFromRequest извлекает IP-адрес из запроса
func getIPFromRequest(r *http.Request) string {
    // Проверяем заголовки для проксированных запросов
    if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
        return strings.Split(ip, ",")[0]
    }
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return ip
    }
    
    // Извлекаем из RemoteAddr
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return ip
}
```

### 7. Интеграция в основной сервер

**Файл:** `cmd/server/main.go`

Добавить создание наблюдателей и интеграцию middleware:

```go
// После инициализации конфигурации
var auditObservers []audit.Observer

// Создаём наблюдателя для файла, если указан путь
if serverConfig.AuditFile != "" {
    fileObserver := audit.NewFileObserver(serverConfig.AuditFile)
    auditObservers = append(auditObservers, fileObserver)
}

// Создаём наблюдателя для URL, если указан URL
if serverConfig.AuditURL != "" {
    urlObserver := audit.NewURLObserver(serverConfig.AuditURL)
    auditObservers = append(auditObservers, urlObserver)
}

// Применяем middleware только к эндпоинтам обновления метрик
if len(auditObservers) > 0 {
    r.With(
        handler.AuditMiddleware(auditObservers),
    ).Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
    
    r.With(
        handler.AuditMiddleware(auditObservers),
    ).Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
    
    r.With(
        handler.AuditMiddleware(auditObservers),
    ).Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
    
    r.With(
        handler.AuditMiddleware(auditObservers),
    ).Post(`/updates`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON))))
    
    r.With(
        handler.AuditMiddleware(auditObservers),
    ).Post(`/updates/`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON))))
} else {
    // Если аудит отключён, используем стандартные роуты
    r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
    r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
    r.Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
    r.Post(`/updates`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON))))
    r.Post(`/updates/`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON))))
}
```

### 8. Тестирование middleware

**Файл:** `internal/handler/audit_middleware_test.go`

```go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/paxren/metrics/internal/audit"
    "github.com/paxren/metrics/internal/models"
)

// mockObserver - мок наблюдателя для тестирования
type mockObserver struct {
    events []*models.AuditEvent
}

func (m *mockObserver) Notify(event *models.AuditEvent) error {
    m.events = append(m.events, event)
    return nil
}

func TestAuditMiddleware(t *testing.T) {
    // Создаём мок наблюдателя
    observer := &mockObserver{}
    observers := []audit.Observer{observer}
    
    // Создаём middleware
    middleware := AuditMiddleware(observers)
    
    // Создаём тестовый обработчик, который всегда возвращает статус 200
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    // Создаём тестовый запрос с JSON-телом
    metric := models.Metrics{
        ID:    "test_metric",
        MType: "gauge",
        Value: func() *float64 { v := 42.5; return &v }(),
    }
    
    body, _ := json.Marshal(metric)
    req := httptest.NewRequest("POST", "/update", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Создаём ResponseRecorder
    rr := httptest.NewRecorder()
    
    // Применяем middleware к тестовому обработчику
    handler := middleware(testHandler)
    
    // Выполняем запрос
    handler.ServeHTTP(rr, req)
    
    // Проверяем статус ответа
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
    }
    
    // Проверяем, что наблюдатель получил событие
    if len(observer.events) != 1 {
        t.Fatalf("Expected 1 event, got %d", len(observer.events))
    }
    
    event := observer.events[0]
    if len(event.Metrics) != 1 || event.Metrics[0] != "test_metric" {
        t.Errorf("Expected metrics [test_metric], got %v", event.Metrics)
    }
    
    if event.IPAddress == "" {
        t.Error("Expected IP address to be set")
    }
    
    if event.TS == 0 {
        t.Error("Expected timestamp to be set")
    }
}

func TestAuditMiddleware_WithMultipleMetrics(t *testing.T) {
    // Создаём мок наблюдателя
    observer := &mockObserver{}
    observers := []audit.Observer{observer}
    
    // Создаём middleware
    middleware := AuditMiddleware(observers)
    
    // Создаём тестовый обработчик, который всегда возвращает статус 200
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    // Создаём тестовый запрос с пакетом метрик
    metrics := []models.Metrics{
        {ID: "metric1", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()},
        {ID: "metric2", MType: "counter", Delta: func() *int64 { v := int64(10); return &v }()},
        {ID: "metric3", MType: "gauge", Value: func() *float64 { v := 3.14; return &v }()},
    }
    
    body, _ := json.Marshal(metrics)
    req := httptest.NewRequest("POST", "/updates", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Создаём ResponseRecorder
    rr := httptest.NewRecorder()
    
    // Применяем middleware к тестовому обработчику
    handler := middleware(testHandler)
    
    // Выполняем запрос
    handler.ServeHTTP(rr, req)
    
    // Проверяем статус ответа
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
    }
    
    // Проверяем, что наблюдатель получил событие
    if len(observer.events) != 1 {
        t.Fatalf("Expected 1 event, got %d", len(observer.events))
    }
    
    event := observer.events[0]
    expectedMetrics := []string{"metric1", "metric2", "metric3"}
    if len(event.Metrics) != len(expectedMetrics) {
        t.Errorf("Expected %d metrics, got %d", len(expectedMetrics), len(event.Metrics))
    }
    
    for i, expected := range expectedMetrics {
        if i >= len(event.Metrics) || event.Metrics[i] != expected {
            t.Errorf("Expected metric %s at index %d, got %s", expected, i, event.Metrics[i])
        }
    }
}

func TestAuditMiddleware_WithNonJSONRequest(t *testing.T) {
    // Создаём мок наблюдателя
    observer := &mockObserver{}
    observers := []audit.Observer{observer}
    
    // Создаём middleware
    middleware := AuditMiddleware(observers)
    
    // Создаём тестовый обработчик, который всегда возвращает статус 200
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    // Создаём тестовый запрос без JSON-тела
    req := httptest.NewRequest("POST", "/update/gauge/test_metric/42.5", nil)
    
    // Создаём ResponseRecorder
    rr := httptest.NewRecorder()
    
    // Применяем middleware к тестовому обработчику
    handler := middleware(testHandler)
    
    // Выполняем запрос
    handler.ServeHTTP(rr, req)
    
    // Проверяем статус ответа
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
    }
    
    // Проверяем, что наблюдатель не получил событие (не JSON-запрос)
    if len(observer.events) != 0 {
        t.Errorf("Expected 0 events for non-JSON request, got %d", len(observer.events))
    }
}

func TestAuditMiddleware_WithErrorStatus(t *testing.T) {
    // Создаём мок наблюдателя
    observer := &mockObserver{}
    observers := []audit.Observer{observer}
    
    // Создаём middleware
    middleware := AuditMiddleware(observers)
    
    // Создаём тестовый обработчик, который возвращает ошибку
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusBadRequest)
    })
    
    // Создаём тестовый запрос с JSON-телом
    metric := models.Metrics{
        ID:    "test_metric",
        MType: "gauge",
        Value: func() *float64 { v := 42.5; return &v }(),
    }
    
    body, _ := json.Marshal(metric)
    req := httptest.NewRequest("POST", "/update", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Создаём ResponseRecorder
    rr := httptest.NewRecorder()
    
    // Применяем middleware к тестовому обработчику
    handler := middleware(testHandler)
    
    // Выполняем запрос
    handler.ServeHTTP(rr, req)
    
    // Проверяем статус ответа
    if status := rr.Code; status != http.StatusBadRequest {
        t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, status)
    }
    
    // Проверяем, что наблюдатель не получил событие (статус не 2xx)
    if len(observer.events) != 0 {
        t.Errorf("Expected 0 events for error status, got %d", len(observer.events))
    }
}
```

### 9. Тестирование файлового наблюдателя

**Файл:** `internal/audit/file_observer_test.go`

```go
package audit

import (
    "encoding/json"
    "os"
    "testing"

    "github.com/paxren/metrics/internal/models"
)

func TestFileObserver_Notify(t *testing.T) {
    // Создаём временный файл
    tmpFile, err := os.CreateTemp("", "audit_test_*.log")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpFile.Name())
    tmpFile.Close()
    
    // Создаём наблюдателя
    observer := NewFileObserver(tmpFile.Name())
    
    // Создаём тестовое событие
    event := &models.AuditEvent{
        TS:        1234567890,
        Metrics:   []string{"Alloc", "Frees"},
        IPAddress: "192.168.0.42",
    }
    
    // Уведомляем наблюдателя
    err = observer.Notify(event)
    if err != nil {
        t.Fatalf("Failed to notify observer: %v", err)
    }
    
    // Проверяем содержимое файла
    data, err := os.ReadFile(tmpFile.Name())
    if err != nil {
        t.Fatalf("Failed to read file: %v", err)
    }
    
    var savedEvent models.AuditEvent
    err = json.Unmarshal(data[:len(data)-1], &savedEvent) // Убираем последний символ \n
    if err != nil {
        t.Fatalf("Failed to unmarshal event: %v", err)
    }
    
    if savedEvent.TS != event.TS {
        t.Errorf("Expected TS %d, got %d", event.TS, savedEvent.TS)
    }
    
    if len(savedEvent.Metrics) != len(event.Metrics) {
        t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(savedEvent.Metrics))
    }
    
    if savedEvent.IPAddress != event.IPAddress {
        t.Errorf("Expected IP %s, got %s", event.IPAddress, savedEvent.IPAddress)
    }
}
```

### 10. Тестирование URL наблюдателя

**Файл:** `internal/audit/url_observer_test.go`

```go
package audit

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/paxren/metrics/internal/models"
)

func TestURLObserver_Notify(t *testing.T) {
    // Создаём тестовый сервер
    receivedEvents := make([]*models.AuditEvent, 0)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var event models.AuditEvent
        err := json.NewDecoder(r.Body).Decode(&event)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        receivedEvents = append(receivedEvents, &event)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    // Создаём наблюдателя
    observer := NewURLObserver(server.URL)
    
    // Создаём тестовое событие
    event := &models.AuditEvent{
        TS:        1234567890,
        Metrics:   []string{"Alloc", "Frees"},
        IPAddress: "192.168.0.42",
    }
    
    // Уведомляем наблюдателя
    err := observer.Notify(event)
    if err != nil {
        t.Fatalf("Failed to notify observer: %v", err)
    }
    
    // Проверяем, что событие было получено
    if len(receivedEvents) != 1 {
        t.Fatalf("Expected 1 event, got %d", len(receivedEvents))
    }
    
    receivedEvent := receivedEvents[0]
    if receivedEvent.TS != event.TS {
        t.Errorf("Expected TS %d, got %d", event.TS, receivedEvent.TS)
    }
    
    if len(receivedEvent.Metrics) != len(event.Metrics) {
        t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(receivedEvent.Metrics))
    }
    
    if receivedEvent.IPAddress != event.IPAddress {
        t.Errorf("Expected IP %s, got %s", event.IPAddress, receivedEvent.IPAddress)
    }
}
```

## Документация

### Использование аудита

1. **Аудит в файл:**
   ```
   ./server --audit-file=/var/log/metrics/audit.log
   или
   export AUDIT_FILE=/var/log/metrics/audit.log
   ./server
   ```

2. **Аудит на удалённый сервер:**
   ```
   ./server --audit-url=http://audit.example.com/api/events
   или
   export AUDIT_URL=http://audit.example.com/api/events
   ./server
   ```

3. **Комбинированный аудит:**
   ```
   ./server --audit-file=/var/log/metrics/audit.log --audit-url=http://audit.example.com/api/events
   ```

4. **Отключение аудита:**
   ```
   ./server  # без параметров аудита
   ```

### Формат логов аудита

Каждая строка в файле аудита представляет собой JSON-объект:
```json
{"ts":1234567890,"metrics":["Alloc","Frees"],"ip_address":"192.168.0.42"}
```

## Заключение

Реализация с использованием паттерна "Наблюдатель" и единого middleware обеспечивает:
- Гибкость в настройке приёмников аудита
- Отделение логики аудита от бизнес-логики
- Возможность добавления новых типов наблюдателей без изменения основного кода
- Выборочное применение аудита только к нужным эндпоинтам
- Корректную обработку тела запроса с его восстановлением для последующих обработчиков
- Извлечение метрик только из JSON-запросов, как и требовалось