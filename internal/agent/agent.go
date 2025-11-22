package agent

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/hash"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"

	"io"
	"net/http"
	"os"

	"bytes"
	"compress/gzip"
	"encoding/json"
)

type Agent struct {
	Repo         repository.Repository
	host         config.HostAddress
	hashKey      string
	hashKeyBytes []byte
}

func NewAgent(r repository.Repository, host config.HostAddress) *Agent {
	return &Agent{
		Repo:         r,
		host:         host,
		hashKey:      "",
		hashKeyBytes: nil,
	}
}

func NewAgentWithKey(r repository.Repository, host config.HostAddress, key string) *Agent {

	var hashKeyBytes []byte = nil
	if key != "" {
		hashKeyBytes = []byte(key)
	}

	fmt.Printf("hashBytes %s %v\n", hashKeyBytes, hashKeyBytes)
	return &Agent{
		Repo:         r,
		host:         host,
		hashKey:      key,
		hashKeyBytes: hashKeyBytes,
	}
}

func (a Agent) makeMetrics() ([]models.Metrics, []error) {

	errors := make([]error, 0)
	metrics := make([]models.Metrics, 0, 10)

	gaugesKeys := a.Repo.GetGaugesKeys()
	for _, vkey := range gaugesKeys {
		vv, err := a.Repo.GetGauge(vkey)

		if err == nil {
			metrics = append(metrics, models.Metrics{
				ID:    vkey,
				MType: models.Gauge,
				Value: &vv,
			})
		} else {
			errors = append(errors, err)
		}

	}

	countersKeys := a.Repo.GetCountersKeys()
	for _, vkey := range countersKeys {

		vv, err := a.Repo.GetCounter(vkey)

		if err == nil {
			metrics = append(metrics, models.Metrics{
				ID:    vkey,
				MType: models.Counter,
				Delta: &vv,
			})
		} else {
			errors = append(errors, err)
		}
	}

	return metrics, errors
}

func (a Agent) makeRequest(metrics []models.Metrics) (*http.Request, []error) {

	errors := make([]error, 0)
	//var request *http.Request
	metricJSON, err := json.Marshal(metrics)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	var gzipped bytes.Buffer
	// создаём переменную w — в неё будут записываться входящие данные,
	// которые будут сжиматься и сохраняться в bytes.Buffer
	w := gzip.NewWriter(&gzipped)

	_, err = w.Write(metricJSON)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	err = w.Close()
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	request, err := http.NewRequest(http.MethodPost, "http://"+a.host.String()+"/updates", &gzipped)
	if err != nil {
		errors = append(errors, err)
	}
	request.Header.Set(`Content-Type`, `application/json`)
	request.Header.Set(`Accept-Encoding`, `gzip`)
	request.Header.Set(`Content-Encoding`, `gzip`)

	fmt.Println("a1")
	if a.hashKeyBytes != nil {
		fmt.Println("a2")
		src := make([]byte, gzipped.Len())
		copy(src, gzipped.Bytes())
		hash, err := hash.MakeHash(&a.hashKeyBytes, &src)
		if err == nil {
			fmt.Println(hash)
			request.Header.Set(`HashSHA256`, hash)
		}
	}

	return request, errors

}

func (a Agent) SendAll() []error {

	errors := make([]error, 0)

	client := http.Client{}

	metrics, errors1 := a.makeMetrics()

	errors = append(errors, errors1...)

	request, errors2 := a.makeRequest(metrics)
	errors = append(errors, errors2...)

	const maxRetries = 3

	var waitSec int64 = 1
	var response *http.Response
	var success = false
	var attempt int64 = 0
	var err error
	//attempt := 0; attempt < maxRetries; attempt++

	for !success {

		response, err = client.Do(request)
		if err != nil {
			fmt.Printf("net err: %t %v\n", err, err)
			if attempt < maxRetries {
				fmt.Printf("жду...\n")
				errors = append(errors, err)
				time.Sleep(time.Duration(waitSec) * time.Second)
				waitSec += 2
			} else {
				fmt.Printf("не дождался...\n")
				return errors
			}
			attempt++

			//
		} else {
			success = true
			defer response.Body.Close()
		}

	}

	//response.Header.Get("Content-Encoding")
	contentEncoding := response.Header.Get("Content-Encoding")
	receiveGzip := strings.Contains(contentEncoding, "gzip")

	if receiveGzip {

		// переменная r будет читать входящие данные и распаковывать их
		r, err := gzip.NewReader(response.Body)
		if err != nil {
			errors = append(errors, err)
			return errors
		}
		defer r.Close()

		var b bytes.Buffer
		// в переменную b записываются распакованные данные
		_, err = b.ReadFrom(r)
		if err != nil {
			errors = append(errors, err)
			return errors
		}

		io.Copy(os.Stdout, &b) // вывод ответа в консоль
		//fmt.Println(1)
	} //else {
	//TODO сжатый другим методом или несжатый
	//io.Copy(os.Stdout, response.Body)
	//}
	//response.Body.Close()

	return errors

}

func (a Agent) work1(metricOut *models.Metrics, client *http.Client, errors []error) []error {
	metricJSON, err := json.Marshal(metricOut)
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	var gzipped bytes.Buffer
	// создаём переменную w — в неё будут записываться входящие данные,
	// которые будут сжиматься и сохраняться в bytes.Buffer
	w := gzip.NewWriter(&gzipped)

	_, err = w.Write(metricJSON)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	err = w.Close()
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	request, err := http.NewRequest(http.MethodPost, "http://"+a.host.String()+"/update", &gzipped)
	if err != nil {
		errors = append(errors, err)
	}
	request.Header.Set(`Content-Type`, `application/json`)
	request.Header.Set(`Accept-Encoding`, `gzip`)
	request.Header.Set(`Content-Encoding`, `gzip`)

	response, err := client.Do(request)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	defer response.Body.Close()

	//response.Header.Get("Content-Encoding")
	contentEncoding := response.Header.Get("Content-Encoding")
	receiveGzip := strings.Contains(contentEncoding, "gzip")

	if receiveGzip {

		// переменная r будет читать входящие данные и распаковывать их
		r, err := gzip.NewReader(response.Body)
		if err != nil {
			errors = append(errors, err)
			return errors
		}
		defer r.Close()

		var b bytes.Buffer
		// в переменную b записываются распакованные данные
		_, err = b.ReadFrom(r)
		if err != nil {
			errors = append(errors, err)
			return errors
		}

		io.Copy(os.Stdout, &b) // вывод ответа в консоль
		//fmt.Println(1)
	} //else {
	//TODO сжатый другим методом или несжатый
	//io.Copy(os.Stdout, response.Body)
	//}
	//response.Body.Close()

	return errors
}

func (a Agent) Send() []error {

	errors := make([]error, 0)

	client := http.Client{}

	gaugesKeys := a.Repo.GetGaugesKeys()

	var metricOut models.Metrics
	for _, vkey := range gaugesKeys {

		vv, err := a.Repo.GetGauge(vkey)

		if err == nil {

			metricOut.ID = vkey
			metricOut.Value = &vv
			metricOut.MType = models.Gauge
			errors = a.work1(&metricOut, &client, errors)
		} else {
			errors = append(errors, err)
		}

	}

	countersKeys := a.Repo.GetCountersKeys()
	for _, vkey := range countersKeys {

		vv, err := a.Repo.GetCounter(vkey)

		if err == nil {
			metricOut.ID = vkey
			metricOut.Delta = &vv
			metricOut.MType = models.Counter
			errors = a.work1(&metricOut, &client, errors)
		} else {
			errors = append(errors, err)
		}
	}

	return errors
	//для каунтера
}

func (a Agent) Add(memStats *runtime.MemStats) {

	a.Repo.UpdateGauge("Alloc", float64(memStats.Alloc))
	a.Repo.UpdateGauge("BuckHashSys", float64(memStats.BuckHashSys))
	a.Repo.UpdateGauge("Frees", float64(memStats.Frees))
	a.Repo.UpdateGauge("GCCPUFraction", float64(memStats.GCCPUFraction))
	a.Repo.UpdateGauge("GCSys", float64(memStats.GCSys))
	a.Repo.UpdateGauge("HeapAlloc", float64(memStats.HeapAlloc))
	a.Repo.UpdateGauge("HeapIdle", float64(memStats.HeapIdle))
	a.Repo.UpdateGauge("HeapInuse", float64(memStats.HeapInuse))
	a.Repo.UpdateGauge("HeapObjects", float64(memStats.HeapObjects))
	a.Repo.UpdateGauge("HeapReleased", float64(memStats.HeapReleased))

	a.Repo.UpdateGauge("HeapSys", float64(memStats.HeapSys))
	a.Repo.UpdateGauge("LastGC", float64(memStats.LastGC))
	a.Repo.UpdateGauge("Lookups", float64(memStats.Lookups))
	a.Repo.UpdateGauge("MCacheInuse", float64(memStats.MCacheInuse))
	a.Repo.UpdateGauge("MCacheSys", float64(memStats.MCacheSys))
	a.Repo.UpdateGauge("MSpanInuse", float64(memStats.MSpanInuse))
	a.Repo.UpdateGauge("MSpanSys", float64(memStats.MSpanSys))
	a.Repo.UpdateGauge("Mallocs", float64(memStats.Mallocs))
	a.Repo.UpdateGauge("NextGC", float64(memStats.NextGC))

	a.Repo.UpdateGauge("NumForcedGC", float64(memStats.NumForcedGC))
	a.Repo.UpdateGauge("NumGC", float64(memStats.NumGC))
	a.Repo.UpdateGauge("OtherSys", float64(memStats.OtherSys))
	a.Repo.UpdateGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	a.Repo.UpdateGauge("StackInuse", float64(memStats.StackInuse))
	a.Repo.UpdateGauge("StackSys", float64(memStats.StackSys))
	a.Repo.UpdateGauge("Sys", float64(memStats.Sys))
	a.Repo.UpdateGauge("TotalAlloc", float64(memStats.TotalAlloc))

}
