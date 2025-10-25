package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/repository"

	"math/rand"
)

var (
	hostAdress           = config.NewHostAddress()
	reportInterval int64 = 10
	pollInterval   int64 = 2
)

func init() {
	// используем init-функцию
	flag.Var(hostAdress, "a", "Net address host:port")
	flag.Int64Var(&reportInterval, "r", 10, "reportInterval")
	flag.Int64Var(&pollInterval, "p", 2, "pollInterval")
}

func main() {

	flag.Parse()

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
