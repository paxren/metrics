package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/paxren/metrics/internal/models"

	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

func Send(memStorage *models.MemStorage) {

	client := http.Client{}

	for k, v := range memStorage.GetGauges() {
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/gauge/"+k+"/"+strconv.FormatFloat(v, 'f', 2, 64), nil)
		if err != nil {
			panic(err)
		}
		request.Header.Set(`Content-Type`, `text/plain`)
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, response.Body) // вывод ответа в консоль
		response.Body.Close()

	}

	for k, v := range memStorage.GetCounters() {
		request, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/counter/"+k+"/"+strconv.FormatInt(v, 10), nil)
		if err != nil {
			panic(err)
		}
		request.Header.Set(`Content-Type`, `text/plain`)
		response, err := client.Do(request)
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, response.Body) // вывод ответа в консоль
		response.Body.Close()
	}
	//для каунтера
}

func Add(memStorage *models.MemStorage, memStats *runtime.MemStats) {

	memStorage.UpdateGauge("Alloc", float64(memStats.Alloc))
	memStorage.UpdateGauge("BuckHashSys", float64(memStats.BuckHashSys))
	memStorage.UpdateGauge("Frees", float64(memStats.Frees))
	memStorage.UpdateGauge("GCCPUFraction", float64(memStats.GCCPUFraction))
	memStorage.UpdateGauge("GCSys", float64(memStats.GCSys))
	memStorage.UpdateGauge("HeapAlloc", float64(memStats.HeapAlloc))
	memStorage.UpdateGauge("HeapIdle", float64(memStats.HeapIdle))
	memStorage.UpdateGauge("HeapInuse", float64(memStats.HeapInuse))
	memStorage.UpdateGauge("HeapObjects", float64(memStats.HeapObjects))
	memStorage.UpdateGauge("HeapReleased", float64(memStats.HeapReleased))

	memStorage.UpdateGauge("HeapSys", float64(memStats.HeapSys))
	memStorage.UpdateGauge("LastGC", float64(memStats.LastGC))
	memStorage.UpdateGauge("Lookups", float64(memStats.Lookups))
	memStorage.UpdateGauge("MCacheInuse", float64(memStats.MCacheInuse))
	memStorage.UpdateGauge("MCacheSys", float64(memStats.MCacheSys))
	memStorage.UpdateGauge("MSpanInuse", float64(memStats.MSpanInuse))
	memStorage.UpdateGauge("MSpanSys", float64(memStats.MSpanSys))
	memStorage.UpdateGauge("Mallocs", float64(memStats.Mallocs))
	memStorage.UpdateGauge("NextGC", float64(memStats.NextGC))

	memStorage.UpdateGauge("NumForcedGC", float64(memStats.NumForcedGC))
	memStorage.UpdateGauge("NumGC", float64(memStats.NumGC))
	memStorage.UpdateGauge("OtherSys", float64(memStats.OtherSys))
	memStorage.UpdateGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	memStorage.UpdateGauge("StackInuse", float64(memStats.StackInuse))
	memStorage.UpdateGauge("StackSys", float64(memStats.StackSys))
	memStorage.UpdateGauge("Sys", float64(memStats.Sys))
	memStorage.UpdateGauge("TotalAlloc", float64(memStats.TotalAlloc))

}

func main() {

	var memStats runtime.MemStats

	memStorage := models.MakeMemStorage()

	var PollCount int64
	var randFloat float64
	//var test int64

	pollTicker := time.NewTicker(2000 * time.Millisecond)
	reportTicker := time.NewTicker(10 * time.Second)

	for {

		select {
		case <-pollTicker.C:
			fmt.Println("собираю данные")
			runtime.ReadMemStats(&memStats)

			PollCount++

			memStorage.UpdateCounter("PollCount", PollCount)
			//test, _ = memStorage.GetCounter("PollCount")
			randFloat = rand.Float64()
			memStorage.UpdateGauge("RandomValue", randFloat)
			//fmt.Printf("memstorage: %v \r\n", memStorage)
			//fmt.Printf("memstorage: %v \r\n", test)
			Add(memStorage, &memStats)
		case <-reportTicker.C:
			fmt.Println("отправляю данные")
			Send(memStorage)
			//memStorage := models.MakeMemStorage()

		}

	}

	//fmt.Println(ms.Alloc)
	//fmt.Println(ms1)

}
