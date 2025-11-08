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

	//os.Exit(1)
	// PREPARE STORAGES
	storage := repository.MakeMemStorage()
	//работа с файлами
	savedStorage := repository.MakeSavedRepo(storage, serverConfig.FileStoragePath, serverConfig.StoreInterval)
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
	//запуск обработчиков
	handlerv := handler.NewHandler(savedStorage)
	sugar.Infow(
		"handler init",
		"handler obj", handlerv,
	)
	handlerv.SetDBString(serverConfig.DatabaseDSN)
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
	savedStorage.Save()

}
