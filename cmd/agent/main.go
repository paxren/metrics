package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/paxren/metrics/internal/agent"
	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/repository"

	"github.com/caarlos0/env/v11"
)

var (
	hostAdress            = config.NewHostAddress()
	reportInterval int64  = 10
	pollInterval   int64  = 2
	rateLimit      int64  = 1
	key            string = ""

	paramHostAdress            = config.NewHostAddress()
	paramReportInterval int64  = 10
	paramPollInterval   int64  = 2
	paramRateLimit      int64  = 1
	paramKey            string = ""
)

type ConfigRI struct {
	Val int64 `env:"REPORT_INTERVAL,required"`
}

type ConfigPI struct {
	Val int64 `env:"POLL_INTERVAL,required"`
}

type ConfigRM struct {
	Val int64 `env:"RATE_LIMIT,required"`
}

type ConfigKey struct {
	Val string `env:"KEY,required"`
}

func init() {
	// используем init-функцию
	flag.Var(paramHostAdress, "a", "Net address host:port")
	flag.Int64Var(&paramReportInterval, "r", 10, "reportInterval")
	flag.Int64Var(&paramPollInterval, "p", 2, "pollInterval")
	flag.Int64Var(&paramRateLimit, "l", 1, "rateLimit")
	flag.StringVar(&paramKey, "k", "", "pollInterval")
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

	var ri ConfigRI
	err2 := env.Parse(&ri)
	fmt.Printf("ri=%v  err=%v \n", ri, err2)
	if err2 != nil {
		//fmt.Printf("Error parsing REPORT_INTERVAL1: %v, using default value\n", err2)
		reportInterval = paramReportInterval
	} else {
		//fmt.Printf("Successfully parsed REPORT_INTERVAL1: %d\n", ri.Val)
		reportInterval = ri.Val
	}

	// Убираем os.Exit(1), чтобы программа продолжала выполнение
	// os.Exit(1)

	var pi ConfigPI
	err3 := env.Parse(&pi)
	//fmt.Printf("POLL_INTERVAL from os.Getenv: %s\n", os.Getenv("POLL_INTERVAL"))
	fmt.Printf("pi=%v  err=%v \n", pi, err3)
	if err3 != nil {
		//fmt.Printf("Error parsing POLL_INTERVAL: %v, using default value\n", err3)
		pollInterval = paramPollInterval
	} else {
		//fmt.Printf("Successfully parsed POLL_INTERVAL: %d\n", pi.Val)
		pollInterval = pi.Val
	}

	var testKey ConfigKey
	err4 := env.Parse(&testKey)
	//fmt.Printf("POLL_INTERVAL from os.Getenv: %s\n", os.Getenv("POLL_INTERVAL"))
	fmt.Printf("key=%v  err=%v \n", testKey, err4)
	if err4 != nil {
		//fmt.Printf("Error parsing POLL_INTERVAL: %v, using default value\n", err3)
		key = paramKey
	} else {
		//fmt.Printf("Successfully parsed POLL_INTERVAL: %d\n", pi.Val)
		key = testKey.Val
	}

	var testRM ConfigRM
	err5 := env.Parse(&testRM)
	//fmt.Printf("POLL_INTERVAL from os.Getenv: %s\n", os.Getenv("POLL_INTERVAL"))
	fmt.Printf("key=%v  err=%v \n", testRM, err5)
	if err5 != nil {
		//fmt.Printf("Error parsing POLL_INTERVAL: %v, using default value\n", err3)
		rateLimit = paramRateLimit
	} else {
		//fmt.Printf("Successfully parsed POLL_INTERVAL: %d\n", pi.Val)
		rateLimit = testRM.Val
	}
	//======== START

	fmt.Printf("report interval: %d \r\n poll interval: %d \r\n", reportInterval, pollInterval)
	fmt.Printf("rate limiter: %d \r\n key: %s \r\n", rateLimit, key)

	//var memStats runtime.MemStats

	agent := agent.NewAgentExtended(repository.MakeMemStorage(), *hostAdress, key, rateLimit, pollInterval, reportInterval)

	agent.Start()
	// var PollCount int64
	// var randFloat float64
	//var test int64

	//pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	//reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)

	// for {

	// 	select {
	// 	case <-pollTicker.C:
	// 		fmt.Println("собираю данные")
	// 		runtime.ReadMemStats(&memStats)

	// 		PollCount++

	// 		agent.Repo.UpdateCounter("PollCount", PollCount)
	// 		//test1, _ = agent.Repo.GetCounter("PollCount")
	// 		randFloat = rand.Float64()
	// 		agent.Repo.UpdateGauge("RandomValue", randFloat)
	// 		//fmt.Printf("memstorage: %v \r\n", agent.Repo)
	// 		//fmt.Printf("memstorage: %v \r\n", test1)
	// 		agent.Add(&memStats)
	// 	case <-reportTicker.C:
	// 		fmt.Println("отправляю данные")
	// 		agent.SendAll()
	// 		//memStorage := models.MakeMemStorage()

	// 	}

	// }

	//fmt.Println(ms.Alloc)
	//fmt.Println(ms1)

}
