package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/paxren/metrics/internal/audit"
	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/crypto"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"net/http/pprof"
	//_ "net/http/pprof"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
	serverConfig = config.NewServerConfig()
)

func init() {
	serverConfig.Init()
}

func main() {
	// Выводим информацию о сборке
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

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
		var pstorage *repository.PostgresStorageWithRetry
		pstorage, err = repository.MakePostgresStorageWithRetry(serverConfig.DatabaseDSN)

		if err != nil {
			// вызываем панику, если ошибка
			sugar.Fatal(
				"Storage init1",
				"Storage obj", pstorage,
				"err", pstorage,
			)
			//panic("cannot initialize postgress")
		}
		var mutexed repository.Repository
		mutexed, err = repository.MakeMutexedRegistry(pstorage)
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

	// Создаём дешифратор, если указан путь к приватному ключу
	var cryptoMiddleware *handler.CryptoMiddleware
	if serverConfig.CryptoKey != "" {
		hybridDecryptor, err := crypto.NewHybridDecryptor(serverConfig.CryptoKey)
		if err != nil {
			sugar.Fatal(
				"Failed to load private key",
				"error", err,
			)
		}
		// Создаем адаптер для совместимости с существующим middleware
		decryptor := crypto.NewHybridDecryptorAdapter(hybridDecryptor)
		cryptoMiddleware = handler.NewCryptoMiddleware(decryptor)
		sugar.Infow(
			"Hybrid crypto middleware enabled",
			"key", serverConfig.CryptoKey,
		)
	}

	// Создаём менеджер сжатия
	compressionConfig, err := handler.ParseCompressionConfig()
	if err != nil {
		// Используем конфигурацию по умолчанию при ошибке
		compressionConfig = &handler.CompressionConfig{
			EnableCompression: true,
			CompressionLevel:  6,
			MinContentSize:    1024,
		}
	}
	compressionManager := handler.NewCompressionManager(compressionConfig)
	compressor := handler.NewCompressor(compressionManager)

	// Создаём наблюдателей для аудита
	var auditObservers []audit.Observer

	// Создаём наблюдателя для файла, если указан путь
	if serverConfig.AuditFile != "" {
		fileObserver := audit.NewFileObserverWithBufferSize(serverConfig.AuditFile, 1000)
		auditObservers = append(auditObservers, fileObserver)
	}

	// Создаём наблюдателя для URL, если указан URL
	if serverConfig.AuditURL != "" {
		urlObserver := audit.NewURLObserverWithBufferSize(serverConfig.AuditURL, 500)
		auditObservers = append(auditObservers, urlObserver)
	}

	// Создаём аудитор
	auditor := handler.NewAuditor(auditObservers)

	// Добавляем функцию завершения работы аудита в массив finish
	if auditor != nil {
		finish = append(finish, auditor.Close)
	}

	r := chi.NewRouter()

	// Применяем middleware ко всем эндпоинтам обновления метрик
	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(auditor.WithAudit(handlerv.UpdateMetric)))

	// Для JSON эндпоинтов применяем crypto middleware, если он настроен
	if cryptoMiddleware != nil {
		r.Post(`/update/`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(cryptoMiddleware.DecryptMiddleware(handlerv.UpdateJSON))))))
		r.Post(`/update`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(cryptoMiddleware.DecryptMiddleware(handlerv.UpdateJSON))))))
		r.Post(`/updates`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(cryptoMiddleware.DecryptMiddleware(handlerv.UpdatesJSON))))))
		r.Post(`/updates/`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(cryptoMiddleware.DecryptMiddleware(handlerv.UpdatesJSON))))))
	} else {
		r.Post(`/update/`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(compressor.OptimizedGzipMiddleware(handlerv.UpdateJSON)))))
		r.Post(`/update`, hasher.HashMiddleware(hlog.WithLogging(auditor.WithAudit(compressor.OptimizedGzipMiddleware(handlerv.UpdateJSON)))))
		r.Post(`/updates`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(handlerv.UpdatesJSON)))))
		r.Post(`/updates/`, hlog.WithLogging(auditor.WithAudit(hasher.HashMiddleware(compressor.OptimizedGzipMiddleware(handlerv.UpdatesJSON)))))
	}

	r.Post(`/value/`, hasher.HashMiddleware(hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetValueJSON))))
	r.Post(`/value`, hasher.HashMiddleware(hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetValueJSON))))
	r.Get(`/value/{metric_type}/{metric_name}`, hlog.WithLogging(handlerv.GetMetric))
	r.Get(`/ping`, hlog.WithLogging(handlerv.PingDB))
	r.Get(`/ping/`, hlog.WithLogging(handlerv.PingDB))
	r.Get(`/`, hlog.WithLogging(compressor.OptimizedGzipMiddleware(handlerv.GetMain)))

	// Добавляем эндпоинты для pprof
	//r.Handle("/debug/pprof/*", http.HandlerFunc(pprof.Index))
	// Замените строку 186 на эти:
	r.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Handler("heap").ServeHTTP))
	r.Handle("/debug/pprof/goroutine", http.HandlerFunc(pprof.Handler("goroutine").ServeHTTP))
	r.Handle("/debug/pprof/block", http.HandlerFunc(pprof.Handler("block").ServeHTTP))
	r.Handle("/debug/pprof/mutex", http.HandlerFunc(pprof.Handler("mutex").ServeHTTP))
	r.Handle("/debug/pprof/allocs", http.HandlerFunc(pprof.Handler("allocs").ServeHTTP))

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
