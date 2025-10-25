package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo repository.Repository
}

func NewHandler(r repository.Repository) *Handler {
	return &Handler{
		repo: r,
	}
}

func (h Handler) UpdateMetric(res http.ResponseWriter, req *http.Request) {
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
		h.repo.UpdateCounter(nameE, val)
	case "gauge":
		val, err := strconv.ParseFloat(elems[4], 64)
		if err != nil {
			http.Error(res, fmt.Sprintf("Некорректное значение метрики: %v, все элементы: %v \r\n", valueE, elems), http.StatusBadRequest)
			return
		}
		h.repo.UpdateGauge(nameE, val)
	}

	res.Write([]byte(fmt.Sprintf("elems: %v repo: %v \r\n", elems, h.repo)))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL)
}

func (h Handler) GetMetric(res http.ResponseWriter, req *http.Request) {
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
		v, err := h.repo.GetCounter(nameE)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
			return
		}

		stringValue = strconv.FormatInt(v, 10)
	case "gauge":
		v, err := h.repo.GetGauge(nameE)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v, все элементы: %v \r\n", nameE, elems), http.StatusNotFound)
			return
		}

		stringValue = strconv.FormatFloat(v, 'f', -1, 64)
	}

	res.Write([]byte(stringValue))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL, stringValue)
}

func (h Handler) GetMain(res http.ResponseWriter, req *http.Request) {
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
	gaugesKeys := h.repo.GetGaugesKeys()

	for _, vkey := range gaugesKeys {
		vv, err := h.repo.GetGauge(vkey)
		if err == nil {
			formMetrics += fmt.Sprintf(`<label>%s = %f</label><br/>`, vkey, vv)
		} else {
			formMetrics += fmt.Sprintf(`<label>%s = READ ERROR</label><br/>`, vkey)
		}
	}

	formMetrics += `<label>Метрики counters:</label><br/>`
	countersKeys := h.repo.GetCountersKeys()

	for _, vkey := range countersKeys {
		vv, err := h.repo.GetCounter(vkey)
		if err == nil {
			formMetrics += fmt.Sprintf(`<label>%s = %d</label><br/>`, vkey, vv)
		} else {
			formMetrics += fmt.Sprintf(`<label>%s = READ ERROR</label><br/>`, vkey)
		}
	}

	var form = formStart + formMetrics + formEnd

	res.Write([]byte(form))
	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	fmt.Println(req.URL)
}
