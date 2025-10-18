package repository

import "errors"

var (
	ErrGaugeNotFound   = errors.New("метрика gauge не найдена")
	ErrCounterNotFound = errors.New("метрика gauge не найдена")
)

type Repository interface {
	UpdateGauge(key string, value float64) error
	UpdateCounter(key string, value int64) error
	GetGauge(key string) (float64, error)
	GetCounter(key string) (int64, error)
	GetGaugesKeys() []string
	GetCountersKeys() []string
}
