package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"
)

// ExampleHandler_UpdateMetric демонстрирует обновление метрики через URL
func ExampleHandler_UpdateMetric() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём тестовый запрос для обновления метрики gauge
	req := httptest.NewRequest("POST", "/update/gauge/alloc/123.45", nil)
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.UpdateMetric(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Метрика gauge успешно обновлена через URL")
	}

	// Output:
	// Метрика gauge успешно обновлена через URL
}

// ExampleHandler_UpdateMetric_counter демонстрирует обновление метрики типа counter через URL
func ExampleHandler_UpdateMetric_counter() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём тестовый запрос для обновления метрики counter
	req := httptest.NewRequest("POST", "/update/counter/requests/1", nil)
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.UpdateMetric(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Метрика counter успешно обновлена через URL")
	}

	// Output:
	// Метрика counter успешно обновлена через URL
}

// ExampleHandler_UpdateJSON демонстрирует обновление метрики через JSON
func ExampleHandler_UpdateJSON() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём метрику gauge
	value := 123.45
	metric := models.Metrics{
		ID:    "alloc",
		MType: "gauge",
		Value: &value,
	}

	// Сериализуем в JSON
	body, _ := json.Marshal(metric)

	// Создаём тестовый запрос
	req := httptest.NewRequest("POST", "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.UpdateJSON(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Метрика gauge успешно обновлена через JSON")
	}

	// Output:
	// Метрика gauge успешно обновлена через JSON
}

// ExampleHandler_UpdateJSON_counter демонстрирует обновление метрики типа counter через JSON
func ExampleHandler_UpdateJSON_counter() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём метрику counter
	delta := int64(1)
	metric := models.Metrics{
		ID:    "requests",
		MType: "counter",
		Delta: &delta,
	}

	// Сериализуем в JSON
	body, _ := json.Marshal(metric)

	// Создаём тестовый запрос
	req := httptest.NewRequest("POST", "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.UpdateJSON(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Метрика counter успешно обновлена через JSON")
	}

	// Output:
	// Метрика counter успешно обновлена через JSON
}

// ExampleHandler_UpdatesJSON демонстрирует пакетное обновление метрик через JSON
func ExampleHandler_UpdatesJSON() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём несколько метрик
	value1 := 123.45
	value2 := 67.89
	delta1 := int64(1)
	delta2 := int64(2)

	metrics := []models.Metrics{
		{
			ID:    "alloc",
			MType: "gauge",
			Value: &value1,
		},
		{
			ID:    "sys",
			MType: "gauge",
			Value: &value2,
		},
		{
			ID:    "requests",
			MType: "counter",
			Delta: &delta1,
		},
		{
			ID:    "errors",
			MType: "counter",
			Delta: &delta2,
		},
	}

	// Сериализуем в JSON
	body, _ := json.Marshal(metrics)

	// Создаём тестовый запрос
	req := httptest.NewRequest("POST", "/updates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.UpdatesJSON(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Пакетное обновление метрик успешно выполнено")
	}

	// Output:
	// Пакетное обновление метрик успешно выполнено
}

// ExampleHandler_GetValueJSON демонстрирует получение значения метрики через JSON
func ExampleHandler_GetValueJSON() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Сначала добавляем метрику
	value := 123.45
	storage.UpdateGauge("alloc", value)

	// Создаём запрос для получения метрики
	metric := models.Metrics{
		ID:    "alloc",
		MType: "gauge",
	}

	// Сериализуем в JSON
	body, _ := json.Marshal(metric)

	// Создаём тестовый запрос
	req := httptest.NewRequest("POST", "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.GetValueJSON(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Значение метрики успешно получено через JSON")
	}

	// Output:
	// Значение метрики успешно получено через JSON
}

// ExampleHandler_GetMetric демонстрирует получение значения метрики через URL
func ExampleHandler_GetMetric() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Сначала добавляем метрику
	storage.UpdateGauge("alloc", 123.45)

	// Создаём тестовый запрос для получения метрики
	req := httptest.NewRequest("GET", "/value/gauge/alloc", nil)

	// Устанавливаем контекст с параметрами chi, которые обычно устанавливаются роутером
	ctx := req.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"metric_type", "metric_name"},
			Values: []string{"gauge", "alloc"},
		},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.GetMetric(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Значение метрики успешно получено через URL")
	}

	// Output:
	// Значение метрики успешно получено через URL
}

// ExampleHandler_GetMain демонстрирует получение главной страницы со всеми метриками
func ExampleHandler_GetMain() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Добавляем несколько метрик
	storage.UpdateGauge("alloc", 123.45)
	storage.UpdateGauge("sys", 67.89)
	storage.UpdateCounter("requests", 1)
	storage.UpdateCounter("errors", 2)

	// Создаём тестовый запрос для получения главной страницы
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.GetMain(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Главная страница со всеми метриками успешно получена")
	}

	// Output:
	// Главная страница со всеми метриками успешно получена
}

// ExampleHandler_PingDB демонстрирует проверку соединения с базой данных
func ExampleHandler_PingDB() {
	// Создаём хранилище и обработчик
	storage := repository.MakeMemStorage()
	h := handler.NewHandler(storage)

	// Создаём тестовый запрос для проверки соединения
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	// Выполняем обработчик
	h.PingDB(w, req)

	// Проверяем, что запрос обработан корректно
	if w.Code == http.StatusOK {
		fmt.Println("Проверка соединения с базой данных выполнена")
	}

	// Output:
	// Проверка соединения с базой данных выполнена
}
