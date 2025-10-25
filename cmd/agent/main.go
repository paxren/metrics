package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/repository"

	"math/rand"

	"github.com/caarlos0/env/v11"
)

var (
	hostAdress           = config.NewHostAddress()
	reportInterval int64 = 10
	pollInterval   int64 = 2

	paramHostAdress           = config.NewHostAddress()
	paramReportInterval int64 = 10
	paramPollInterval   int64 = 2
)

type ConfigRI struct {
	val string `env:"REPORT_INTERVAL1,required"`
}

type ConfigPI struct {
	val int64 `env:"POLL_INTERVAL"`
}

func init() {
	// используем init-функцию
	flag.Var(paramHostAdress, "a", "Net address host:port")
	flag.Int64Var(&paramReportInterval, "r", 10, "reportInterval")
	flag.Int64Var(&paramPollInterval, "p", 2, "pollInterval")
}

func main() {

	flag.Parse()
	//========== ADDRESS
	adr := os.Getenv("ADDRESS")

	err1 := hostAdress.Set(adr)
	if err1 != nil {
		hostAdress = paramHostAdress
	}

	fmt.Println(hostAdress)

	//========= INTERVASLS
	mp := env.ToMap(os.Environ())
	fmt.Printf("mp=%v  \n", mp)

	var ri ConfigRI
	err2 := env.Parse(&ri)
	fmt.Printf("ri=%v  err=%v \n", ri, err2)
	if err2 != nil {
		reportInterval = paramReportInterval
	} else {
		reportInterval = 1 //ri.val
	}

	os.Exit(1)

	var pi ConfigPI
	err3 := env.Parse(&pi)
	fmt.Println(pi)
	if err3 != nil {
		pollInterval = paramPollInterval
	} else {
		pollInterval = pi.val
	}
	//======== START

	fmt.Printf("report interval: %d \r\n poll interval: %d \r\n", reportInterval, pollInterval)

	var memStats runtime.MemStats

	agent := agent.NewAgent(repository.MakeMemStorage(), *hostAdress)

	var PollCount int64
	var randFloat float64
	//var test int64

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)

	for {

		select {
		case <-pollTicker.C:
			fmt.Println("собираю данные")
			runtime.ReadMemStats(&memStats)

			PollCount++

			agent.Repo.UpdateCounter("PollCount", PollCount)
			//test, _ = memStorage.GetCounter("PollCount")
			randFloat = rand.Float64()
			agent.Repo.UpdateGauge("RandomValue", randFloat)
			//fmt.Printf("memstorage: %v \r\n", memStorage)
			//fmt.Printf("memstorage: %v \r\n", test)
			agent.Add(&memStats)
		case <-reportTicker.C:
			fmt.Println("отправляю данные")
			agent.Send()
			//memStorage := models.MakeMemStorage()

		}

	}

	//fmt.Println(ms.Alloc)
	//fmt.Println(ms1)

}
