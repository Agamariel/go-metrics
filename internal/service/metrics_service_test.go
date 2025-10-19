package service

import (
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

// MockStorage - мок для тестирования
type MockStorage struct {
	metrics map[string]models.Metrics
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		metrics: make(map[string]models.Metrics),
	}
}

func (m *MockStorage) UpdateMetric(metric models.Metrics) error {
	key := metric.ID + ":" + metric.MType
	m.metrics[key] = metric
	return nil
}

func TestNewMetricsService(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	if service == nil {
		t.Fatal("NewMetricsService returned nil")
	}

	if service.storage == nil {
		t.Error("Service storage is nil")
	}
}

func TestUpdateMetricByPathGauge(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	err := service.UpdateMetricByPath("gauge/TestGauge/123.456")
	if err != nil {
		t.Errorf("UpdateMetricByPath failed: %v", err)
	}

}

func TestUpdateMetricByPathCounter(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	err := service.UpdateMetricByPath("counter/TestCounter/42")
	if err != nil {
		t.Errorf("UpdateMetricByPath failed: %v", err)
	}

}

func TestUpdateMetricByPathInvalidType(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	err := service.UpdateMetricByPath("invalid/Test/123")
	if err != ErrInvalidType {
		t.Errorf("Expected ErrInvalidType, got %v", err)
	}
}

func TestUpdateMetricByPathInvalidGaugeValue(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	err := service.UpdateMetricByPath("gauge/Test/not-a-number")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

func TestUpdateMetricByPathInvalidCounterValue(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	err := service.UpdateMetricByPath("counter/Test/not-a-number")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

func TestUpdateMetricByPathCounterFloat(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	// Counter не должен принимать float
	err := service.UpdateMetricByPath("counter/Test/123.456")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue for float counter, got %v", err)
	}
}
