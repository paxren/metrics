package repository

import (
	"sync"
)

// ConcurentMemStorage реализует потокобезопасное хранилище метрик в оперативной памяти.
//
// Использует sync.Map для хранения метрик, что обеспечивает безопасный доступ
// из нескольких горутин без необходимости в явной блокировке.
// Все данные теряются при завершении работы программы.
type ConcurentMemStorage struct {
	counters sync.Map
	gauges   sync.Map
}

// MakeConcurentMemStorage создаёт новое экземпляр потокобезопасного хранилища в памяти.
//
// Инициализирует пустые sync.Map для хранения метрик.
//
// Возвращает:
//   - *ConcurentMemStorage: указатель на созданное хранилище
//
// Пример использования:
//
//	storage := MakeConcurentMemStorage()
//	storage.UpdateGauge("alloc", 123.45)
//	storage.UpdateCounter("requests", 1)
func MakeConcurentMemStorage() *ConcurentMemStorage {

	return &ConcurentMemStorage{}
}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Операция является потокобезопасной.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: всегда nil, так как операция не может завершиться с ошибкой
func (m *ConcurentMemStorage) UpdateGauge(key string, value float64) error {

	m.gauges.Store(key, value)
	return nil
}

// GetGauge возвращает значение метрики типа gauge по имени.
//
// Операция является потокобезопасной.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - float64: значение метрики
//   - error: ErrGaugeNotFound, если метрика не найдена
func (m *ConcurentMemStorage) GetGauge(key string) (float64, error) {

	v, ok := m.gauges.Load(key)

	if !ok {
		return 0, ErrGaugeNotFound
	}

	return v.(float64), nil
}

// GetCounter возвращает значение метрики типа counter по имени.
//
// Операция является потокобезопасной.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - int64: значение метрики
//   - error: ErrCounterNotFound, если метрика не найдена
func (m *ConcurentMemStorage) GetCounter(key string) (int64, error) {

	v, ok := m.counters.Load(key)

	if !ok {
		return 0, ErrCounterNotFound
	}

	return v.(int64), nil
}

// UpdateCounter обновляет или создаёт метрику типа counter, устанавливая указанное значение.
//
// ВНИМАНИЕ: В данной реализации значение перезаписывается, а не добавляется к существующему.
// Если метрика с таким именем не существует, она будет создана с указанным значением.
// Операция является потокобезопасной.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: всегда nil, так как операция не может завершиться с ошибкой
func (m *ConcurentMemStorage) UpdateCounter(key string, value int64) error {

	m.counters.Store(key, value)
	return nil
}

// GetGaugesKeys возвращает список всех имён метрик типа gauge.
//
// Операция является потокобезопасной.
//
// Возвращает:
//   - []string: срез имён метрик gauge
func (m *ConcurentMemStorage) GetGaugesKeys() []string {

	keys := make([]string, 0)

	m.gauges.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true // if false, Range stops
	})

	return keys
}

// GetCountersKeys возвращает список всех имён метрик типа counter.
//
// Операция является потокобезопасной.
//
// Возвращает:
//   - []string: срез имён метрик counter
func (m *ConcurentMemStorage) GetCountersKeys() []string {

	keys := make([]string, 0)
	m.counters.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true // if false, Range stops
	})

	return keys
}
