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
)

var hostAdress = config.NewHostAddress()

func init() {
	// используем init-функцию
	flag.Var(hostAdress, "a", "Net address host:port")
}

func main() {

	adr := os.Getenv("ADDRESS")

	err1 := hostAdress.Set(adr)

	if err1 != nil {
		flag.Parse()
	}

	fmt.Println(hostAdress)

	handler := handler.NewHandler(repository.MakeMemStorage())
	//fmt.Printf("host param: %s", hostAdress.String())

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, handler.UpdateMetric)
	r.Get(`/value/{metric_type}/{metric_name}`, handler.GetMetric)
	r.Get(`/`, handler.GetMain)

	err := http.ListenAndServe(hostAdress.String(), r)
	if err != nil {
		panic(err)
	}

}
