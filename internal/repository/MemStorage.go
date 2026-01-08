package repository

// MemStorage реализует хранилище метрик в оперативной памяти.
//
// ВНИМАНИЕ: Данная реализация не является потокобезопасной!
// Для использования в многопоточной среде применяйте ConcurentMemStorage или MutexedRegistry.
//
// Хранит метрики в двух map: для счётчиков (counters) и датчиков (gauges).
// Все данные теряются при завершении работы программы.
type MemStorage struct {
	counters map[string]int64
	gauges   map[string]float64
}

// MakeMemStorage создаёт новое экземпляр хранилища в памяти.
//
// Инициализирует пустые map для хранения метрик.
//
// Возвращает:
//   - *MemStorage: указатель на созданное хранилище
//
// Пример использования:
//
//	storage := MakeMemStorage()
//	storage.UpdateGauge("alloc", 123.45)
//	storage.UpdateCounter("requests", 1)
func MakeMemStorage() *MemStorage {

	return &MemStorage{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: всегда nil, так как операция не может завершиться с ошибкой
func (m *MemStorage) UpdateGauge(key string, value float64) error {

	m.gauges[key] = value
	return nil
}

// GetGauge возвращает значение метрики типа gauge по имени.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - float64: значение метрики
//   - error: ErrGaugeNotFound, если метрика не найдена
func (m *MemStorage) GetGauge(key string) (float64, error) {

	v, ok := m.gauges[key]

	if !ok {
		return 0, ErrGaugeNotFound
	}

	return v, nil
}

// GetCounter возвращает значение метрики типа counter по имени.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - int64: значение метрики
//   - error: ErrCounterNotFound, если метрика не найдена
func (m *MemStorage) GetCounter(key string) (int64, error) {

	v, ok := m.counters[key]

	if !ok {
		return 0, ErrCounterNotFound
	}

	return v, nil
}

// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему.
//
// Если метрика с таким именем не существует, она будет создана с указанным значением.
//
// Параметры:
//   - key: имя метрики
//   - value: значение, которое нужно добавить к текущему
//
// Возвращает:
//   - error: всегда nil, так как операция не может завершиться с ошибкой
func (m *MemStorage) UpdateCounter(key string, value int64) error {

	m.counters[key] += value
	return nil
}

// GetGaugesKeys возвращает список всех имён метрик типа gauge.
//
// Возвращает:
//   - []string: срез имён метрик gauge
func (m *MemStorage) GetGaugesKeys() []string {

	keys := make([]string, 0)

	for k := range m.gauges {
		keys = append(keys, k)
	}

	return keys
}

// GetCountersKeys возвращает список всех имён метрик типа counter.
//
// Возвращает:
//   - []string: срез имён метрик counter
func (m *MemStorage) GetCountersKeys() []string {

	keys := make([]string, 0)

	for k := range m.counters {
		keys = append(keys, k)
	}

	return keys
}
