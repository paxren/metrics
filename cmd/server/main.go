package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"

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

		pstorage, err := repository.MakePostgresStorageWithRetry(serverConfig.DatabaseDSN)

		if err != nil {
			// вызываем панику, если ошибка
			sugar.Fatal(
				"Storage init1",
				"Storage obj", pstorage,
				"err", pstorage,
			)
			//panic("cannot initialize postgress")
		}
		mutexed, err := repository.MakeMutexedRegistry(pstorage)
		if err != nil {
			// вызываем панику, если ошибка
			sugar.Fatal(
				"Storage init1",
				"Storage obj", pstorage,
				"err", pstorage,
			)
			//panic("cannot initialize postgress")
		}

		sugar.Infow(
			"postgresStorage init",
			"postgresStorage obj", pstorage,
		)

		//storage = pstorage
		storage = mutexed

		finish = append(finish, pstorage.Close)
	} else {
		mstorage := repository.MakeConcurentMemStorage()
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

	hasher := handler.NewHasher(serverConfig.Key)

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
	r.Post(`/value/`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON))))
	r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
	r.Post(`/value`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON))))
	r.Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON))))
	r.Post(`/updates`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON)))) //hasher.HashMiddleware(
	r.Post(`/updates/`, hlog.WithLogging(hasher.HashMiddleware(handler.GzipMiddleware(handlerv.UpdatesJSON))))
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
