package agent

//создано roo code + glm 4.6

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/repository"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	gauges   map[string]float64
	counters map[string]int64
	errors   map[string]bool // Map to simulate errors for specific keys
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		errors:   make(map[string]bool),
	}
}

func (m *MockRepository) UpdateGauge(key string, value float64) error {
	if m.errors[key] {
		return fmt.Errorf("mock error for gauge %s", key)
	}
	m.gauges[key] = value
	return nil
}

func (m *MockRepository) UpdateCounter(key string, value int64) error {
	if m.errors[key] {
		return fmt.Errorf("mock error for counter %s", key)
	}
	m.counters[key] += value
	return nil
}

func (m *MockRepository) GetGauge(key string) (float64, error) {
	if m.errors[key] {
		return 0, fmt.Errorf("mock error for gauge %s", key)
	}
	val, ok := m.gauges[key]
	if !ok {
		return 0, repository.ErrGaugeNotFound
	}
	return val, nil
}

func (m *MockRepository) GetCounter(key string) (int64, error) {
	if m.errors[key] {
		return 0, fmt.Errorf("mock error for counter %s", key)
	}
	val, ok := m.counters[key]
	if !ok {
		return 0, repository.ErrCounterNotFound
	}
	return val, nil
}

func (m *MockRepository) GetGaugesKeys() []string {
	keys := make([]string, 0, len(m.gauges))
	for k := range m.gauges {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockRepository) GetCountersKeys() []string {
	keys := make([]string, 0, len(m.counters))
	for k := range m.counters {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockRepository) SetError(key string, shouldError bool) {
	m.errors[key] = shouldError
}

func TestAgent_Send(t *testing.T) {
	// Create a test server to handle the HTTP requests with JSON and gzip support
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request path matches the expected pattern
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check Content-Encoding header
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding gzip, got %s", r.Header.Get("Content-Encoding"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Read and decompress the request body
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("Failed to create gzip reader: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer reader.Close()

		var buf bytes.Buffer
		_, err = buf.ReadFrom(reader)
		if err != nil {
			t.Errorf("Failed to read decompressed data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse the JSON to verify it's a valid metric
		var metric models.Metrics
		if err := json.Unmarshal(buf.Bytes(), &metric); err != nil {
			t.Errorf("Failed to unmarshal JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate metric structure
		if metric.ID == "" {
			t.Error("Metric ID is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if metric.MType != models.Gauge && metric.MType != models.Counter {
			t.Errorf("Invalid metric type: %s", metric.MType)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return success response with gzip compression
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()
		response := `{"status":"success"}`
		gzWriter.Write([]byte(response))
	}))
	defer server.Close()

	// Extract host and port from the test server URL
	defaultHost := config.NewHostAddress()

	tests := []struct {
		name          string
		repo          repository.Repository
		host          config.HostAddress
		wantErrCount  int
		setupRepo     func(repository.Repository)
		useTestServer bool
	}{
		{
			name:          "Empty repository",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  0,
			useTestServer: true,
		},
		{
			name:          "Repository with gauges only",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  0,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateGauge("Alloc", 12345.67)
				mockRepo.UpdateGauge("HeapAlloc", 98765.43)
			},
		},
		{
			name:          "Repository with counters only",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  0,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateCounter("PollCount", 42)
				mockRepo.UpdateCounter("RandomCounter", 100)
			},
		},
		{
			name:          "Repository with both gauges and counters",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  0,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateGauge("Alloc", 12345.67)
				mockRepo.UpdateGauge("HeapAlloc", 98765.43)
				mockRepo.UpdateCounter("PollCount", 42)
				mockRepo.UpdateCounter("RandomCounter", 100)
			},
		},
		{
			name:          "Repository with gauge retrieval errors",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  1,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateGauge("Alloc", 12345.67)
				mockRepo.SetError("Alloc", true) // This will cause an error when retrieving
			},
		},
		{
			name:          "Repository with counter retrieval errors",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  1,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateCounter("PollCount", 42)
				mockRepo.SetError("PollCount", true) // This will cause an error when retrieving
			},
		},
		{
			name:          "Repository with mixed errors",
			repo:          NewMockRepository(),
			host:          *defaultHost,
			wantErrCount:  2,
			useTestServer: true,
			setupRepo: func(r repository.Repository) {
				mockRepo := r.(*MockRepository)
				mockRepo.UpdateGauge("Alloc", 12345.67)
				mockRepo.UpdateCounter("PollCount", 42)
				mockRepo.SetError("Alloc", true)
				mockRepo.SetError("PollCount", true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup repository if needed
			if tt.setupRepo != nil {
				tt.setupRepo(tt.repo)
			}

			// Use test server if specified
			if tt.useTestServer {
				// Parse the test server URL to get host and port
				parts := server.URL[7:] // Remove "http://" prefix
				hostAddr := config.NewHostAddress()
				hostAddr.Set(parts)
				tt.host = *hostAddr
			}

			// Create agent with the test repository
			a := NewAgent(tt.host)
			// Set the repository manually since NewAgent doesn't accept it as parameter
			a.Repo = tt.repo

			// Call Send method
			got := a.Send()

			// Check error count
			if len(got) != tt.wantErrCount {
				t.Errorf("Send() error count = %d, want %d", len(got), tt.wantErrCount)
				if len(got) > 0 {
					for i, err := range got {
						t.Errorf("Error %d: %v", i, err)
					}
				}
			}
		})
	}
}

func TestAgent_Add(t *testing.T) {
	tests := []struct {
		name     string
		repo     repository.Repository
		host     config.HostAddress
		memStats *runtime.MemStats
	}{
		{
			name:     "Add with valid MemStats",
			repo:     repository.MakeMemStorage(),
			host:     *config.NewHostAddress(),
			memStats: &runtime.MemStats{Alloc: 1024, HeapAlloc: 2048, NumGC: 5},
		},
		{
			name:     "Add with zero MemStats",
			repo:     repository.MakeMemStorage(),
			host:     *config.NewHostAddress(),
			memStats: &runtime.MemStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAgent(tt.host)
			// Set the repository manually since NewAgent doesn't accept it as parameter
			a.Repo = tt.repo

			// Call Add method
			a.Add(tt.memStats)

			// Verify that gauges were added to the repository
			gaugesKeys := tt.repo.GetGaugesKeys()
			if len(gaugesKeys) == 0 {
				t.Error("Expected gauges to be added to repository, but got none")
			}

			// Check specific metrics that should be added
			expectedMetrics := []string{
				"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
				"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
				"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
				"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
				"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
				"Sys", "TotalAlloc",
			}

			for _, metric := range expectedMetrics {
				_, err := tt.repo.GetGauge(metric)
				if err != nil {
					t.Errorf("Expected gauge %s to be added, but got error: %v", metric, err)
				}
			}
		})
	}
}
