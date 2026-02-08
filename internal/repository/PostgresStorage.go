package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/paxren/metrics/internal/models"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// PostgresStorage реализует хранилище метрик в базе данных PostgreSQL.
//
// ВНИМАНИЕ: Данная реализация не является потокобезопасной!
// Для использования в многопоточной среде применяйте MutexedRegistry.
//
// Использует миграции для создания таблицы метрик и поддерживает
// транзакции для пакетного обновления.
type PostgresStorage struct {
	db *sql.DB
}

// MakePostgresStorage создаёт новое хранилище метрик в PostgreSQL.
//
// Автоматически выполняет миграции базы данных и проверяет соединение.
//
// Параметры:
//   - con: строка подключения к базе данных в формате DSN
//
// Возвращает:
//   - *PostgresStorage: указатель на созданное хранилище
//   - error: ошибка при подключении или миграции
//
// Пример использования:
//
//	storage, err := MakePostgresStorage("host=localhost user=postgres password=postgres dbname=metrics sslmode=disable")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer storage.Close()
func MakePostgresStorage(con string) (*PostgresStorage, error) {

	fmt.Println("1")
	db, err := sql.Open("pgx", con)
	if err != nil {
		fmt.Printf("err=%v", err)
		return nil, err
	}
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	fmt.Println("2")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Printf("driver err! err=%v", err)
		return nil, err
	}

	fmt.Println("3")
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		fmt.Printf("migration err! err=%v", err)
		return nil, err
	}
	fmt.Println("4")
	m.Up()

	fmt.Println("5")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		fmt.Printf("err=%v", err)
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

// Close закрывает соединение с базой данных.
//
// Должен вызываться при завершении работы с хранилищем.
//
// Возвращает:
//   - error: всегда nil
func (ps *PostgresStorage) Close() error {

	ps.db.Close()
	return nil
}

// UpdateGauge обновляет или создаёт метрику типа gauge с указанным именем и значением.
//
// Параметры:
//   - key: имя метрики
//   - value: новое значение метрики
//
// Возвращает:
//   - error: ошибка при выполнении запроса к базе данных
func (ps *PostgresStorage) UpdateGauge(key string, value float64) error {

	return ps.update(models.Gauge, key, nil, &value)
}

// UpdateCounter обновляет или создаёт метрику типа counter, добавляя указанное значение к текущему.
//
// Параметры:
//   - key: имя метрики
//   - value: значение, которое нужно добавить к текущему
//
// Возвращает:
//   - error: ошибка при выполнении запроса к базе данных
func (ps *PostgresStorage) UpdateCounter(key string, value int64) error {

	return ps.update(models.Counter, key, &value, nil)
}

// update внутренний метод для обновления метрики в базе данных.
//
// Использует UPSERT операцию для создания или обновления метрики.
// ВАЖНО: Если тип метрики изменится при старом ID, сумма counter всегда будет обнуляться (NULL + что-то всегда NULL).
//
// Параметры:
//   - mtype: тип метрики ("counter" или "gauge")
//   - id: имя метрики
//   - delta: указатель на значение для counter (nil для gauge)
//   - value: указатель на значение для gauge (nil для counter)
//
// Возвращает:
//   - error: ошибка при выполнении запроса к базе данных
func (ps *PostgresStorage) update(mtype string, id string, delta *int64, value *float64) error {

	//тут важный нюанс, если случайно тип метрики изменится при старом ид, то сумма всегда будет обнуляться (NULL + что-то всегда нулл)
	_, err := ps.db.ExecContext(
		context.Background(), `
		INSERT INTO metrics (id, mtype, delta, value, hash)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			mtype = EXCLUDED.mtype,
			delta = metrics.delta + EXCLUDED.delta,
			value = EXCLUDED.value,
			hash = EXCLUDED.hash
		`,
		id, mtype, delta, value, "")

	return err

}

// GetGauge возвращает значение метрики типа gauge по имени.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - float64: значение метрики
//   - error: ошибка если метрика не найдена или имеет неверный тип
func (ps *PostgresStorage) GetGauge(key string) (float64, error) {

	row := ps.db.QueryRowContext(context.Background(),
		"SELECT delta, value, mtype FROM metrics WHERE id = $1", key)
	var (
		mtype string
		delta sql.NullInt64
		value sql.NullFloat64
	)
	// порядок переменных должен соответствовать порядку колонок в запросе
	err := row.Scan(&delta, &value, &mtype)
	if err != nil {
		return 0, err
	}

	if mtype != models.Gauge {
		return 0, fmt.Errorf("тип в базе не соответствуют нужному mtype = %s", mtype)
	}

	if !value.Valid {
		return 0, fmt.Errorf("значение в базе null mtype = %v", value)
	}

	return value.Float64, nil

}

// GetCounter возвращает значение метрики типа counter по имени.
//
// Параметры:
//   - key: имя метрики
//
// Возвращает:
//   - int64: значение метрики
//   - error: ошибка если метрика не найдена или имеет неверный тип
func (ps *PostgresStorage) GetCounter(key string) (int64, error) {

	row := ps.db.QueryRowContext(context.Background(),
		"SELECT delta, value, mtype FROM metrics WHERE id = $1", key)
	var (
		mtype string
		delta sql.NullInt64
		value sql.NullFloat64
	)
	// порядок переменных должен соответствовать порядку колонок в запросе
	// если метрика не найдена, то тут будет ошибка
	err := row.Scan(&delta, &value, &mtype)
	if err != nil {
		return 0, err
	}

	if mtype != models.Counter {
		return 0, fmt.Errorf("тип в базе не соответствуют нужному mtype = %s", mtype)
	}

	if !delta.Valid {
		return 0, fmt.Errorf("значение в базе null mtype = %v", delta)
	}

	return delta.Int64, nil

}

// getKeys внутренний метод для получения списка имён метрик указанного типа.
//
// TODO: переделать интерфейс и не подавлять ошибки?
//
// Параметры:
//   - mtype: тип метрики ("counter" или "gauge")
//
// Возвращает:
//   - []string: список имён метрик указанного типа
func (ps *PostgresStorage) getKeys(mtype string) []string {

	//todo переделать интерфейс и не подавлять ошибки?
	keys := make([]string, 0, 10)

	rows, _ := ps.db.QueryContext(context.Background(),
		"SELECT id FROM metrics WHERE mtype = $1", mtype)

	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		// if err != nil {
		//     return nil, err
		// }

		keys = append(keys, s)
	}

	_ = rows.Err()
	// if err != nil {
	//     return nil, err
	// }
	return keys
}

// GetGaugesKeys возвращает список всех имён метрик типа gauge.
//
// TODO: переделать интерфейс и не подавлять ошибки?
//
// Возвращает:
//   - []string: срез имён метрик gauge
func (ps *PostgresStorage) GetGaugesKeys() []string {

	//todo переделать интерфейс и не подавлять ошибки?
	return ps.getKeys(models.Gauge)

}

// GetCountersKeys возвращает список всех имён метрик типа counter.
//
// TODO: переделать интерфейс и не подавлять ошибки?
//
// Возвращает:
//   - []string: срез имён метрик counter
func (ps *PostgresStorage) GetCountersKeys() []string {

	//todo переделать интерфейс и не подавлять ошибки?
	return ps.getKeys(models.Counter)

}

// Ping проверяет доступность базы данных.
//
// Использует контекст с таймаутом для предотвращения блокировки.
//
// Возвращает:
//   - error: ошибка если база данных недоступна
func (ps *PostgresStorage) Ping() error {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := ps.db.PingContext(ctx); err != nil {

		return err
	}

	return nil
}

// MassUpdate обновляет множество метрик за одну транзакцию.
//
// Все изменения выполняются в рамках одной транзакции, что обеспечивает
// атомарность операции. При ошибке все изменения откатываются.
//
// ВАЖНО: Если тип метрики изменится при старом ID, сумма counter всегда будет обнуляться (NULL + что-то всегда NULL).
//
// Параметры:
//   - metrics: срез метрик для обновления
//
// Возвращает:
//   - error: ошибка при выполнении транзакции
func (ps *PostgresStorage) MassUpdate(metrics []models.Metrics) error {

	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}

	for _, metric := range metrics {
		//тут важный нюанс, если случайно тип метрики изменится при старом ид, то сумма всегда будет обнуляться (NULL + что-то всегда нулл)
		// все изменения записываются в транзакцию
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO metrics (id, mtype, delta, value, hash)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				mtype = EXCLUDED.mtype,
				delta = metrics.delta + EXCLUDED.delta,
				value = EXCLUDED.value,
				hash = EXCLUDED.hash
			`,
			metric.ID, metric.MType, metric.Delta, metric.Value, "")
		if err != nil {
			// если ошибка, то откатываем изменения
			tx.Rollback()
			return err
		}
	}
	// завершаем транзакцию
	return tx.Commit()

}

// func testSQL() {	//
// 	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
// 		`localhost`, `dbtest1`, `dbtest1`, `dbtest1`)

// 	fmt.Println("1")
// 	db, err := sql.Open("pgx", ps)
// 	if err != nil {
// 		fmt.Printf("err=%v", err)
// 		return
// 	}
// 	defer db.Close() //TODO вынести в конец программы

// 	fmt.Println("2")
// 	driver, err := postgres.WithInstance(db, &postgres.Config{})
// 	if err != nil {
// 		fmt.Printf("driver err! err=%v", err)
// 		return
// 	}

// 	fmt.Println("3")
// 	m, err := migrate.NewWithDatabaseInstance(
// 		"file://../../migrations",
// 		"postgres", driver)
// 	if err != nil {
// 		fmt.Printf("migration err! err=%v", err)
// 		return
// 	}
// 	fmt.Println("4")
// 	m.Up()

// 	fmt.Println("5")
// 	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 	defer cancel()
// 	if err = db.PingContext(ctx); err != nil {
// 		fmt.Printf("err=%v", err)
// 		return
// 	}

// 	var ui int64 = 200
// 	metric := models.Metrics{
// 		ID:    "test1",
// 		MType: "counter",
// 		Delta: &ui,
// 	}

// 	fmt.Println("6")
// 	res, err := db.ExecContext(context.Background(), `
//     INSERT INTO metrics (id, mtype, delta, value, hash)
//     VALUES ($1, $2, $3, $4, $5)
//     ON CONFLICT (id) DO UPDATE SET
//         delta = EXCLUDED.delta,
//         value = EXCLUDED.value,
//         hash = EXCLUDED.hash
// 	`,
// 		metric.ID, metric.MType, metric.Delta, metric.Value, metric.Hash)

// 	fmt.Println(res)

// 	if err != nil {
// 		fmt.Printf("isert err! err=%v", err)
// 		return
// 	}

// 	row := db.QueryRowContext(context.Background(),
// 		"SELECT delta, value, mtype FROM metrics WHERE id = $1", metric.ID)
// 	var (
// 		mtype string
// 		delta sql.NullInt64
// 		value sql.NullFloat64
// 	)
// 	// порядок переменных должен соответствовать порядку колонок в запросе
// 	err = row.Scan(&delta, &value, &mtype)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Printf("%v | %v | %s \r\n", delta, value, mtype)

// }
