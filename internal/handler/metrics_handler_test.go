package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (m *MockStorage) GetMetric(id string, mType string) (models.Metrics, bool) {
	var result models.Metrics
	result.ID = id
	result.MType = mType

	switch mType {
	case models.Gauge:
		val, ok := m.gauges[id]
		if !ok {
			return models.Metrics{}, false
		}
		result.Value = &val
		return result, true
	case models.Counter:
		val, ok := m.counters[id]
		if !ok {
			return models.Metrics{}, false
		}
		result.Delta = &val
		return result, true
	default:
		return models.Metrics{}, false
	}
}

func (m *MockStorage) GetAllMetrics() []models.Metrics {
	var all []models.Metrics
	for id, val := range m.gauges {
		v := val
		all = append(all, models.Metrics{
			ID:    id,
			MType: models.Gauge,
			Value: &v,
		})
	}
	for id, val := range m.counters {
		v := val
		all = append(all, models.Metrics{
			ID:    id,
			MType: models.Counter,
			Delta: &v,
		})
	}
	return all
}

func createRequestWithParams(method, path string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
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
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
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
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
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
			t.Errorf("Expected status %d or %d for %s, got %d", http.StatusBadRequest, http.StatusNotFound, tt.name, w.Code)
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
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
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
				t.Errorf("Expected status %d for %s, got %d", http.StatusBadRequest, tt.name, w.Code)
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

func TestGetMetricHandlerGauge(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	// Сначала сохраняем метрику
	val := 123.456
	storage.gauges["TestGauge"] = val

	req := createRequestWithParams(http.MethodGet, "/value/gauge/TestGauge", map[string]string{
		"type": "gauge",
		"name": "TestGauge",
	})
	w := httptest.NewRecorder()

	handler.GetMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := "123.456"
	if w.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, w.Body.String())
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestGetMetricHandlerCounter(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	// Сначала сохраняем метрику
	storage.counters["TestCounter"] = 42

	req := createRequestWithParams(http.MethodGet, "/value/counter/TestCounter", map[string]string{
		"type": "counter",
		"name": "TestCounter",
	})
	w := httptest.NewRecorder()

	handler.GetMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := "42"
	if w.Body.String() != expected {
		t.Errorf("Expected body '%s', got '%s'", expected, w.Body.String())
	}
}

func TestGetMetricHandlerNotFound(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := createRequestWithParams(http.MethodGet, "/value/gauge/NonExistent", map[string]string{
		"type": "gauge",
		"name": "NonExistent",
	})
	w := httptest.NewRecorder()

	handler.GetMetricHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetMetricHandlerInvalidType(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := createRequestWithParams(http.MethodGet, "/value/invalid/TestMetric", map[string]string{
		"type": "invalid",
		"name": "TestMetric",
	})
	w := httptest.NewRecorder()

	handler.GetMetricHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestListMetricsHandler(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	// Добавляем несколько метрик
	storage.gauges["Gauge1"] = 123.456
	storage.gauges["Gauge2"] = 789.012
	storage.counters["Counter1"] = 100
	storage.counters["Counter2"] = 200

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ListMetricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
	}

	body := w.Body.String()

	// Проверяем, что в HTML есть названия метрик
	if !strings.Contains(body, "Gauge1") {
		t.Error("HTML does not contain 'Gauge1'")
	}
	if !strings.Contains(body, "Gauge2") {
		t.Error("HTML does not contain 'Gauge2'")
	}
	if !strings.Contains(body, "Counter1") {
		t.Error("HTML does not contain 'Counter1'")
	}
	if !strings.Contains(body, "Counter2") {
		t.Error("HTML does not contain 'Counter2'")
	}
}

func TestListMetricsHandlerEmpty(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ListMetricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Нет доступных метрик") {
		t.Error("HTML should show empty metrics message")
	}
}
