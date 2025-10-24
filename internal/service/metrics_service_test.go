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

func (m *MockStorage) GetMetric(id string, mType string) (models.Metrics, bool) {
	key := id + ":" + mType
	metric, found := m.metrics[key]
	return metric, found
}

func (m *MockStorage) GetAllMetrics() []models.Metrics {
	var all []models.Metrics
	for _, metric := range m.metrics {
		all = append(all, metric)
	}
	return all
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

func TestGetMetric(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	// Сохраняем метрику
	val := 123.456
	metric := models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
		Value: &val,
	}
	storage.UpdateMetric(metric)

	// Получаем метрику
	result, err := service.GetMetric("TestGauge", models.Gauge)
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

	_, err := service.GetMetric("NonExistent", models.Gauge)
	if err != ErrInvalidName {
		t.Errorf("Expected ErrInvalidName, got %v", err)
	}
}

func TestGetMetricInvalidType(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	_, err := service.GetMetric("Test", "invalid")
	if err != ErrInvalidType {
		t.Errorf("Expected ErrInvalidType, got %v", err)
	}
}

func TestGetMetricEmptyName(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	_, err := service.GetMetric("", models.Gauge)
	if err != ErrInvalidName {
		t.Errorf("Expected ErrInvalidName for empty name, got %v", err)
	}
}

func TestGetAllMetrics(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	// Добавляем несколько метрик
	val1 := 123.456
	storage.UpdateMetric(models.Metrics{
		ID:    "Gauge1",
		MType: models.Gauge,
		Value: &val1,
	})

	val2 := 789.012
	storage.UpdateMetric(models.Metrics{
		ID:    "Gauge2",
		MType: models.Gauge,
		Value: &val2,
	})

	delta1 := int64(100)
	storage.UpdateMetric(models.Metrics{
		ID:    "Counter1",
		MType: models.Counter,
		Delta: &delta1,
	})

	// Получаем все метрики
	all := service.GetAllMetrics()

	if len(all) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(all))
	}
}

func TestGetAllMetricsEmpty(t *testing.T) {
	storage := NewMockStorage()
	service := NewMetricsService(storage)

	all := service.GetAllMetrics()

	if len(all) != 0 {
		t.Errorf("Expected 0 metrics, got %d", len(all))
	}
}
