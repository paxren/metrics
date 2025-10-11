package models

import "errors"

const (
	Counter = "counter"
	Gauge   = "gauge"
)

var (
	ErrGaugeNotFound   = errors.New("метрика gauge не найдена")
	ErrCounterNotFound = errors.New("метрика gauge не найдена")
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

type MemStorage struct {
	counters map[string]int64
	gauges   map[string]float64
}

func MakeMemStorage() *MemStorage {

	return &MemStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *MemStorage) UpdateGauge(key string, value float64) error {

	m.gauges[key] = value
	return nil
}

func (m *MemStorage) GetGauge(key string) (float64, error) {

	v, ok := m.gauges[key]

	if !ok {
		return 0, ErrGaugeNotFound
	}

	return v, nil
}

func (m *MemStorage) GetCounter(key string) (int64, error) {

	v, ok := m.counters[key]

	if !ok {
		return 0, ErrCounterNotFound
	}

	return v, nil
}

func (m *MemStorage) GetGauges() map[string]float64 {

	//todo добавить проверку на наличие ключа
	return m.gauges
}

func (m *MemStorage) GetCounters() map[string]int64 {

	//todo добавить проверку на наличие ключа
	return m.counters
}

func (m *MemStorage) UpdateCounter(key string, value int64) error {

	m.counters[key] += value

	// c, ok := m.counters[key]

	// if !ok {
	// 	c = make([]int64, 0)

	// }

	// c = append(c, value)

	// m.counters[key] = c

	return nil
}
