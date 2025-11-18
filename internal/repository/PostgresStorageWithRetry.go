package repository

import (
	"fmt"
	"time"

	"github.com/paxren/metrics/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// ПОТОКО НЕБЕЗОПАСНО!

// type UpdateGauge func(key string, value float64) error

// func (f UpdateGauge) Execute(w ResponseWriter, r *Request) {
// 	f(w, r)
// }

type PostgresStorageWithRetry struct {
	*PostgresStorage
	classifier *PostgresErrorClassifier
}

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

func (ps *PostgresStorageWithRetry) UpdateGauge(key string, value float64) error {

	var err1 error
	fn := func() error {
		err := ps.PostgresStorage.UpdateGauge(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1
	// const maxRetries = 3
	// var lastErr error
	// var waitSec int64 = 1

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	err := ps.PostgresStorage.UpdateGauge(key, value)
	// 	if err == nil {
	// 		return nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// 	// .... делаем что-то полезное
	// }

	// return fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)

}

func (ps *PostgresStorageWithRetry) UpdateCounter(key string, value int64) error {

	var err1 error
	fn := func() error {
		err := ps.PostgresStorage.UpdateCounter(key, value)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

	// const maxRetries = 3
	// var lastErr error
	// var waitSec int64 = 1

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	err := ps.PostgresStorage.UpdateCounter(key, value)
	// 	if err == nil {
	// 		return nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// 	// .... делаем что-то полезное
	// }

	// return fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)
}

func (ps *PostgresStorageWithRetry) GetGauge(key string) (float64, error) {

	var err error
	var value float64
	fn := func() error {
		value, err = ps.PostgresStorage.GetGauge(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err

	// const maxRetries = 3
	// var lastErr error
	// var err error
	// var waitSec int64 = 1
	// var res float64

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	res, err = ps.PostgresStorage.GetGauge(key)
	// 	if err == nil {
	// 		return res, nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return 0, err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// }

	// return 0, fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)

}

func (ps *PostgresStorageWithRetry) GetCounter(key string) (int64, error) {

	var err error
	var value int64
	fn := func() error {
		value, err = ps.PostgresStorage.GetCounter(key)
		return err
	}
	_ = ps.executeWithRetry(fn)

	return value, err

	// const maxRetries = 3
	// var lastErr error
	// var err error
	// var waitSec int64 = 1
	// var res int64

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	res, err = ps.PostgresStorage.GetCounter(key)
	// 	if err == nil {
	// 		return res, nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return 0, err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// }

	// return 0, fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)

}

func (ps *PostgresStorageWithRetry) Ping() error {

	var err1 error

	fn := func() error {
		err := ps.PostgresStorage.Ping()
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

	// const maxRetries = 3
	// var lastErr error
	// var err error
	// var waitSec int64 = 1

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	err = ps.PostgresStorage.Ping()
	// 	if err == nil {
	// 		return nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// }

	// return fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)

}

func (ps *PostgresStorageWithRetry) MassUpdate(metrics []models.Metrics) error {

	var err1 error

	fn := func() error {
		err := ps.PostgresStorage.MassUpdate(metrics)
		return err
	}
	err1 = ps.executeWithRetry(fn)

	return err1

	// const maxRetries = 3
	// var lastErr error
	// var err error
	// var waitSec int64 = 1

	// for attempt := 0; attempt < maxRetries; attempt++ {

	// 	err = ps.PostgresStorage.MassUpdate(metrics)
	// 	if err == nil {
	// 		return nil
	// 	}
	// 	// Определяем классификацию ошибки
	// 	classification := ps.classifier.Classify(err)

	// 	if classification == NonRetriable {
	// 		// Нет смысла повторять, возвращаем ошибку
	// 		fmt.Printf("Непредвиденная ошибка: %v\n", err)
	// 		return err
	// 	}

	// 	lastErr = err

	// 	time.Sleep(time.Duration(waitSec) * time.Second)
	// 	waitSec += 2
	// }

	// return fmt.Errorf("операция прервана после %d попыток: %w", maxRetries, lastErr)

}
