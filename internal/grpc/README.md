# gRPC интеграция

## Обзор

Пакет `grpc` предоставляет реализацию gRPC-сервера и клиента для обмена метриками между агентом и сервером.

## Архитектура

```
┌─────────────┐                    ┌─────────────┐
│    Агент    │                    │   Сервер    │
│             │                    │             │
│  gRPC-клиент│─────────────────>│ gRPC-сервер │
│             │                    │             │
│  HTTP-клиент│─────────────────>│ HTTP-сервер │
│             │                    │             │
└─────────────┘                    └─────────────┘
```

## Компоненты

### Сервер (`server.go`)

**Основные типы:**

- `Server` - реализация gRPC-сервера для работы с метриками
- `TrustedSubnetInterceptor` - interceptor для проверки доверенной подсети

**Основные функции:**

- `NewServer()` - создание нового gRPC-сервера
- `UpdateMetrics()` - обработка запроса на обновление метрик
- `checkTrustedSubnet()` - проверка IP-адреса клиента
- `protoToMetric()` - конвертация proto.Metric в models.Metrics
- `mTypeToString()` - конвертация типа метрики в строку
- `TrustedSubnetInterceptor()` - создание interceptor для проверки подсети
- `StartServer()` - запуск gRPC-сервера

### Клиент (`client.go`)

**Основные типы:**

- `Client` - реализация gRPC-клиента для отправки метрик

**Основные функции:**

- `NewClient()` - создание нового gRPC-клиента
- `SendMetrics()` - отправка батча метрик на сервер
- `Close()` - закрытие соединения с сервером
- `metricToProto()` - конвертация models.Metrics в proto.Metric
- `stringToMType()` - конвертация строки типа метрики в proto.Metric_MType
- `getLocalIP()` - получение локального IP-адреса

## Конфигурация

### Сервер

Добавлен новый параметр конфигурации:

```go
type ServerConfig struct {
    // ... существующие поля
    GRPCAddress HostAddress
}
```

**Переменная окружения:**
- `GRPC_ADDRESS` - адрес gRPC-сервера (например, `localhost:3200`)

**Флаг командной строки:**
- `-grpc-a` - адрес gRPC-сервера

**Файл конфигурации:**
```json
{
  "grpc_address": "localhost:3200"
}
```

### Агент

Добавлен новый параметр конфигурации:

```go
type AgentConfig struct {
    // ... существующие поля
    GRPCAddress HostAddress
}
```

**Переменная окружения:**
- `GRPC_ADDRESS` - адрес gRPC-сервера (например, `localhost:3200`)

**Флаг командной строки:**
- `-grpc-a` - адрес gRPC-сервера

**Файл конфигурации:**
```json
{
  "grpc_address": "localhost:3200"
}
```

## Использование

### Запуск сервера с gRPC

```bash
# Через переменную окружения
export GRPC_ADDRESS=localhost:3200
./server

# Через флаг командной строки
./server -grpc-a localhost:3200

# Через файл конфигурации
./server -c config.json
```

### Запуск агента с gRPC

```bash
# Через переменную окружения
export GRPC_ADDRESS=localhost:3200
./agent

# Через флаг командной строки
./agent -grpc-a localhost:3200

# Через файл конфигурации
./agent -c config.json
```

## Протокол обмена данными

### Proto-файл

Файл [`api/metrics.proto`](../../api/metrics.proto) определяет протокол обмена метриками:

```protobuf
syntax = "proto3";

package metrics;

option go_package = "github.com/paxren/metrics/internal/proto";

message Metric {
  string id = 1;
  enum MType {
    GAUGE = 0;
    COUNTER = 1;
  }
  MType type = 2;
  int64 delta = 3;
  double value = 4;
}

message UpdateMetricsRequest {
  repeated Metric metrics = 1;
}

message UpdateMetricsResponse {}

service Metrics {
  rpc UpdateMetrics(UpdateMetricsRequest) returns (UpdateMetricsResponse);
}
```

### Проверка доверенной подсети

gRPC-сервер использует `UnaryInterceptor` для проверки IP-адреса клиента:

1. Клиент добавляет IP-адрес в метаданные запроса с ключом `x-real-ip`
2. Interceptor извлекает IP из метаданных
3. IP проверяется на принадлежность к доверенной подсети
4. Если IP не в доверенной подсети, возвращается ошибка `PermissionDenied`

**Пример добавления IP в метаданные (клиент):**

```go
md := metadata.Pairs("x-real-ip", localIP)
ctx := metadata.NewOutgoingContext(ctx, md)
```

**Пример проверки IP (сервер):**

```go
md, ok := metadata.FromIncomingContext(ctx)
if !ok {
    return status.Error(codes.PermissionDenied, "metadata not found")
}

realIP := md["x-real-ip"]
if len(realIP) == 0 {
    return status.Error(codes.PermissionDenied, "x-real-ip metadata is required")
}

ip := subnetMW.ParseIP(realIP[0])
if ip == nil {
    return status.Error(codes.PermissionDenied, "invalid IP address")
}

if !subnetMW.Contains(ip) {
    return status.Error(codes.PermissionDenied, "IP address is not in trusted subnet")
}
```

## Выбор протокола

Агент автоматически выбирает протокол на основе конфигурации:

- Если указан `GRPC_ADDRESS` - используется gRPC
- Иначе используется HTTP

**Пример логики выбора:**

```go
func (a *Agent) SendAll(metrics []models.Metrics) []error {
    // Если включен gRPC, используем gRPC-клиент
    if a.useGRPC && a.grpcClient != nil {
        return a.sendMetricsViaGRPC(metrics)
    }
    
    // Иначе используем HTTP
    return a.sendMetricsViaHTTP(metrics)
}
```

## Тестирование

### Запуск тестов

```bash
go test ./internal/grpc/...
```

### Покрытие тестами

- `TestTrustedSubnetMiddleware_ParseIP` - тестирование парсинга IP-адреса
- `TestTrustedSubnetMiddleware_Contains` - тестирование проверки принадлежности IP к подсети
- `TestServer_UpdateMetrics` - тестирование метода UpdateMetrics
- `TestServer_protoToMetric` - тестирование конвертации метрик

## Зависимости

```go
require (
    google.golang.org/grpc v1.79.2
    google.golang.org/protobuf v1.36.11
)
```

## Генерация Go-кода из proto

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/metrics.proto
```

Сгенерированные файлы:
- `internal/proto/metrics.pb.go` - сообщения protobuf
- `internal/proto/metrics_grpc.pb.go` - gRPC сервис

## Примечания

1. **Параллельная работа**: gRPC-сервер работает параллельно с HTTP-сервером на разных портах
2. **Совместимость**: gRPC-клиент полностью совместим с существующей логикой агента
3. **Безопасность**: Проверка доверенной подсети работает одинаково для HTTP и gRPC
4. **Конфигурация**: Все параметры настраиваются через переменные окружения, флаги и файлы конфигурации
