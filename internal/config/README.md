# internal/config

В этом пакете хранятся конфигурации приложения.

## Источники конфигурации

Конфигурация приложения может быть загружена из различных источников в следующем порядке приоритета (от высшего к низшему):

1. **Флаги командной строки** - имеют наивысший приоритет
2. **Переменные окружения** - средний приоритет
3. **Файл конфигурации JSON** - низший приоритет
4. **Значения по умолчанию** - используются, если ничего не задано

## Файл конфигурации JSON

Имя файла конфигурации может быть задано через:
- Флаг командной строки: `-c` или `-config`
- Переменную окружения: `CONFIG`

### Формат файла для агента

```json
{
    "address": "localhost:8080",
    "report_interval": "1s",
    "poll_interval": "1s",
    "rate_limit": 1,
    "key": "hash_key",
    "crypto_key": "/path/to/key.pem"
}
```

### Формат файла для сервера

```json
{
    "address": "localhost:8080",
    "restore": true,
    "store_interval": "1s",
    "store_file": "/path/to/file.db",
    "database_dsn": "",
    "key": "hash_key",
    "audit_file": "/path/to/audit.log",
    "audit_url": "http://audit.example.com/logs",
    "crypto_key": "/path/to/key.pem"
}
```

## Поддерживаемые поля

### Агент
- `address` (string) - адрес сервера в формате host:port
- `report_interval` (string) - интервал отправки метрик (например, "1s", "30s", "5m")
- `poll_interval` (string) - интервал сбора метрик (например, "1s", "2s")
- `rate_limit` (int64) - лимит скорости отправки запросов
- `key` (string) - ключ для хеширования
- `crypto_key` (string) - путь к файлу с публичным ключом

### Сервер
- `address` (string) - адрес сервера в формате host:port
- `restore` (bool) - флаг восстановления из файла
- `store_interval` (string) - интервал сохранения в файл (например, "1s", "5m")
- `store_file` (string) - путь к файлу для сохранения метрик
- `database_dsn` (string) - строка подключения к базе данных
- `key` (string) - ключ для хеширования
- `audit_file` (string) - путь к файлу для аудита
- `audit_url` (string) - URL для отправки аудита
- `crypto_key` (string) - путь к файлу с приватным ключом

## Примеры использования

### Использование файла конфигурации

```bash
# Через флаг
./agent -c config.json
./server -config config.json

# Через переменную окружения
CONFIG=config.json ./agent
CONFIG=config.json ./server
```

### Комбинирование источников

```bash
# Флаг переопределит значение из файла
./agent -c config.json -a "localhost:9090"

# Переменная окружения переопределит значение из файла
ADDRESS="localhost:9090" ./agent -c config.json
```
