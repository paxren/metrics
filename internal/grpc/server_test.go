package grpc

import (
	"context"
	"testing"

	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/proto"
	"github.com/paxren/metrics/internal/repository"
	"go.uber.org/zap"
)

// mockMetricsServer реализует интерфейс proto.MetricsServer для тестирования
type mockMetricsServer struct {
	proto.UnimplementedMetricsServer
	storage repository.Repository
}

func (m *mockMetricsServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	for _, metric := range req.Metrics {
		metricModel := models.Metrics{
			ID: metric.Id,
		}
		if metric.Type == proto.Metric_GAUGE {
			metricModel.Value = &metric.Value
		} else {
			metricModel.Delta = &metric.Delta
		}

		if metricModel.Value != nil {
			m.storage.UpdateGauge(metricModel.ID, *metricModel.Value)
		} else if metricModel.Delta != nil {
			m.storage.UpdateCounter(metricModel.ID, *metricModel.Delta)
		}
	}
	return &proto.UpdateMetricsResponse{}, nil
}

// TestTrustedSubnetMiddleware_ParseIP тестирует метод ParseIP
func TestTrustedSubnetMiddleware_ParseIP(t *testing.T) {
	subnetMW, err := handler.NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create subnet middleware: %v", err)
	}

	tests := []struct {
		name     string
		ipStr    string
		expected bool
	}{
		{
			name:     "Валидный IPv4",
			ipStr:    "192.168.1.100",
			expected: true,
		},
		{
			name:     "Валидный IPv4 другой подсети",
			ipStr:    "10.0.0.1",
			expected: true,
		},
		{
			name:     "Невалидный IP",
			ipStr:    "invalid-ip",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := subnetMW.ParseIP(tt.ipStr)
			if tt.expected {
				if ip == nil {
					t.Errorf("Expected valid IP, got nil")
				}
			} else {
				if ip != nil {
					t.Errorf("Expected nil IP, got %v", ip)
				}
			}
		})
	}
}

// TestTrustedSubnetMiddleware_Contains тестирует метод Contains
func TestTrustedSubnetMiddleware_Contains(t *testing.T) {
	subnetMW, err := handler.NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create subnet middleware: %v", err)
	}

	tests := []struct {
		name     string
		ipStr    string
		expected bool
	}{
		{
			name:     "IP в доверенной подсети",
			ipStr:    "192.168.1.100",
			expected: true,
		},
		{
			name:     "IP не в доверенной подсети",
			ipStr:    "10.0.0.1",
			expected: false,
		},
		{
			name:     "Доверенная подсеть не задана",
			ipStr:    "192.168.1.100",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := subnetMW.ParseIP(tt.ipStr)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ipStr)
			}

			result := subnetMW.Contains(ip)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestServer_UpdateMetrics тестирует метод UpdateMetrics сервера
func TestServer_UpdateMetrics(t *testing.T) {
	// Создаём тестовое хранилище
	storage := repository.MakeMemStorage()

	// Создаём middleware для доверенной подсети
	subnetMW, err := handler.NewTrustedSubnetMiddleware("192.168.1.0/24")
	if err != nil {
		t.Fatalf("Failed to create subnet middleware: %v", err)
	}

	// Создаём logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Создаём сервер
	server := NewServer(storage, logger, subnetMW)

	tests := []struct {
		name        string
		clientIP    string
		metrics     []*proto.Metric
		expectError bool
	}{
		{
			name:     "Успешное обновление метрик",
			clientIP: "192.168.1.100",
			metrics: []*proto.Metric{
				{
					Id:    "test_gauge",
					Type:  proto.Metric_GAUGE,
					Value: 42.5,
				},
				{
					Id:    "test_counter",
					Type:  proto.Metric_COUNTER,
					Delta: 10,
				},
			},
			expectError: false,
		},
		{
			name:     "IP не в доверенной подсети",
			clientIP: "10.0.0.1",
			metrics: []*proto.Metric{
				{
					Id:    "test_gauge",
					Type:  proto.Metric_GAUGE,
					Value: 42.5,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём контекст с метаданными
			ctx := context.Background()

			// Вызываем UpdateMetrics
			resp, err := server.UpdateMetrics(ctx, &proto.UpdateMetricsRequest{
				Metrics: tt.metrics,
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if resp != nil {
					t.Errorf("Expected nil response, got %v", resp)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if resp == nil {
					t.Errorf("Expected response, got nil")
				}

				// Проверяем, что метрики сохранены
				for _, m := range tt.metrics {
					if m.Type == proto.Metric_GAUGE {
						val, err := storage.GetGauge(m.Id)
						if err != nil {
							t.Errorf("Failed to get gauge metric: %v", err)
						}
						if val != m.Value {
							t.Errorf("Expected gauge value %v, got %v", m.Value, val)
						}
					} else if m.Type == proto.Metric_COUNTER {
						val, err := storage.GetCounter(m.Id)
						if err != nil {
							t.Errorf("Failed to get counter metric: %v", err)
						}
						if val != m.Delta {
							t.Errorf("Expected counter value %v, got %v", m.Delta, val)
						}
					}
				}
			}
		})
	}
}

// TestServer_protoToMetric тестирует конвертацию proto.Metric в models.Metrics
func TestServer_protoToMetric(t *testing.T) {
	storage := repository.MakeMemStorage()
	subnetMW, _ := handler.NewTrustedSubnetMiddleware("")
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	server := NewServer(storage, logger, subnetMW)

	tests := []struct {
		name     string
		proto    *proto.Metric
		expected models.Metrics
	}{
		{
			name: "Gauge метрика",
			proto: &proto.Metric{
				Id:    "test_gauge",
				Type:  proto.Metric_GAUGE,
				Value: 42.5,
			},
			expected: models.Metrics{
				ID:    "test_gauge",
				MType: "gauge",
				Value: float64Ptr(42.5),
			},
		},
		{
			name: "Counter метрика",
			proto: &proto.Metric{
				Id:    "test_counter",
				Type:  proto.Metric_COUNTER,
				Delta: 10,
			},
			expected: models.Metrics{
				ID:    "test_counter",
				MType: "counter",
				Delta: int64Ptr(10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.protoToMetric(tt.proto)

			if result.ID != tt.expected.ID {
				t.Errorf("Expected ID %s, got %s", tt.expected.ID, result.ID)
			}
			if result.MType != tt.expected.MType {
				t.Errorf("Expected MType %s, got %s", tt.expected.MType, result.MType)
			}
			if tt.expected.Value != nil {
				if result.Value == nil {
					t.Errorf("Expected Value %v, got nil", *tt.expected.Value)
				} else if *result.Value != *tt.expected.Value {
					t.Errorf("Expected Value %v, got %v", *tt.expected.Value, *result.Value)
				}
			}
			if tt.expected.Delta != nil {
				if result.Delta == nil {
					t.Errorf("Expected Delta %v, got nil", *tt.expected.Delta)
				} else if *result.Delta != *tt.expected.Delta {
					t.Errorf("Expected Delta %v, got %v", *tt.expected.Delta, *result.Delta)
				}
			}
		})
	}
}

// Вспомогательные функции для тестов
func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
