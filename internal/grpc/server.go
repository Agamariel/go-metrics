// Package grpc содержит реализацию gRPC-сервера для приёма метрик.
package grpc

import (
	"context"

	"github.com/Agamariel/go-metrics/internal/models"
	pb "github.com/Agamariel/go-metrics/internal/proto"
	"github.com/Agamariel/go-metrics/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsServer реализует gRPC-сервис Metrics.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	service *service.MetricsService
}

// NewMetricsServer создаёт новый gRPC-сервер метрик.
func NewMetricsServer(svc *service.MetricsService) *MetricsServer {
	return &MetricsServer{service: svc}
}

// RegisterMetricsServer регистрирует MetricsServer на gRPC-сервере.
func RegisterMetricsServer(s *grpc.Server, svc *service.MetricsService) {
	pb.RegisterMetricsServer(s, NewMetricsServer(svc))
}

// UpdateMetrics принимает батч метрик и сохраняет их через сервис.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if len(req.Metrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	metrics := make([]models.Metrics, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		metric, err := protoToModel(m)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "некорректная метрика %q: %v", m.Id, err)
		}
		metrics = append(metrics, metric)
	}

	if err := s.service.UpdateMetrics(ctx, metrics); err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка сохранения метрик: %v", err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// protoToModel конвертирует proto-метрику в модель приложения.
func protoToModel(m *pb.Metric) (models.Metrics, error) {
	switch m.Type {
	case pb.Metric_GAUGE:
		v := m.Value
		return models.Metrics{
			ID:    m.Id,
			MType: models.Gauge,
			Value: &v,
		}, nil
	case pb.Metric_COUNTER:
		d := m.Delta
		return models.Metrics{
			ID:    m.Id,
			MType: models.Counter,
			Delta: &d,
		}, nil
	default:
		return models.Metrics{}, status.Errorf(codes.InvalidArgument, "неизвестный тип метрики: %v", m.Type)
	}
}
