package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"encoding/json"

	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"

	"github.com/go-chi/chi/v5"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Handler представляет обработчик HTTP-запросов для работы с метриками.
//
// Предоставляет методы для обновления, получения и отображения метрик
// различных типов (gauge и counter) через HTTP-интерфейс.
type Handler struct {
	repo repository.Repository

	//todo переделать!!!
	dbConnectionString string
}

// NewHandler создаёт новый экземпляр обработчика метрик.
//
// Параметры:
//   - r: реализация интерфейса Repository для хранения метрик
//
// Возвращает:
//   - *Handler: указатель на созданный обработчик
func NewHandler(r repository.Repository) *Handler {
	return &Handler{
		repo: r,
	}
}

// func (h *Handler) SetDBString(str string) {
// 	// fmt.Printf("перед присваиванием h.dbConnectionString %s \n", h.dbConnectionString)
// 	// fmt.Printf("перед присваиванием str %s\n", str)
// 	h.dbConnectionString = str
// 	// fmt.Printf("после присваивания h.dbConnectionString %s \n", h.dbConnectionString)
// }

// UpdateMetric обрабатывает запрос на обновление метрики через URL.
//
// Поддерживает формат URL: /update/{metric_type}/{metric_name}/{metric_value}
// где metric_type - "gauge" или "counter", metric_value - числовое значение.
//
// Принимает только POST-запросы.
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

	// Убираем вывод в консоль для корректной работы тестов
	// fmt.Println(req.URL)
}

// GetMetric обрабатывает запрос на получение значения метрики через URL.
//
// Поддерживает формат URL: /value/{metric_type}/{metric_name}
// где metric_type - "gauge" или "counter".
//
// Возвращает значение метрики в виде текста.
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

	// Убираем вывод в консоль для корректной работы тестов
	// fmt.Println(req.URL, stringValue)
}

// GetMain обрабатывает запрос на получение главной страницы со всеми метриками.
//
// Возвращает HTML-страницу с таблицей всех сохранённых метрик.
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

	res.Header().Set("Content-Type", "text/html ; charset=utf-8")
	//res.Header().Set("Content-Type", "")

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(form))

	//res.Write([]byte(fmt.Sprintf("len %v \r\n", len(elems))))

	// Убираем вывод в консоль для корректной работы тестов
	// fmt.Println(req.URL)
}

// UpdateJSON обрабатывает запрос на обновление одной метрики через JSON.
//
// Принимает JSON-объект метрики в теле POST-запроса с Content-Type: application/json.
// Поддерживает метрики типов "gauge" и "counter".
func (h Handler) UpdateJSON(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusResetContent)
		return
	}

	var metric models.Metrics

	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в Metric
	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "counter":
		if metric.Delta == nil {
			http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
			return
		}

		err := h.repo.UpdateCounter(metric.ID, *metric.Delta)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

	case "gauge":
		if metric.Value == nil {
			http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
			return
		}

		err := h.repo.UpdateGauge(metric.ID, *metric.Value)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// UpdatesJSON обрабатывает запрос на пакетное обновление метрик через JSON.
//
// Принимает массив JSON-объектов метрик в теле POST-запроса с Content-Type: application/json.
// Поддерживает метрики типов "gauge" и "counter".
func (h Handler) UpdatesJSON(res http.ResponseWriter, req *http.Request) {

	// Убираем вывод в консоль для корректной работы тестов
	// fmt.Println("===handlers start updates")
	// defer fmt.Println("===handlers finish updates")

	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		// fmt.Println("-=UpdatesJSON:   err http.MethodPost")
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusResetContent)
		// fmt.Println("-=UpdatesJSON:   err req.Header.Get Content-Type...")
		return
	}

	//var metric models.Metrics

	var metrics []models.Metrics

	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		// fmt.Println("-=UpdatesJSON:   err ReadFrom(req.Body)")
		return
	}
	// десериализуем JSON в Metric
	if err = json.Unmarshal(buf.Bytes(), &metrics); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		// fmt.Println("-=UpdatesJSON:   err json.Unmarshal")
		return
	}

	if massUpdater, ok := h.repo.(repository.MassUpdater); ok {
		err := massUpdater.MassUpdate(metrics)
		if err != nil {
			http.Error(res, fmt.Sprintf("mass updater выдал ошибку: %v, err = %s \r\n", metrics, err), http.StatusInternalServerError)
			return
		}
	} else {
		for _, metric := range metrics {
			switch metric.MType {
			case "counter":
				if metric.Delta == nil {
					http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
					return
				}

				err := h.repo.UpdateCounter(metric.ID, *metric.Delta)
				if err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}

			case "gauge":
				if metric.Value == nil {
					http.Error(res, fmt.Sprintf("Нет значения метрики: %v \r\n", metric), http.StatusBadRequest)
					return
				}

				err := h.repo.UpdateGauge(metric.ID, *metric.Value)
				if err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusBadRequest)
				return
			}
		}
	}

	// fmt.Println("   before status ok")
	res.WriteHeader(http.StatusOK)
	// fmt.Println("   after status ok")
}

// GetValueJSON обрабатывает запрос на получение значения метрики через JSON.
//
// Принимает JSON-объект с полями ID и MType в теле POST-запроса с Content-Type: application/json.
// Возвращает JSON-объект метрики с заполненным полем Value или Delta.
func (h Handler) GetValueJSON(res http.ResponseWriter, req *http.Request) {
	//Content-Type application/json
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusResetContent)
		return
	}

	var metric models.Metrics
	var metricOut models.Metrics
	var buf bytes.Buffer
	// читаем тело запроса
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	// десериализуем JSON в Metric
	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	switch metric.MType {
	case "counter":
		v, err := h.repo.GetCounter(metric.ID)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v \r\n", metric.ID), http.StatusNotFound)
			return
		}

		metricOut.Delta = &v
	case "gauge":
		v, err := h.repo.GetGauge(metric.ID)
		if err != nil {
			http.Error(res, fmt.Sprintf("Неизвестное имя метрики: %v \r\n", metric.ID), http.StatusNotFound)
			return
		}

		metricOut.Value = &v
	default:
		http.Error(res, fmt.Sprintf("Неизвестное тип метрики: %v \r\n", metric.MType), http.StatusNotFound)
		return
	}

	metricOut.MType = metric.MType
	metricOut.ID = metric.ID

	resp, err := json.Marshal(metricOut)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write(resp)
}

// PingDB обрабатывает запрос на проверку соединения с базой данных.
//
// Возвращает статус 200, если соединение с базой данных установлено,
// или статус 500 в случае ошибки.
func (h Handler) PingDB(res http.ResponseWriter, req *http.Request) {

	if pinger, ok := h.repo.(repository.Pinger); ok {
		if err := pinger.Ping(); err != nil {
			http.Error(res, fmt.Sprintf("Ошибка: %v \r\n", err), http.StatusInternalServerError)
			return
		}
	}

	res.WriteHeader(http.StatusOK)

}
