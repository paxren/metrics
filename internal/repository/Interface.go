package repository

import (
	"errors"

	"github.com/paxren/metrics/internal/models"
)

var (
	ErrGaugeNotFound   = errors.New("метрика gauge не найдена")
	ErrCounterNotFound = errors.New("метрика counter не найдена")
)

type Repository interface {
	UpdateGauge(key string, value float64) error
	UpdateCounter(key string, value int64) error
	GetGauge(key string) (float64, error)
	GetCounter(key string) (int64, error)
	GetGaugesKeys() []string
	GetCountersKeys() []string
}

type Pinger interface {
	Ping() error
}

type MassUpdater interface {
	MassUpdate(metrics []models.Metrics) error
}
