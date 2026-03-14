package agent

import (
	"context"
	"net"
	"testing"

	pb "github.com/Agamariel/go-metrics/internal/proto"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// testMetricsServer — тестовая реализация gRPC-сервера MetricsServer.
type testMetricsServer struct {
	pb.UnimplementedMetricsServer
	receivedMetrics []*pb.Metric
	receivedMD      metadata.MD
}

func (s *testMetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	s.receivedMetrics = req.Metrics
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		s.receivedMD = md
	}
	return &pb.UpdateMetricsResponse{}, nil
}

// startTestGRPCServer запускает тестовый gRPC-сервер и возвращает адрес, сервер и функцию остановки.
func startTestGRPCServer(t *testing.T) (addr string, srv *testMetricsServer, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcSrv := grpc.NewServer()
	testSrv := &testMetricsServer{
		// just to avoid unused import errors
		receivedMetrics: nil,
	}
	_ = service.NewMetricsService(repository.NewMemStorage()) // keep import
	pb.RegisterMetricsServer(grpcSrv, testSrv)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	return lis.Addr().String(), testSrv, func() {
		grpcSrv.GracefulStop()
	}
}

func TestNewGRPCSender_Success(t *testing.T) {
	addr, _, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	require.NotNil(t, sender)
	defer sender.Close()
}

func TestGRPCSender_SendAllMetrics_Empty(t *testing.T) {
	addr, _, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	defer sender.Close()

	err = sender.SendAllMetrics(context.Background(), nil, nil)
	assert.NoError(t, err)
}

func TestGRPCSender_SendAllMetrics_Gauges(t *testing.T) {
	addr, srv, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	defer sender.Close()

	gauges := map[string]float64{
		"cpu":    0.75,
		"memory": 1024.5,
	}
	err = sender.SendAllMetrics(context.Background(), gauges, nil)
	require.NoError(t, err)
	assert.Len(t, srv.receivedMetrics, 2)
	for _, m := range srv.receivedMetrics {
		assert.Equal(t, pb.Metric_GAUGE, m.Type)
	}
}

func TestGRPCSender_SendAllMetrics_Counters(t *testing.T) {
	addr, srv, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	defer sender.Close()

	counters := map[string]int64{
		"requests": 100,
		"errors":   5,
	}
	err = sender.SendAllMetrics(context.Background(), nil, counters)
	require.NoError(t, err)
	assert.Len(t, srv.receivedMetrics, 2)
	for _, m := range srv.receivedMetrics {
		assert.Equal(t, pb.Metric_COUNTER, m.Type)
	}
}

func TestGRPCSender_SendAllMetrics_Mixed(t *testing.T) {
	addr, srv, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	defer sender.Close()

	gauges := map[string]float64{"temp": 36.6}
	counters := map[string]int64{"hits": 42}
	err = sender.SendAllMetrics(context.Background(), gauges, counters)
	require.NoError(t, err)
	assert.Len(t, srv.receivedMetrics, 2)
}

func TestGRPCSender_SendAllMetrics_SetsXRealIP(t *testing.T) {
	addr, srv, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)
	defer sender.Close()

	sender.localIP = "192.168.1.100"

	gauges := map[string]float64{"cpu": 0.5}
	err = sender.SendAllMetrics(context.Background(), gauges, nil)
	require.NoError(t, err)

	ips := srv.receivedMD.Get("x-real-ip")
	require.Len(t, ips, 1)
	assert.Equal(t, "192.168.1.100", ips[0])
}

func TestGRPCSender_Close(t *testing.T) {
	addr, _, stop := startTestGRPCServer(t)
	defer stop()

	sender, err := NewGRPCSender(addr)
	require.NoError(t, err)

	err = sender.Close()
	assert.NoError(t, err)
}

func TestGRPCSender_SendError_Timeout(t *testing.T) {
	sender, err := NewGRPCSender("127.0.0.1:1") // порт 1 — недоступен
	require.NoError(t, err)
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50)
	defer cancel()

	err = sender.SendAllMetrics(ctx, map[string]float64{"m": 1.0}, nil)
	assert.Error(t, err)
}

func TestResolveLocalIP(t *testing.T) {
	ip := resolveLocalIP()
	if ip != "" {
		parsed := net.ParseIP(ip)
		assert.NotNil(t, parsed, "resolveLocalIP вернул невалидный IP: %s", ip)
		assert.NotNil(t, parsed.To4(), "ожидался IPv4-адрес, получен: %s", ip)
	}
}
