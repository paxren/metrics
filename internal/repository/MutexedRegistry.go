package repository

import (
	"sync"

	"github.com/paxren/metrics/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MutexedRegistry реализует потокобезопасную обёртку для любого хранилища метрик.
//
// Использует Mutex для обеспечения безопасного доступа к базовому хранилищу
// из нескольких горутин. Все операции выполняются под блокировкой,
// что гарантирует консистентность данных.
type MutexedRegistry struct {
	Repository
	mutex sync.Mutex
}

// MakeMutexedRegistry создаёт новую потокобезопасную обёртку для указанного хранилища.
//
// Параметры:
//   - repo: базовое хранилище метрик
//
// Возвращает:
//   - *MutexedRegistry: указатель на созданную обёртку
//   - error: всегда nil
//
// Пример использования:
//
//	storage := MakeMemStorage()
//	mutexedStorage, _ := MakeMutexedRegistry(storage)
//	// Теперь можно безопасно использовать mutexedStorage из нескольких горутин
func MakeMutexedRegistry(repo Repository) (*MutexedRegistry, error) {

	return &MutexedRegistry{
		Repository: repo,
	}, nil
}

// executeWithRetry выполняет функцию под блокировкой мьютекса.
//
// Внутренний вспомогательный метод для обеспечения потокобезопасности.
//
// Параметры:
//   - undateFn: функция для выполнения под блокировкой
//
// Возвращает:
//   - error: ошибка выполнения функции, если она произошла
func (ps *MutexedRegistry) executeWithRetry(undateFn func() error) error {

	ps.mutex.Lock()
	err := undateFn()
	ps.mutex.Unlock()
	return err

}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: ошибка базового хранилища, если она произошла
func (ps *MutexedRegistry) UpdateGauge(key string, value float64) error {

	var err1 error
	fn := func() error {
		err := ps.Repository.UpdateGauge(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Параметры:
//   - key: имя метрики
//   - value: значение, которое нужно добавить к текущему
//
// Возвращает:
//   - error: ошибка базового хранилища, если она произошла
func (ps *MutexedRegistry) UpdateCounter(key string, value int64) error {

	var err1 error
	fn := func() error {
		err := ps.Repository.UpdateCounter(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

// GetGauge возвращает значение метрики типа gauge по имени.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - float64: значение метрики
//   - error: ошибка базового хранилища, если она произошла
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

// GetCounter возвращает значение метрики типа counter по имени.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - int64: значение метрики
//   - error: ошибка базового хранилища, если она произошла
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

// Ping проверяет доступность базового хранилища, если оно поддерживает интерфейс Pinger.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Возвращает:
//   - error: ошибка проверки соединения, если хранилище не поддерживает Pinger или произошла ошибка
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

// MassUpdate обновляет множество метрик за одну операцию, если базовое хранилище поддерживает интерфейс MassUpdater.
//
// Операция является потокобезопасной благодаря использованию мьютекса.
//
// Параметры:
//   - metrics: срез метрик для обновления
//
// Возвращает:
//   - error: ошибка обновления, если хранилище не поддерживает MassUpdater или произошла ошибка
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
