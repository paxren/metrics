package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Client представляет gRPC-клиент для отправки метрик
type Client struct {
	conn    *grpc.ClientConn
	client  proto.MetricsClient
	localIP string
}

// NewClient создаёт новый gRPC-клиент
//
// Параметры:
//   - cfg: конфигурация агента с адресом gRPC-сервера
//
// Возвращает:
//   - *Client: указатель на созданный клиент
//   - error: ошибка при подключении к серверу
func NewClient(cfg *config.AgentConfig) (*Client, error) {
	if cfg.GRPCAddress.Host == "" || cfg.GRPCAddress.Port == 0 {
		return nil, fmt.Errorf("gRPC address not configured")
	}

	addr := fmt.Sprintf("%s:%d", cfg.GRPCAddress.Host, cfg.GRPCAddress.Port)

	// Подключаемся к gRPC-серверу
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	// Получаем локальный IP-адрес
	localIP, err := getLocalIP()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	return &Client{
		conn:    conn,
		client:  proto.NewMetricsClient(conn),
		localIP: localIP,
	}, nil
}

// SendMetrics отправляет батч метрик на gRPC-сервер
//
// IP-адрес клиента добавляется в метаданные запроса с ключом "x-real-ip".
//
// Параметры:
//   - ctx: контекст запроса
//   - metrics: список метрик для отправки
//
// Возвращает:
//   - error: ошибка при отправке метрик
func (c *Client) SendMetrics(ctx context.Context, metrics []models.Metrics) error {
	// Создаём запрос с метриками
	req := &proto.UpdateMetricsRequest{
		Metrics: make([]*proto.Metric, 0, len(metrics)),
	}

	for _, m := range metrics {
		req.Metrics = append(req.Metrics, c.metricToProto(m))
	}

	// Добавляем IP-адрес в метаданные
	md := metadata.Pairs("x-real-ip", c.localIP)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Отправляем запрос
	_, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	return nil
}

// Close закрывает соединение с gRPC-сервером
//
// Возвращает:
//   - error: ошибка при закрытии соединения
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// metricToProto конвертирует models.Metrics в proto.Metric
//
// Параметры:
//   - m: метрика из models
//
// Возвращает:
//   - *proto.Metric: сконвертированная метрика
func (c *Client) metricToProto(m models.Metrics) *proto.Metric {
	protoMetric := &proto.Metric{
		Id:   m.ID,
		Type: c.stringToMType(m.MType),
	}

	switch m.MType {
	case models.Counter:
		if m.Delta != nil {
			protoMetric.Delta = *m.Delta
		}
	case models.Gauge:
		if m.Value != nil {
			protoMetric.Value = *m.Value
		}
	}

	return protoMetric
}

// stringToMType конвертирует строку типа метрики в proto.Metric_MType
//
// Параметры:
//   - mt: строковое представление типа метрики
//
// Возвращает:
//   - proto.Metric_MType: тип метрики в формате proto
func (c *Client) stringToMType(mt string) proto.Metric_MType {
	switch mt {
	case models.Counter:
		return proto.Metric_COUNTER
	case models.Gauge:
		return proto.Metric_GAUGE
	default:
		return proto.Metric_GAUGE
	}
}

// getLocalIP возвращает локальный IP-адрес машины
//
// Перебирает сетевые интерфейсы и возвращает первый непустой IP-адрес,
// исключая loopback интерфейсы.
//
// Возвращает:
//   - string: строковое представление IP-адреса
//   - error: ошибка при получении IP
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Пропускаем отключенные интерфейсы
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Пропускаем loopback интерфейсы
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Пропускаем nil IP
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// Возвращаем первый найденный IPv4 адрес
			ip = ip.To4()
			if ip != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid IP address found")
}
