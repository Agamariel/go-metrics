package agent

import (
	"context"
	"fmt"
	"net"

	pb "github.com/Agamariel/go-metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GRPCSender отправляет метрики на gRPC-сервер.
type GRPCSender struct {
	conn   *grpc.ClientConn
	client pb.MetricsClient
	localIP string
}

// NewGRPCSender создаёт новый gRPC-клиент для отправки метрик.
// address — адрес gRPC-сервера в формате "host:port".
func NewGRPCSender(address string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("подключение к gRPC-серверу %s: %w", address, err)
	}

	localIP := resolveLocalIP()

	return &GRPCSender{
		conn:    conn,
		client:  pb.NewMetricsClient(conn),
		localIP: localIP,
	}, nil
}

// SendAllMetrics конвертирует gauge и counter метрики в proto-запрос и отправляет их батчем.
func (s *GRPCSender) SendAllMetrics(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	total := len(gauges) + len(counters)
	if total == 0 {
		return nil
	}

	protoMetrics := make([]*pb.Metric, 0, total)

	for name, value := range gauges {
		v := value
		protoMetrics = append(protoMetrics, &pb.Metric{
			Id:    name,
			Type:  pb.Metric_GAUGE,
			Value: v,
		})
	}

	for name, delta := range counters {
		d := delta
		protoMetrics = append(protoMetrics, &pb.Metric{
			Id:    name,
			Type:  pb.Metric_COUNTER,
			Delta: d,
		})
	}

	req := &pb.UpdateMetricsRequest{Metrics: protoMetrics}

	// Добавляем IP-адрес агента в метаданные запроса
	outCtx := ctx
	if s.localIP != "" {
		outCtx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", s.localIP)
	}

	if _, err := s.client.UpdateMetrics(outCtx, req); err != nil {
		return fmt.Errorf("gRPC UpdateMetrics: %w", err)
	}

	return nil
}

// Close закрывает gRPC-соединение.
func (s *GRPCSender) Close() error {
	return s.conn.Close()
}

// resolveLocalIP возвращает первый не-loopback IPv4-адрес машины.
func resolveLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil || ip.IsLoopback() {
			continue
		}

		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String()
		}
	}

	return ""
}
