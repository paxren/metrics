package repository

import (
	"fmt"
	"time"

	"github.com/paxren/metrics/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// PostgresStorageWithRetry реализует хранилище метрик в PostgreSQL с автоматическим повтором операций при временных ошибках.
//
// Оборачивает PostgresStorage и добавляет логику повторных попыток для операций,
// которые могут завершиться с временной ошибкой (например, потеря соединения).
// Использует PostgresErrorClassifier для определения, можно ли повторить операцию.
type PostgresStorageWithRetry struct {
	*PostgresStorage
	classifier *PostgresErrorClassifier
}

// MakePostgresStorageWithRetry создаёт новое хранилище метрик в PostgreSQL с поддержкой повторов.
//
// Параметры:
//   - con: строка подключения к базе данных в формате DSN
//
// Возвращает:
//   - *PostgresStorageWithRetry: указатель на созданное хранилище
//   - error: ошибка при подключении или миграции
//
// Пример использования:
//
//	storage, err := MakePostgresStorageWithRetry("host=localhost user=postgres password=postgres dbname=metrics sslmode=disable")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer storage.Close()
func MakePostgresStorageWithRetry(con string) (*PostgresStorageWithRetry, error) {

	repo, err := MakePostgresStorage(con)
	if err != nil {
		fmt.Printf("err=%v", err)
		return nil, err
	}

	return &PostgresStorageWithRetry{
		PostgresStorage: repo,
		classifier:      NewPostgresErrorClassifier(),
	}, nil
}

// executeWithRetry выполняет функцию с повторными попытками при временных ошибках.
//
// Использует экспоненциальную задержку между попытками.
// Повторяет только операции, которые классифицированы как временные ошибки.
//
// Параметры:
//   - undateFn: функция для выполнения с повторами
//
// Возвращает:
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) executeWithRetry(undateFn func() error) error {
	const maxRetries = 3
	var lastErr error
	var waitSec int64 = 1

	for attempt := 0; attempt < maxRetries; attempt++ {

		err := undateFn()
		if err == nil {
			return nil
		}
		// Определяем классификацию ошибки
		classification := ps.classifier.Classify(err)

		if classification == NonRetriable {
			// Нет смысла повторять, возвращаем ошибку
			fmt.Printf("Непредвиденная ошибка: %v\n", err)
			return err
		}

		lastErr = err

		time.Sleep(time.Duration(waitSec) * time.Second)
		waitSec += 2
		// .... делаем что-то полезное
	}

	return fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)
}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) UpdateGauge(key string, value float64) error {

	var err1 error
	fn := func() error {
		err := ps.PostgresStorage.UpdateGauge(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Параметры:
//   - key: имя метрики
//   - value: значение, которое нужно добавить к текущему
//
// Возвращает:
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) UpdateCounter(key string, value int64) error {

	var err1 error
	fn := func() error {
		err := ps.PostgresStorage.UpdateCounter(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

// GetGauge возвращает значение метрики типа gauge по имени.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - float64: значение метрики
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) GetGauge(key string) (float64, error) {

	var err error
	var value float64
	fn := func() error {
		value, err = ps.PostgresStorage.GetGauge(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err
}

// GetCounter возвращает значение метрики типа counter по имени.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - int64: значение метрики
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) GetCounter(key string) (int64, error) {

	var err error
	var value int64
	fn := func() error {
		value, err = ps.PostgresStorage.GetCounter(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err
}

// Ping проверяет доступность базы данных.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Возвращает:
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) Ping() error {

	var err1 error

	fn := func() error {
		err := ps.PostgresStorage.Ping()
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}

// MassUpdate обновляет множество метрик за одну операцию.
//
// Выполняет операцию с повторными попытками при временных ошибках.
//
// Параметры:
//   - metrics: срез метрик для обновления
//
// Возвращает:
//   - error: ошибка после всех попыток или nil при успехе
func (ps *PostgresStorageWithRetry) MassUpdate(metrics []models.Metrics) error {

	var err1 error

	fn := func() error {
		err := ps.PostgresStorage.MassUpdate(metrics)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

}
