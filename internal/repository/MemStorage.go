package repository

// ПОТОКО НЕБЕЗОПАСНО!

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

func (m *MemStorage) UpdateCounter(key string, value int64) error {

	m.counters[key] += value
	return nil
}

func (m *MemStorage) GetGaugesKeys() []string {

	//todo добавить проверку на наличие ключа

	keys := make([]string, 0)

	for k := range m.gauges {
		keys = append(keys, k)
	}

	return keys
}

func (m *MemStorage) GetCountersKeys() []string {

	keys := make([]string, 0)

	for k := range m.counters {
		keys = append(keys, k)
	}

	return keys
}
