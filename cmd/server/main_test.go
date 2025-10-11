package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paxren/metrics/internal/models"
)

func TestUpdateMetric(t *testing.T) {
	// Create a fresh storage for each test
	originalStorage := memStorage
	defer func() {
		memStorage = originalStorage
	}()

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
		setupFunc      func()
	}{
		{
			name:           "Valid gauge metric",
			method:         "POST",
			url:            "/update/gauge/temperature/36.6",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Valid counter metric",
			method:         "POST",
			url:            "/update/counter/requests/42",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid HTTP method (GET)",
			method:         "GET",
			url:            "/update/gauge/temperature/36.6",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid URL path - too few segments",
			method:         "POST",
			url:            "/update/gauge/temperature",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "неверное количество параметров",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid URL path - too many segments",
			method:         "POST",
			url:            "/update/gauge/temperature/36.6/extra",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "неверное количество параметров",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid metric type",
			method:         "POST",
			url:            "/update/invalid/temperature/36.6",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректный тип метрики",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Empty metric name",
			method:         "POST",
			url:            "/update/gauge//36.6",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Пустое имя метрики",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid gauge value (not a number)",
			method:         "POST",
			url:            "/update/gauge/temperature/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректное значение метрики",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Invalid counter value (not a number)",
			method:         "POST",
			url:            "/update/counter/requests/invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректное значение метрики",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Negative gauge value",
			method:         "POST",
			url:            "/update/gauge/temperature/-10.5",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Negative counter value",
			method:         "POST",
			url:            "/update/counter/requests/-5",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Zero gauge value",
			method:         "POST",
			url:            "/update/gauge/temperature/0",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
		{
			name:           "Zero counter value",
			method:         "POST",
			url:            "/update/counter/requests/0",
			expectedStatus: http.StatusOK,
			expectedBody:   "elems:",
			setupFunc: func() {
				memStorage = models.MakeMemStorage()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Create request
			req := httptest.NewRequest(tt.method, tt.url, nil)
			if req == nil {
				t.Fatalf("Failed to create request")
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			updateMetric(rr, req)

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
						if _, exists := memStorage.GetGauges()[metricName]; !exists {
							t.Errorf("Gauge metric %s was not stored in memory", metricName)
						}
					case "counter":
						if _, exists := memStorage.GetCounters()[metricName]; !exists {
							t.Errorf("Counter metric %s was not stored in memory", metricName)
						}
					}
				}
			}
		})
	}
}

func TestUpdateMetricStorageIntegration(t *testing.T) {
	// Create a fresh storage
	memStorage = models.MakeMemStorage()
	defer func() {
		memStorage = models.MakeMemStorage()
	}()

	// Test gauge metric storage
	req := httptest.NewRequest("POST", "/update/gauge/test_gauge/123.45", nil)
	rr := httptest.NewRecorder()
	updateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if gauge was stored
	gauges := memStorage.GetGauges()
	if value, exists := gauges["test_gauge"]; !exists || value != 123.45 {
		t.Errorf("Gauge metric not stored correctly: got %v, want 123.45", value)
	}

	// Test counter metric storage
	req = httptest.NewRequest("POST", "/update/counter/test_counter/100", nil)
	rr = httptest.NewRecorder()
	updateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if counter was stored
	counters := memStorage.GetCounters()
	if value, exists := counters["test_counter"]; !exists || value != 100 {
		t.Errorf("Counter metric not stored correctly: got %v, want 100", value)
	}

	// Test counter accumulation (should add to existing value)
	req = httptest.NewRequest("POST", "/update/counter/test_counter/50", nil)
	rr = httptest.NewRecorder()
	updateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if counter was accumulated
	counters = memStorage.GetCounters()
	if value, exists := counters["test_counter"]; !exists || value != 150 {
		t.Errorf("Counter metric not accumulated correctly: got %v, want 150", value)
	}

	// Test gauge overwrite (should replace existing value)
	req = httptest.NewRequest("POST", "/update/gauge/test_gauge/999.99", nil)
	rr = httptest.NewRecorder()
	updateMetric(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Check if gauge was overwritten
	gauges = memStorage.GetGauges()
	if value, exists := gauges["test_gauge"]; !exists || value != 999.99 {
		t.Errorf("Gauge metric not overwritten correctly: got %v, want 999.99", value)
	}
}

func BenchmarkUpdateMetric(b *testing.B) {
	memStorage = models.MakeMemStorage()
	defer func() {
		memStorage = models.MakeMemStorage()
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/update/gauge/benchmark_metric/123.45", nil)
		rr := httptest.NewRecorder()
		updateMetric(rr, req)
	}
}
