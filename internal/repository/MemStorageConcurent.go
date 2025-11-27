package repository

import (
	"sync"
)

// ПОТОКО НЕБЕЗОПАСНО!

type ConcurentMemStorage struct {
	counters sync.Map
	gauges   sync.Map
}

func MakeConcurentMemStorage() *ConcurentMemStorage {

	return &ConcurentMemStorage{}
}

func (m *ConcurentMemStorage) UpdateGauge(key string, value float64) error {

	m.gauges.Store(key, value)
	return nil
}

func (m *ConcurentMemStorage) GetGauge(key string) (float64, error) {

	v, ok := m.gauges.Load(key)

	if !ok {
		return 0, ErrGaugeNotFound
	}

	return v.(float64), nil
}

func (m *ConcurentMemStorage) GetCounter(key string) (int64, error) {

	v, ok := m.counters.Load(key)

	if !ok {
		return 0, ErrCounterNotFound
	}

	return v.(int64), nil
}

func (m *ConcurentMemStorage) UpdateCounter(key string, value int64) error {

	m.counters.Store(key, value)
	return nil
}

func (m *ConcurentMemStorage) GetGaugesKeys() []string {

	//todo добавить проверку на наличие ключа

	keys := make([]string, 0)

	m.gauges.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true // if false, Range stops
	})

	return keys
}

func (m *ConcurentMemStorage) GetCountersKeys() []string {

	keys := make([]string, 0)
	m.counters.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true // if false, Range stops
	})

	return keys
}
