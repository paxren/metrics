package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/paxren/metrics/internal/models"

	"github.com/go-chi/chi/v5"
)

// type Metrics interface {
// 	UpdateGauge(string,float64) error
// 	UpdateCounter(string,int64) error
// }

// type Metric struct {
// 	counter []int64
// 	gauge   float64
// }

// ПОТОКО НЕБЕЗОПАСНО!
var memStorage *models.MemStorage = models.MakeMemStorage()

// type MemStorage struct {
// 	counters map[string][]int64
// 	gauges   map[string]float64
// }

// func (m *MemStorage) UpdateGauge(key string, value float64) error {

// 	m.gauges[key] = value
// 	return nil
// }

// func (m *MemStorage) UpdateCounter(key string, value int64) error {

// 	c, ok := m.counters[key]

// 	if !ok {
// 		c = make([]int64, 0)

// 	}

// 	c = append(c, value)

// 	m.counters[key] = c

// 	return nil
// }

func updateMetric(res http.ResponseWriter, req *http.Request) {
	//res.Write([]byte("Привет!"))
	//fmt.Println("run update")
	if req.Method != http.MethodPost {
		// разрешаем только POST-запросы
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	//TODO проверка на наличие Content-Type: text/plain

	//	req.URL
	elems := strings.Split(req.URL.Path, "/")

	if len(elems) != 5 {
		http.Error(res, fmt.Sprintf("неверное количество параметров: %v, все элементы: %v \r\n", len(elems), elems), http.StatusNotFound)
		return
	}

	typeE := elems[2]
	nameE := elems[3]
	valueE := elems[4]

	if !(typeE == "counter" || typeE == "gauge") {
		http.Error(res, fmt.Sprintf("Некорректный тип метрики: %v, все элементы: %v \r\n", typeE, elems), http.StatusBadRequest)
		return
	}

	if nameE == "" {
		http.Error(res, fmt.Sprintf("Пустое имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
		return
	}

	switch typeE {
	case "counter":
		val, err := strconv.ParseInt(elems[4], 10, 64)
		if err != nil {
			http.Error(res, fmt.Sprintf("Некорректное значение метрики: %v, все элементы: %v \r\n", valueE, elems), http.StatusBadRequest)
			return
		}
		memStorage.UpdateCounter(nameE, val)
	case "gauge":
		val, err := strconv.ParseFloat(elems[4], 64)
		if err != nil {
			http.Error(res, fmt.Sprintf("Некорректное значение метрики: %v, все элементы: %v \r\n", valueE, elems), http.StatusBadRequest)
			return
		}
		memStorage.UpdateGauge(nameE, val)
	}

	res.Write([]byte(fmt.Sprintf("elems: %v memStorage: %v \r\n", elems, memStorage)))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL)
}

func getMetric(res http.ResponseWriter, req *http.Request) {
	//res.Write([]byte("Привет!"))
	//fmt.Println("run get")
	// if req.Method != http.MethodGet {
	// 	// разрешаем только POST-запросы
	// 	res.WriteHeader(http.StatusMethodNotAllowed)
	// 	return
	// }

	//TODO проверка на наличие Content-Type: text/plain

	//	req.URL
	elems := strings.Split(req.URL.Path, "/")

	if len(elems) != 4 {
		http.Error(res, fmt.Sprintf("неверное количество параметров: %v, все элементы: %v \r\n", len(elems), elems), http.StatusNotFound)
		return
	}

	typeE := chi.URLParam(req, "metric_type")
	nameE := chi.URLParam(req, "metric_name")
	var stringValue string

	if !(typeE == "counter" || typeE == "gauge") {
		http.Error(res, fmt.Sprintf("Некорректный тип метрики: %v, все элементы: %v \r\n", typeE, elems), http.StatusBadRequest)
		return
	}

	if nameE == "" {
		http.Error(res, fmt.Sprintf("Пустое имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
		return
	}

	switch typeE {
	case "counter":
		v, err := memStorage.GetCounter(nameE)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
			return
		}

		stringValue = strconv.FormatInt(v, 10)
	case "gauge":
		v, err := memStorage.GetGauge(nameE)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
			return
		}

		stringValue = strconv.FormatFloat(v, 'f', 3, 64)
	}

	res.Write([]byte(stringValue))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL, stringValue)
}

func getMain(res http.ResponseWriter, req *http.Request) {
	const formStart = `<html>
<head>
<title>Известные метрики:</title>
    </head>
    <body>
	`

	//<label>Логин <input type="text" name="login"></label>
	//<label>Пароль <input type="password" name="password"></label>

	const formEnd = `
    </body>
</html>`

	var formMetrics = `<label>Метрики gauges:</label><br/>`
	gauges := memStorage.GetGauges()

	for k, v := range gauges {
		formMetrics += fmt.Sprintf(`<label>%s = %f</label><br/>`, k, v)
	}

	formMetrics += `<label>Метрики counters:</label><br/>`
	counters := memStorage.GetCounters()

	for k, v := range counters {
		formMetrics += fmt.Sprintf(`<label>%s = %d</label><br/>`, k, v)
	}

	var form = formStart + formMetrics + formEnd

	res.Write([]byte(form))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL)
}

func main() {

	r := chi.NewRouter()

	r.Post(`/update/{metric_type}/{metric_name}/{metric_value}`, updateMetric)
	r.Get(`/value/{metric_type}/{metric_name}`, getMetric)
	r.Get(`/`, getMain)

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}

}
