package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
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

// Helper функция для создания запроса с chi параметрами
func createRequestWithParams(method, path string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	
	// Создаем chi context с параметрами
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	
	// Добавляем context к запросу
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	
	return req
}
func TestUpdateMetricHandlerGauge(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := createRequestWithParams(http.MethodPost, "/update/gauge/TestGauge/123.456", map[string]string{
		"type":  "gauge",
		"name":  "TestGauge",
		"value": "123.456",
	})
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

	req := createRequestWithParams(http.MethodPost, "/update/counter/TestCounter/42", map[string]string{
		"type":  "counter",
		"name":  "TestCounter",
		"value": "42",
	})
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
		name   string
		params map[string]string
	}{
		{"empty type", map[string]string{"type": "", "name": "test", "value": "123"}},
		{"empty name", map[string]string{"type": "gauge", "name": "", "value": "123"}},
		{"empty value", map[string]string{"type": "gauge", "name": "test", "value": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequestWithParams(http.MethodPost, "/update/", tt.params)
			w := httptest.NewRecorder()

			handler.UpdateMetricHandler(w, req)

			if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
				t.Errorf("Expected status 400 or 404 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestUpdateMetricHandlerInvalidType(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := createRequestWithParams(http.MethodPost, "/update/invalid/TestMetric/123", map[string]string{
		"type":  "invalid",
		"name":  "TestMetric",
		"value": "123",
	})
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
			req := createRequestWithParams(http.MethodPost, path, map[string]string{
				"type":  tt.mType,
				"name":  "Test",
				"value": tt.value,
			})
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
	req1 := createRequestWithParams(http.MethodPost, "/update/counter/TestCounter/10", map[string]string{
		"type":  "counter",
		"name":  "TestCounter",
		"value": "10",
	})
	w1 := httptest.NewRecorder()
	handler.UpdateMetricHandler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	// Второй запрос
	req2 := createRequestWithParams(http.MethodPost, "/update/counter/TestCounter/5", map[string]string{
		"type":  "counter",
		"name":  "TestCounter",
		"value": "5",
	})
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
