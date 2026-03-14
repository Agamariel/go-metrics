package grpc

import (
	"context"
	"testing"

	pb "github.com/Agamariel/go-metrics/internal/proto"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestService() *service.MetricsService {
	return service.NewMetricsService(repository.NewMemStorage())
}

func TestMetricsServer_UpdateMetrics_EmptyRequest(t *testing.T) {
	srv := NewMetricsServer(newTestService())
	resp, err := srv.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMetricsServer_UpdateMetrics_Gauge(t *testing.T) {
	srv := NewMetricsServer(newTestService())
	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "temperature", Type: pb.Metric_GAUGE, Value: 36.6},
		},
	}
	resp, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMetricsServer_UpdateMetrics_Counter(t *testing.T) {
	srv := NewMetricsServer(newTestService())
	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "requests", Type: pb.Metric_COUNTER, Delta: 42},
		},
	}
	resp, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestMetricsServer_UpdateMetrics_Batch(t *testing.T) {
	srv := NewMetricsServer(newTestService())
	req := &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "cpu", Type: pb.Metric_GAUGE, Value: 0.75},
			{Id: "mem", Type: pb.Metric_GAUGE, Value: 1024.0},
			{Id: "hits", Type: pb.Metric_COUNTER, Delta: 100},
			{Id: "errors", Type: pb.Metric_COUNTER, Delta: 5},
		},
	}
	resp, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestProtoToModel_Gauge(t *testing.T) {
	m := &pb.Metric{Id: "temp", Type: pb.Metric_GAUGE, Value: 22.5}
	result, err := protoToModel(m)
	require.NoError(t, err)
	assert.Equal(t, "temp", result.ID)
	assert.Equal(t, "gauge", result.MType)
	require.NotNil(t, result.Value)
	assert.Equal(t, 22.5, *result.Value)
	assert.Nil(t, result.Delta)
}

func TestProtoToModel_Counter(t *testing.T) {
	m := &pb.Metric{Id: "reqs", Type: pb.Metric_COUNTER, Delta: 7}
	result, err := protoToModel(m)
	require.NoError(t, err)
	assert.Equal(t, "reqs", result.ID)
	assert.Equal(t, "counter", result.MType)
	require.NotNil(t, result.Delta)
	assert.Equal(t, int64(7), *result.Delta)
	assert.Nil(t, result.Value)
}

func TestProtoToModel_UnknownType(t *testing.T) {
	m := &pb.Metric{Id: "unknown", Type: pb.Metric_MType(999)}
	_, err := protoToModel(m)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestNewMetricsServer(t *testing.T) {
	svc := newTestService()
	srv := NewMetricsServer(svc)
	assert.NotNil(t, srv)
}
