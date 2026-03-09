package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/paxren/metrics/internal/config"
	"github.com/paxren/metrics/internal/handler"
	"github.com/paxren/metrics/internal/models"
	"github.com/paxren/metrics/internal/proto"
	"github.com/paxren/metrics/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server реализует gRPC-сервер для работы с метриками
type Server struct {
	proto.UnimplementedMetricsServer
	storage  repository.Repository
	logger   *zap.Logger
	subnetMW *handler.TrustedSubnetMiddleware
}

// NewServer создаёт новый gRPC-сервер
//
// Параметры:
//   - storage: хранилище метрик
//   - logger: логгер
//   - subnetMW: middleware для проверки доверенной подсети
//
// Возвращает:
//   - *Server: указатель на созданный сервер
func NewServer(
	storage repository.Repository,
	logger *zap.Logger,
	subnetMW *handler.TrustedSubnetMiddleware,
) *Server {
	return &Server{
		storage:  storage,
		logger:   logger,
		subnetMW: subnetMW,
	}
}

// UpdateMetrics обновляет метрики на сервере
//
// Метод принимает батч метрик и сохраняет их в хранилище.
// Перед сохранением проверяется IP-адрес клиента на принадлежность к доверенной подсети.
//
// Параметры:
//   - ctx: контекст запроса
//   - req: запрос с метриками для обновления
//
// Возвращает:
//   - *proto.UpdateMetricsResponse: пустой ответ при успешном обновлении
//   - error: ошибка при проверке подсети или сохранении метрик
func (s *Server) UpdateMetrics(
	ctx context.Context,
	req *proto.UpdateMetricsRequest,
) (*proto.UpdateMetricsResponse, error) {
	// Проверка доверенной подсети через метаданные
	if err := s.checkTrustedSubnet(ctx); err != nil {
		return nil, err
	}

	// Конвертация и сохранение метрик
	for _, m := range req.Metrics {
		metric := s.protoToMetric(m)

		var err error
		switch metric.MType {
		case models.Counter:
			if metric.Delta != nil {
				err = s.storage.UpdateCounter(metric.ID, *metric.Delta)
			}
		case models.Gauge:
			if metric.Value != nil {
				err = s.storage.UpdateGauge(metric.ID, *metric.Value)
			}
		}

		if err != nil {
			s.logger.Error("failed to update metric",
				zap.String("id", metric.ID),
				zap.String("type", metric.MType),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "failed to update metric")
		}
	}

	return &proto.UpdateMetricsResponse{}, nil
}

// checkTrustedSubnet проверяет IP-адрес клиента на принадлежность к доверенной подсети
//
// IP-адрес извлекается из метаданных запроса с ключом "x-real-ip".
//
// Параметры:
//   - ctx: контекст запроса
//
// Возвращает:
//   - error: ошибка с кодом PermissionDenied, если IP не в доверенной подсети
func (s *Server) checkTrustedSubnet(ctx context.Context) error {
	// Получаем IP из метаданных "x-real-ip"
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.PermissionDenied, "metadata not found")
	}

	realIP := md["x-real-ip"]
	if len(realIP) == 0 {
		return status.Error(codes.PermissionDenied, "x-real-ip metadata is required")
	}

	// Проверяем IP через TrustedSubnetMiddleware
	ip := s.subnetMW.ParseIP(realIP[0])
	if ip == nil {
		return status.Error(codes.PermissionDenied, "invalid IP address")
	}

	if !s.subnetMW.Contains(ip) {
		return status.Error(codes.PermissionDenied, "IP address is not in trusted subnet")
	}

	return nil
}

// protoToMetric конвертирует proto.Metric в models.Metrics
//
// Параметры:
//   - m: метрика из proto
//
// Возвращает:
//   - models.Metrics: сконвертированная метрика
func (s *Server) protoToMetric(m *proto.Metric) models.Metrics {
	metric := models.Metrics{
		ID:    m.Id,
		MType: s.mTypeToString(m.Type),
	}

	switch m.Type {
	case proto.Metric_COUNTER:
		metric.Delta = &m.Delta
	case proto.Metric_GAUGE:
		metric.Value = &m.Value
	}

	return metric
}

// mTypeToString конвертирует proto.Metric_MType в строку
//
// Параметры:
//   - mt: тип метрики из proto
//
// Возвращает:
//   - string: строковое представление типа метрики
func (s *Server) mTypeToString(mt proto.Metric_MType) string {
	switch mt {
	case proto.Metric_COUNTER:
		return models.Counter
	case proto.Metric_GAUGE:
		return models.Gauge
	default:
		return models.Gauge
	}
}

// TrustedSubnetInterceptor создаёт UnaryInterceptor для проверки доверенной подсети
//
// Параметры:
//   - subnetMW: middleware для проверки доверенной подсети
//
// Возвращает:
//   - grpc.UnaryServerInterceptor: interceptor для проверки подсети
func TrustedSubnetInterceptor(subnetMW *handler.TrustedSubnetMiddleware) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Получаем IP из метаданных "x-real-ip"
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "metadata not found")
		}

		realIP := md["x-real-ip"]
		if len(realIP) == 0 {
			return nil, status.Error(codes.PermissionDenied, "x-real-ip metadata is required")
		}

		// Проверяем IP через TrustedSubnetMiddleware
		ip := subnetMW.ParseIP(realIP[0])
		if ip == nil {
			return nil, status.Error(codes.PermissionDenied, "invalid IP address")
		}

		if !subnetMW.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "IP address is not in trusted subnet")
		}

		// IP в доверенной подсети, передаём управление следующему хендлеру
		return handler(ctx, req)
	}
}

// StartServer запускает gRPC-сервер
//
// Параметры:
//   - cfg: конфигурация сервера
//   - storage: хранилище метрик
//   - logger: логгер
//   - subnetMW: middleware для проверки доверенной подсети
//
// Возвращает:
//   - *grpc.Server: запущенный gRPC-сервер
//   - error: ошибка при запуске сервера
func StartServer(
	cfg *config.ServerConfig,
	storage repository.Repository,
	logger *zap.Logger,
	subnetMW *handler.TrustedSubnetMiddleware,
) (*grpc.Server, error) {
	// Создаём listener
	addr := fmt.Sprintf("%s:%d", cfg.GRPCAddress.Host, cfg.GRPCAddress.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// Создаём gRPC-сервер с interceptor
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(TrustedSubnetInterceptor(subnetMW)),
	)

	// Регистрируем сервис Metrics
	proto.RegisterMetricsServer(grpcServer, NewServer(storage, logger, subnetMW))

	// Запускаем сервер в отдельной горутине
	go func() {
		logger.Info("gRPC server started", zap.String("address", addr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server failed", zap.Error(err))
		}
	}()

	return grpcServer, nil
}
