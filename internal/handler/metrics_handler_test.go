package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/service"
)

// MockStorage для тестирования
type MockStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MockStorage) UpdateMetric(metric models.Metrics) error {
	if metric.MType == models.Gauge && metric.Value != nil {
		m.gauges[metric.ID] = *metric.Value
	}
	if metric.MType == models.Counter && metric.Delta != nil {
		m.counters[metric.ID] += *metric.Delta
	}
	return nil
}
func TestUpdateMetricHandlerGauge(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestGauge/123.456", nil)
	w := httptest.NewRecorder()

	handler.UpdateMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Проверяем, что метрика сохранена
	if val, ok := storage.gauges["TestGauge"]; !ok {
		t.Error("Gauge metric not saved")
	} else if val != 123.456 {
		t.Errorf("Expected value 123.456, got %f", val)
	}
}

func TestUpdateMetricHandlerCounter(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/TestCounter/42", nil)
	w := httptest.NewRecorder()

	handler.UpdateMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Проверяем, что метрика сохранена
	if val, ok := storage.counters["TestCounter"]; !ok {
		t.Error("Counter metric not saved")
	} else if val != 42 {
		t.Errorf("Expected value 42, got %d", val)
	}
}

func TestUpdateMetricHandlerInvalidPath(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	tests := []struct {
		name string
		path string
	}{
		{"too few parts", "/update/gauge"},
		{"too many parts", "/update/gauge/name/value/extra"},
		{"empty path", "/update/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			w := httptest.NewRecorder()

			handler.UpdateMetricHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestUpdateMetricHandlerInvalidType(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/update/invalid/TestMetric/123", nil)
	w := httptest.NewRecorder()

	handler.UpdateMetricHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUpdateMetricHandlerInvalidValue(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	tests := []struct {
		name  string
		mType string
		value string
	}{
		{"gauge invalid", models.Gauge, "not-a-number"},
		{"counter invalid", models.Counter, "not-a-number"},
		{"counter float", models.Counter, "123.456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/update/" + tt.mType + "/Test/" + tt.value
			req := httptest.NewRequest(http.MethodPost, path, nil)
			w := httptest.NewRecorder()

			handler.UpdateMetricHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestUpdateMetricHandlerCounterAccumulation(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	// Первый запрос
	req1 := httptest.NewRequest(http.MethodPost, "/update/counter/TestCounter/10", nil)
	w1 := httptest.NewRecorder()
	handler.UpdateMetricHandler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	// Второй запрос
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/TestCounter/5", nil)
	w2 := httptest.NewRecorder()
	handler.UpdateMetricHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request failed with status %d", w2.Code)
	}

	// Проверяем накопление
	if val, ok := storage.counters["TestCounter"]; !ok {
		t.Error("Counter not found")
	} else if val != 15 {
		t.Errorf("Expected accumulated value 15, got %d", val)
	}
}
