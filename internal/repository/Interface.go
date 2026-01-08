package repository

import (
	"errors"

	"github.com/paxren/metrics/internal/models"
)

var (
	// ErrGaugeNotFound возвращается, когда запрошенная метрика типа gauge не найдена в хранилище
	ErrGaugeNotFound = errors.New("метрика gauge не найдена")
	// ErrCounterNotFound возвращается, когда запрошенная метрика типа counter не найдена в хранилище
	ErrCounterNotFound = errors.New("метрика counter не найдена")
)

// Repository определяет интерфейс для хранения и управления метриками.
//
// Предоставляет базовые операции CRUD для метрик двух типов: gauge и counter.
// Реализации этого интерфейса могут использовать различные хранилища:
// память, файлы, базы данных и т.д.
type Repository interface {
	// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением
	UpdateGauge(key string, value float64) error

	// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему
	UpdateCounter(key string, value int64) error

	// GetGauge возвращает значение метрики типа gauge по имени
	GetGauge(key string) (float64, error)

	// GetCounter возвращает значение метрики типа counter по имени
	GetCounter(key string) (int64, error)

	// GetGaugesKeys возвращает список всех имён метрик типа gauge
	GetGaugesKeys() []string

	// GetCountersKeys возвращает список всех имён метрик типа counter
	GetCountersKeys() []string
}

// Pinger определяет интерфейс для проверки доступности хранилища.
//
// Используется для реализации проверки соединения с базой данных
// или другим внешним хранилищем.
type Pinger interface {
	// Ping проверяет доступность хранилища и возвращает ошибку, если хранилище недоступно
	Ping() error
}

// MassUpdater определяет интерфейс для пакетного обновления метрик.
//
// Позволяет эффективно обновлять множество метрик за одну операцию,
// что особенно полезно при работе с базами данных.
type MassUpdater interface {
	// MassUpdate обновляет множество метрик за одну транзакцию
	MassUpdate(metrics []models.Metrics) error
}
