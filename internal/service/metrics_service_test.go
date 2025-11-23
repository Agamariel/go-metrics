package service

import (
	"context"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
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

func (m *MockStorage) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	key := metric.ID + ":" + metric.MType
	m.metrics[key] = metric
	return nil
}

func (m *MockStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		key := metric.ID + ":" + metric.MType
		m.metrics[key] = metric
	}
	return nil
}

func (m *MockStorage) GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error) {
	key := id + ":" + mType
	metric, found := m.metrics[key]
	if !found {
		return models.Metrics{}, repository.ErrNotFound
	}
	return metric, nil
}

func (m *MockStorage) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	var all []models.Metrics
	for _, metric := range m.metrics {
		all = append(all, metric)
	}
	return all, nil
}

func (m *MockStorage) Close() error {
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
	ctx := context.Background()

	err := service.UpdateMetricByPath(ctx, "gauge/TestGauge/123.456")
	if err != nil {
		t.Errorf("UpdateMetricByPath failed: %v", err)
	}

}

func TestUpdateMetricByPathCounter(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	err := service.UpdateMetricByPath(ctx, "counter/TestCounter/42")
	if err != nil {
		t.Errorf("UpdateMetricByPath failed: %v", err)
	}

}

func TestUpdateMetricByPathInvalidType(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	err := service.UpdateMetricByPath(ctx, "invalid/Test/123")
	if err != ErrInvalidType {
		t.Errorf("Expected ErrInvalidType, got %v", err)
	}
}

func TestUpdateMetricByPathInvalidGaugeValue(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	err := service.UpdateMetricByPath(ctx, "gauge/Test/not-a-number")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

func TestUpdateMetricByPathInvalidCounterValue(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	err := service.UpdateMetricByPath(ctx, "counter/Test/not-a-number")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue, got %v", err)
	}
}

func TestUpdateMetricByPathCounterFloat(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	// Counter не должен принимать float
	err := service.UpdateMetricByPath(ctx, "counter/Test/123.456")
	if err != ErrInvalidValue {
		t.Errorf("Expected ErrInvalidValue for float counter, got %v", err)
	}
}

func TestGetMetric(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	// Сохраняем метрику
	val := 123.456
	metric := models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
		Value: &val,
	}
	storage.UpdateMetric(ctx, metric)

	// Получаем метрику
	result, err := service.GetMetric(ctx, "TestGauge", models.Gauge)
	if err != nil {
		t.Errorf("GetMetric failed: %v", err)
	}

	if result.ID != "TestGauge" {
		t.Errorf("Expected ID 'TestGauge', got '%s'", result.ID)
	}

	if result.Value == nil || *result.Value != 123.456 {
		t.Errorf("Expected value 123.456, got %v", result.Value)
	}
}

func TestGetMetricNotFound(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	_, err := service.GetMetric(ctx, "NonExistent", models.Gauge)
	if err != ErrInvalidName {
		t.Errorf("Expected ErrInvalidName, got %v", err)
	}
}

func TestGetMetricInvalidType(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	_, err := service.GetMetric(ctx, "Test", "invalid")
	if err != ErrInvalidType {
		t.Errorf("Expected ErrInvalidType, got %v", err)
	}
}

func TestGetMetricEmptyName(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	_, err := service.GetMetric(ctx, "", models.Gauge)
	if err != ErrInvalidName {
		t.Errorf("Expected ErrInvalidName for empty name, got %v", err)
	}
}

func TestGetAllMetrics(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	// Добавляем несколько метрик
	val1 := 123.456
	storage.UpdateMetric(ctx, models.Metrics{
		ID:    "Gauge1",
		MType: models.Gauge,
		Value: &val1,
	})

	val2 := 789.012
	storage.UpdateMetric(ctx, models.Metrics{
		ID:    "Gauge2",
		MType: models.Gauge,
		Value: &val2,
	})

	delta1 := int64(100)
	storage.UpdateMetric(ctx, models.Metrics{
		ID:    "Counter1",
		MType: models.Counter,
		Delta: &delta1,
	})

	// Получаем все метрики
	all, err := service.GetAllMetrics(ctx)
	if err != nil {
		t.Errorf("GetAllMetrics failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(all))
	}
}

func TestGetAllMetricsEmpty(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	all, err := service.GetAllMetrics(ctx)
	if err != nil {
		t.Errorf("GetAllMetrics failed: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(all))
	}
}
