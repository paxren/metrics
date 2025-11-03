package handler

//сгенерировано roo code + glm 4.6

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"
)

func TestUpdateMetric(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid gauge metric",
			method:         "POST",
			url:            "/update/gauge/temperature/36.6",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
		{
			name:           "Valid counter metric",
			method:         "POST",
			url:            "/update/counter/requests/42",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
		{
			name:           "Invalid HTTP method (GET)",
			method:         "GET",
			url:            "/update/gauge/temperature/36.6",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
		{
			name:           "Invalid URL path - too few segments",
			method:         "POST",
			url:            "/update/gauge/temperature",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "неверное количество параметров",
		},
		{
			name:           "Invalid URL path - too many segments",
			method:         "POST",
			url:            "/update/gauge/temperature/36.6/extra",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "неверное количество параметров",
		},
		{
			name:           "Invalid metric type",
			method:         "POST",
			url:            "/update/invalid/temperature/36.6",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректный тип метрики",
		},
		{
			name:           "Empty metric name",
			method:         "POST",
			url:            "/update/gauge//36.6",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Пустое имя метрики",
		},
		{
			name:           "Invalid gauge value (not a number)",
			method:         "POST",
			url:            "/update/gauge/temperature/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректное значение метрики",
		},
		{
			name:           "Invalid counter value (not a number)",
			method:         "POST",
			url:            "/update/counter/requests/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректное значение метрики",
		},
		{
			name:           "Negative gauge value",
			method:         "POST",
			url:            "/update/gauge/temperature/-10.5",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
		{
			name:           "Negative counter value",
			method:         "POST",
			url:            "/update/counter/requests/-5",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
		{
			name:           "Zero gauge value",
			method:         "POST",
			url:            "/update/gauge/temperature/0",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
		{
			name:           "Zero counter value",
			method:         "POST",
			url:            "/update/counter/requests/0",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh storage and handler for each test
			memStorage := repository.MakeMemStorage()
			handler := NewHandler(memStorage)

			// Create request
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if req == nil {
				t.Fatalf("Failed to create request")
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.UpdateMetric(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check response body
			body := rr.Body.String()
			if tt.expectedBody != "" && !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Handler returned unexpected body: got %v want to contain %v",
					body, tt.expectedBody)
			}

			// For successful requests, verify the metric was actually stored
			if tt.expectedStatus == http.StatusOK {
				elems := strings.Split(tt.url, "/")
				if len(elems) == 5 {
					metricType := elems[2]
					metricName := elems[3]

					switch metricType {
					case "gauge":
						if _, err := memStorage.GetGauge(metricName); err != nil {
							t.Errorf("Gauge metric %s was not stored in memory: %v", metricName, err)
						}
					case "counter":
						if _, err := memStorage.GetCounter(metricName); err != nil {
							t.Errorf("Counter metric %s was not stored in memory: %v", metricName, err)
						}
					}
				}
			}
		})
	}
}

func TestUpdateMetricStorageIntegration(t *testing.T) {
	// Create a fresh storage and handler
	memStorage := repository.MakeMemStorage()
	handler := NewHandler(memStorage)

	// Test gauge metric storage
	req := httptest.NewRequest("POST", "/update/gauge/test_gauge/123.45", nil)
	rr := httptest.NewRecorder()
	handler.UpdateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if gauge was stored
	if value, err := memStorage.GetGauge("test_gauge"); err != nil || value != 123.45 {
		t.Errorf("Gauge metric not stored correctly: got %v, want 123.45, error: %v", value, err)
	}

	// Test counter metric storage
	req = httptest.NewRequest("POST", "/update/counter/test_counter/100", nil)
	rr = httptest.NewRecorder()
	handler.UpdateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if counter was stored
	if value, err := memStorage.GetCounter("test_counter"); err != nil || value != 100 {
		t.Errorf("Counter metric not stored correctly: got %v, want 100, error: %v", value, err)
	}

	// Test counter accumulation (should add to existing value)
	req = httptest.NewRequest("POST", "/update/counter/test_counter/50", nil)
	rr = httptest.NewRecorder()
	handler.UpdateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if counter was accumulated
	if value, err := memStorage.GetCounter("test_counter"); err != nil || value != 150 {
		t.Errorf("Counter metric not accumulated correctly: got %v, want 150, error: %v", value, err)
	}

	// Test gauge overwrite (should replace existing value)
	req = httptest.NewRequest("POST", "/update/gauge/test_gauge/999.99", nil)
	rr = httptest.NewRecorder()
	handler.UpdateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if gauge was overwritten
	if value, err := memStorage.GetGauge("test_gauge"); err != nil || value != 999.99 {
		t.Errorf("Gauge metric not overwritten correctly: got %v, want 999.99, error: %v", value, err)
	}
}

func BenchmarkUpdateMetric(b *testing.B) {
	memStorage := repository.MakeMemStorage()
	handler := NewHandler(memStorage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/gauge/benchmark_metric/123.45", nil)
		rr := httptest.NewRecorder()
		handler.UpdateMetric(rr, req)
	}
}

func TestUpdateJSON(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid gauge metric JSON",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge","value":36.6}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid counter metric JSON",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"requests","type":"counter","delta":42}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid HTTP method (GET)",
			method:         "GET",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge","value":36.6}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid Content-Type",
			method:         "POST",
			contentType:    "text/plain",
			body:           `{"id":"temperature","type":"gauge","value":36.6}`,
			expectedStatus: http.StatusResetContent,
		},
		{
			name:           "Invalid JSON",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge","value":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing value for gauge",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing delta for counter",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"requests","type":"counter"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid metric type",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"invalid","value":36.6}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh storage and handler for each test
			memStorage := repository.MakeMemStorage()
			handler := NewHandler(memStorage)

			// Create request with body
			req := httptest.NewRequest(tt.method, "/update/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.UpdateJSON(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check response body if expected
			if tt.expectedBody != "" && !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("Handler returned unexpected body: got %v want to contain %v",
					rr.Body.String(), tt.expectedBody)
			}

			// For successful requests, verify the metric was actually stored
			if tt.expectedStatus == http.StatusOK {
				// Parse the JSON body to extract metric info
				var metric models.Metrics
				if err := json.Unmarshal([]byte(tt.body), &metric); err == nil {
					switch metric.MType {
					case "gauge":
						if _, err := memStorage.GetGauge(metric.ID); err != nil {
							t.Errorf("Gauge metric %s was not stored in memory: %v", metric.ID, err)
						}
					case "counter":
						if _, err := memStorage.GetCounter(metric.ID); err != nil {
							t.Errorf("Counter metric %s was not stored in memory: %v", metric.ID, err)
						}
					}
				}
			}
		})
	}
}

func TestGetValueJSON(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		setupStorage   func(repository.Repository)
	}{
		{
			name:           "Get existing gauge metric",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge"}`,
			expectedStatus: http.StatusOK,
			setupStorage: func(r repository.Repository) {
				r.UpdateGauge("temperature", 36.6)
			},
		},
		{
			name:           "Get existing counter metric",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"requests","type":"counter"}`,
			expectedStatus: http.StatusOK,
			setupStorage: func(r repository.Repository) {
				r.UpdateCounter("requests", 42)
			},
		},
		{
			name:           "Get non-existing gauge metric",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"nonexistent","type":"gauge"}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Get non-existing counter metric",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"nonexistent","type":"counter"}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid HTTP method (GET)",
			method:         "GET",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge"}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid Content-Type",
			method:         "POST",
			contentType:    "text/plain",
			body:           `{"id":"temperature","type":"gauge"}`,
			expectedStatus: http.StatusResetContent,
		},
		{
			name:           "Invalid JSON",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"gauge"`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid metric type",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"id":"temperature","type":"invalid"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh storage and handler for each test
			memStorage := repository.MakeMemStorage()
			handler := NewHandler(memStorage)

			// Setup storage if needed
			if tt.setupStorage != nil {
				tt.setupStorage(memStorage)
			}

			// Create request with body
			req := httptest.NewRequest(tt.method, "/value/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.GetValueJSON(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// For successful requests, verify the response contains the expected metric
			if tt.expectedStatus == http.StatusOK {
				var responseMetric models.Metrics
				if err := json.Unmarshal(rr.Body.Bytes(), &responseMetric); err != nil {
					t.Errorf("Failed to unmarshal response JSON: %v", err)
				} else {
					// Parse the request body to get the metric info
					var requestMetric models.Metrics
					if err := json.Unmarshal([]byte(tt.body), &requestMetric); err == nil {
						if responseMetric.ID != requestMetric.ID {
							t.Errorf("Response metric ID %s doesn't match request %s", responseMetric.ID, requestMetric.ID)
						}
						if responseMetric.MType != requestMetric.MType {
							t.Errorf("Response metric type %s doesn't match request %s", responseMetric.MType, requestMetric.MType)
						}
					}
				}
			}
		})
	}
}
