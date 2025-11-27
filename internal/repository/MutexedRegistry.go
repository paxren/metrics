package repository

import (
	"sync"

	"github.com/paxren/metrics/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type MutexedRegistry struct {
	Repository
	mutex sync.Mutex
}

func MakeMutexedRegistry(repo Repository) (*MutexedRegistry, error) {

	return &MutexedRegistry{
		Repository: repo,
	}, nil
}

func (ps *MutexedRegistry) executeWithRetry(undateFn func() error) error {

	ps.mutex.Lock()
	err := undateFn()
	ps.mutex.Unlock()
	return err

}

func (ps *MutexedRegistry) UpdateGauge(key string, value float64) error {

	var err1 error
	fn := func() error {
		err := ps.Repository.UpdateGauge(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

func (ps *MutexedRegistry) UpdateCounter(key string, value int64) error {

	var err1 error
	fn := func() error {
		err := ps.Repository.UpdateCounter(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

func (ps *MutexedRegistry) GetGauge(key string) (float64, error) {

	var err error
	var value float64
	fn := func() error {
		value, err = ps.Repository.GetGauge(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err
}

func (ps *MutexedRegistry) GetCounter(key string) (int64, error) {

	var err error
	var value int64
	fn := func() error {
		value, err = ps.Repository.GetCounter(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err
}

func (ps *MutexedRegistry) Ping() error {

	var err1 error

	if pinger, ok := ps.Repository.(Pinger); ok {
		fn := func() error {
			err := pinger.Ping()
			return err
		}
		err1 = ps.executeWithRetry(fn)
	}
	return err1

}

func (ps *MutexedRegistry) MassUpdate(metrics []models.Metrics) error {

	var err1 error
	if massUpdater, ok := ps.Repository.(MassUpdater); ok {
		fn := func() error {
			err := massUpdater.MassUpdate(metrics)
			return err
		}
		err1 = ps.executeWithRetry(fn)
	}
	return err1

}
