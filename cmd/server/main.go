package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var hostAdress = config.NewHostAddress()

func init() {
	// используем init-функцию
	flag.Var(hostAdress, "a", "Net address host:port")
}

func main() {

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
	adr := os.Getenv("ADDRESS")

	err = hostAdress.Set(adr)

	if err != nil {
		sugar.Infow(
			"Failed to set address",
			"error", err,
			"adr", adr,
		)
		flag.Parse()
	}

	fmt.Println(hostAdress)

	handlerv := handler.NewHandler(repository.MakeMemStorage())
	//fmt.Printf("host param: %s", hostAdress.String())

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, hlog.WithLogging(handlerv.UpdateMetric))
	r.Post(`/value/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Post(`/value`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetValueJSON)))
	r.Post(`/update`, hlog.WithLogging(handler.GzipMiddleware(handlerv.UpdateJSON)))
	r.Get(`/value/{metric_type}/{metric_name}`, hlog.WithLogging(handlerv.GetMetric))
	r.Get(`/`, hlog.WithLogging(handler.GzipMiddleware(handlerv.GetMain)))

	err = http.ListenAndServe(hostAdress.String(), r)
	if err != nil {
		sugar.Infow(
			"Failed to serve listener",
			"error", err,
		)
		panic(err)
	}

}
