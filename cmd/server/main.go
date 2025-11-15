package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var (
	serverConfig = config.NewServerConfig()
)

func init() {
	serverConfig.Init()
}

func main() {

	//обработка сигтерм, по статье https://habr.com/ru/articles/908344/
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	//init logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		// вызываем панику, если ошибка
		panic("cannot initialize zap")
	}
	defer logger.Sync()

	hlog := handler.NewLogger(logger)
	sugar := logger.Sugar()

	// init params & envs
	sugar.Infow(
		"serverConfig before",
		"serverConfig", serverConfig,
	)
	serverConfig.Parse()
	sugar.Infow(
		"serverConfig parse",
		"serverConfig", serverConfig,
	)

	//testSQL()
	//os.Exit(1)
	// PREPARE STORAGES
	finish := make([]func() error, 0, 1)
	var storage repository.Repository
	sugar.Infow(
		"Storage init0",
		"Storage obj", storage,
	)

	var handlerv *handler.Handler
	if serverConfig.DatabaseDSN != "" {

		pstorage, err := repository.MakePostgresStorage(serverConfig.DatabaseDSN)
		if err != nil {
			// вызываем панику, если ошибка
			panic("cannot initialize postgress")
		}

		sugar.Infow(
			"postgresStorage init",
			"postgresStorage obj", pstorage,
		)

		storage = pstorage

		finish = append(finish, pstorage.Close)
	} else {
		mstorage := repository.MakeMemStorage()
		//работа с файлами
		savedStorage := repository.MakeSavedRepo(mstorage, serverConfig.FileStoragePath, serverConfig.StoreInterval)
		sugar.Infow(
			"savedStorage init",
			"savedStorage obj", savedStorage,
		)
		if serverConfig.Restore {
			_ = savedStorage.Load(serverConfig.FileStoragePath)
			// if err != nil {
			// 	panic(err)
			// }
		}

		storage = savedStorage

		finish = append(finish, savedStorage.Save)
	}

	//запуск обработчиков
	sugar.Infow(
		"Storage init1",
		"Storage obj", storage,
	)

	handlerv = handler.NewHandler(storage)
	sugar.Infow(
		"handler init",
		"handler obj", handlerv,
	)
	//TODO переделать
	//handlerv.SetDBString(serverConfig.DatabaseDSN)
	sugar.Infow(
		"handler set db",
		"handler obj", handlerv,
	)
	//fmt.Printf("host param: %s", hostAdress.String())

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
	r.Post(`/value/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Post(`/value`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Get(`/value/{metric_type}/{metric_name}`, hlog.WithLogging(handlerv.GetMetric))
	r.Get(`/ping`, hlog.WithLogging(handlerv.PingDB))
	r.Get(`/ping/`, hlog.WithLogging(handlerv.PingDB))
	r.Get(`/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetMain)))

	server := &http.Server{
		Addr:    serverConfig.Address.String(),
		Handler: r,
	}

	go func() {
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			sugar.Infow(
				"Failed to serve listener",
				"error", err,
			)
			panic(err)
		}

	}()

	//обработка сигтерм TODO добработать или переработать после понимания контекста и др
	<-rootCtx.Done()
	stop()
	server.Shutdown(context.Background())
	for _, f := range finish {
		f()
	}
	//savedStorage.Save()
	//TODO закрытие базы
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
