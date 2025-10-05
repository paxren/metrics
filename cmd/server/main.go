package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
var memStorage MemStorage

type MemStorage struct {
	counters map[string][]int64
	gauges   map[string]float64
}

func (m *MemStorage) UpdateGauge(key string, value float64) error {

	m.gauges[key] = value
	return nil
}

func (m *MemStorage) UpdateCounter(key string, value int64) error {

	c, ok := m.counters[key]

	if !ok {
		c = make([]int64, 0)

	}

	c = append(c, value)

	m.counters[key] = c

	return nil
}

func updateMetric(res http.ResponseWriter, req *http.Request) {
	//res.Write([]byte("Привет!"))

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

}

func main() {

	memStorage = MemStorage{
		counters: make(map[string][]int64),
		gauges:   make(map[string]float64),
	}

	mux := http.NewServeMux()

	mux.HandleFunc(`/update/`, updateMetric)

	//fmt.Println(memStorage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}

}
