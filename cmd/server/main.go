package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/repository"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

//var hostAdress = config.NewHostAddress()

var (
	hostAdress             = config.NewHostAddress()
	storeInterval   uint64 = 300
	fileStoragePath string = "save_file"
	restore         bool   = false

	paramHostAdress             = config.NewHostAddress()
	paramStoreInterval   uint64 = 300
	paramFileStoragePath string = "save_file"
	paramRestore         bool   = false
)

type ConfigSI struct {
	Val uint64 `env:"STORE_INTERVAL,required"`
}

type ConfigFSP struct {
	Val string `env:"FILE_STORAGE_PATH,required"`
}

type ConfigRe struct {
	Val bool `env:"RESTORE,required"`
}

func init() {
	// используем init-функцию
	flag.Var(paramHostAdress, "a", "Net address host:port")
	flag.Uint64Var(&paramStoreInterval, "i", 300, "storeInterval")
	flag.StringVar(&paramFileStoragePath, "f", "save_file", "fileStoragePath")
	flag.BoolVar(&paramRestore, "r", false, "paramRestore")
}

func main() {

	//обработка сигтерм
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
	flag.Parse()

	adr := os.Getenv("ADDRESS")

	err = hostAdress.Set(adr)

	if err != nil {
		sugar.Infow(
			"Failed to set address from env",
			"error", err,
			"adr", adr,
		)
		hostAdress = paramHostAdress
	}
	sugar.Infow(
		"host adress initialise",
		"hostAdressParams", paramHostAdress,
		"hostAdressEnv", adr,
		"hostAdressInit", hostAdress,
	)

	var si ConfigSI
	err = env.Parse(&si)
	if err != nil {
		sugar.Infow(
			"Failed to set store interval from env",
			"error", err,
		)
		storeInterval = paramStoreInterval
	} else {
		storeInterval = si.Val
	}
	sugar.Infow(
		"store interval  initialise",
		"storeIntervalParams", paramStoreInterval,
		"storeIntervalEnv", si.Val,
		"storeIntervalInit", storeInterval,
	)

	var fsp ConfigFSP
	err = env.Parse(&fsp)
	if err != nil {
		sugar.Infow(
			"Failed to set file storage path from env",
			"error", err,
		)
		fileStoragePath = paramFileStoragePath
	} else {
		fileStoragePath = fsp.Val
	}
	sugar.Infow(
		"file store path initialise",
		"fileStoragePathParams", paramFileStoragePath,
		"fileStoragePathEnv", fsp.Val,
		"fileStoragePathInit", fileStoragePath,
	)

	var re ConfigRe
	err = env.Parse(&re)
	if err != nil {
		sugar.Infow(
			"Failed to set restore from env",
			"error", err,
		)
		restore = paramRestore
	} else {
		restore = re.Val
	}
	sugar.Infow(
		"restore initialise",
		"restoreParams", paramRestore,
		"restoreEnv", re.Val,
		"restoreInit", restore,
	)
	// PREPARE STORAGES
	storage := repository.MakeMemStorage()
	//работа с файлами
	savedStorage := repository.MakeSavedRepo(storage, fileStoragePath, storeInterval)

	if restore {
		_ = savedStorage.Load(fileStoragePath)
		// if err != nil {
		// 	panic(err)
		// }
	}
	//запуск обработчиков
	handlerv := handler.NewHandler(savedStorage)
	//fmt.Printf("host param: %s", hostAdress.String())

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
	r.Post(`/value/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Post(`/value`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Get(`/value/{metric_type}/{metric_name}`, hlog.WithLogging(handlerv.GetMetric))
	r.Get(`/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetMain)))

	server := &http.Server{
		Addr:    hostAdress.String(),
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
