package agent

import (
	"runtime"
	"strings"

	"github.com/paxren/metrics/internal/config"
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
	Repo repository.Repository
	host config.HostAddress
}

func NewAgent(r repository.Repository, host config.HostAddress) *Agent {
	return &Agent{
		Repo: r,
		host: host,
	}
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
			metricJSON, err := json.Marshal(metricOut)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			var gzipped bytes.Buffer
			// создаём переменную w — в неё будут записываться входящие данные,
			// которые будут сжиматься и сохраняться в bytes.Buffer
			w := gzip.NewWriter(&gzipped)

			_, err = w.Write(metricJSON)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			err = w.Close()
			if err != nil {
				errors = append(errors, err)
				continue
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
				continue
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
					continue
				}
				defer r.Close()

				var b bytes.Buffer
				// в переменную b записываются распакованные данные
				_, err = b.ReadFrom(r)
				if err != nil {
					errors = append(errors, err)
					continue
				}

				io.Copy(os.Stdout, &b) // вывод ответа в консоль
				//fmt.Println(1)
			} //else {
			//TODO сжатый другим методом или несжатый
			//io.Copy(os.Stdout, response.Body)
			//}
			//response.Body.Close()
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
			metricJSON, err := json.Marshal(metricOut)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			var gzipped bytes.Buffer
			// создаём переменную w — в неё будут записываться входящие данные,
			// которые будут сжиматься и сохраняться в bytes.Buffer
			w := gzip.NewWriter(&gzipped)

			_, err = w.Write(metricJSON)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			err = w.Close()
			if err != nil {
				errors = append(errors, err)
				continue
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
				continue
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
					continue
				}
				defer r.Close()

				var b bytes.Buffer
				// в переменную b записываются распакованные данные
				_, err = b.ReadFrom(r)
				if err != nil {
					errors = append(errors, err)
					continue
				}

				io.Copy(os.Stdout, &b) // вывод ответа в консоль
				//fmt.Println(1)
			} //else {
			//TODO сжатый другим методом или несжатый
			//io.Copy(os.Stdout, response.Body)
			//}
			//response.Body.Close()
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
