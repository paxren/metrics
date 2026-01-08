package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/paxren/metrics/internal/audit"
	"github.com/paxren/metrics/internal/models"
)

// responseWriter - обёртка для отслеживания статуса ответа
type responseWriter struct {
	http.ResponseWriter
	status int
}

type Auditor struct {
	observers []audit.Observer
}

func NewAuditor(observers []audit.Observer) *Auditor {
	return &Auditor{
		observers: observers,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// WithAudit создаёт middleware для аудита запросов
func (a *Auditor) WithAudit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если нет наблюдателей, просто передаем управление дальше
		if len(a.observers) == 0 {
			h(w, r)
			return
		}

		// Извлекаем метрики из запроса
		var metrics []string
		var bodyBytes []byte

		// Сохраняем тело запроса для последующего восстановления
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "application/json" {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			if err == nil {
				// Восстанавливаем тело запроса для следующих обработчиков
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Извлекаем метрики из тела запроса
				metrics = extractMetricsFromJSON(bodyBytes, r.URL.Path)
			}
		} else {
			// Извлекаем метрики из URL для не-JSON запросов
			metrics = extractMetricsFromURL(r.URL.Path)
		}

		// Создаём обёртку для ResponseWriter
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// Выполняем основной обработчик
		h(wrapped, r)

		// Если запрос успешный (статус 2xx) и есть метрики для аудита
		if wrapped.status >= 200 && wrapped.status < 300 && len(metrics) > 0 {
			// Создаём событие аудита
			event := models.NewAuditEvent(metrics, getIPFromRequest(r))

			// Уведомляем наблюдателей
			for _, observer := range a.observers {
				observer.Notify(event) // Игнорируем ошибки, чтобы не прерывать обработку
			}
		}
	}
}

// extractMetricsFromJSON извлекает названия метрик из JSON-тела запроса
func extractMetricsFromJSON(bodyBytes []byte, path string) []string {
	var metrics []string

	// Для одиночной метрики (/update, /update/)
	if strings.HasSuffix(path, "/update") || strings.HasSuffix(path, "/update/") {
		var metric models.Metrics
		if err := json.Unmarshal(bodyBytes, &metric); err == nil && metric.ID != "" {
			metrics = append(metrics, metric.ID)
		}
	}

	// Для пакета метрик (/updates, /updates/)
	if strings.HasSuffix(path, "/updates") || strings.HasSuffix(path, "/updates/") {
		var metricModels []models.Metrics
		if err := json.Unmarshal(bodyBytes, &metricModels); err == nil {
			for _, m := range metricModels {
				if m.ID != "" {
					metrics = append(metrics, m.ID)
				}
			}
		}
	}

	return metrics
}

// extractMetricsFromURL извлекает названия метрик из URL пути
func extractMetricsFromURL(path string) []string {
	var metrics []string

	// Для эндпоинта /update/{metric_type}/{metric_name}/{metric_value}
	if strings.Contains(path, "/update/") && !strings.HasSuffix(path, "/update/") {
		elems := strings.Split(path, "/")
		if len(elems) >= 4 {
			metrics = append(metrics, elems[3]) // имя метрики
		}
	}

	return metrics
}

// getIPFromRequest извлекает IP-адрес из запроса
func getIPFromRequest(r *http.Request) string {
	// Проверяем заголовки для проксированных запросов
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Извлекаем из RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
