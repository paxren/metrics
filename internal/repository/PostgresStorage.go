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

// ПОТОКО НЕБЕЗОПАСНО!

type PostgresStorage struct {
	db *sql.DB
}

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
		"file://../../migrations",
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

func (ps *PostgresStorage) Close() error {

	ps.db.Close()
	return nil
}

func (ps *PostgresStorage) UpdateGauge(key string, value float64) error {

	return ps.update(models.Gauge, key, nil, &value)
}

func (ps *PostgresStorage) UpdateCounter(key string, value int64) error {

	return ps.update(models.Counter, key, &value, nil)
}

func (ps *PostgresStorage) update(mtype string, id string, delta *int64, value *float64) error {

	_, err := ps.db.ExecContext(
		context.Background(), `
		INSERT INTO metrics (id, mtype, delta, value, hash) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET 
			mtype = EXCLUDED.mtype,	
			delta = EXCLUDED.delta,
			value = EXCLUDED.value,
			hash = EXCLUDED.hash
		`,
		id, mtype, delta, value, "")

	return err

}

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

func (ps *PostgresStorage) GetGaugesKeys() []string {

	//todo переделать интерфейс и не подавлять ошибки?
	return ps.getKeys(models.Gauge)

}

func (ps *PostgresStorage) GetCountersKeys() []string {

	//todo переделать интерфейс и не подавлять ошибки?
	return ps.getKeys(models.Counter)
}

func (ps *PostgresStorage) Ping() error {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := ps.db.PingContext(ctx); err != nil {

		return err
	}

	return nil
}

func testSQL() {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `dbtest1`, `dbtest1`, `dbtest1`)

	fmt.Println("1")
	db, err := sql.Open("pgx", ps)
	if err != nil {
		fmt.Printf("err=%v", err)
		return
	}
	defer db.Close() //TODO вынести в конец программы

	fmt.Println("2")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Printf("driver err! err=%v", err)
		return
	}

	fmt.Println("3")
	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		"postgres", driver)
	if err != nil {
		fmt.Printf("migration err! err=%v", err)
		return
	}
	fmt.Println("4")
	m.Up()

	fmt.Println("5")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		fmt.Printf("err=%v", err)
		return
	}

	var ui int64 = 200
	metric := models.Metrics{
		ID:    "test1",
		MType: "counter",
		Delta: &ui,
	}

	fmt.Println("6")
	res, err := db.ExecContext(context.Background(), `
    INSERT INTO metrics (id, mtype, delta, value, hash) 
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (id) DO UPDATE SET 
        delta = EXCLUDED.delta,
        value = EXCLUDED.value,
        hash = EXCLUDED.hash
	`,
		metric.ID, metric.MType, metric.Delta, metric.Value, metric.Hash)

	fmt.Println(res)

	if err != nil {
		fmt.Printf("isert err! err=%v", err)
		return
	}

	row := db.QueryRowContext(context.Background(),
		"SELECT delta, value, mtype FROM metrics WHERE id = $1", metric.ID)
	var (
		mtype string
		delta sql.NullInt64
		value sql.NullFloat64
	)
	// порядок переменных должен соответствовать порядку колонок в запросе
	err = row.Scan(&delta, &value, &mtype)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v | %v | %s \r\n", delta, value, mtype)

}
