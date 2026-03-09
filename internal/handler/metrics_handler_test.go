package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
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

func (m *MockStorage) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	if metric.MType == models.Gauge && metric.Value != nil {
		m.gauges[metric.ID] = *metric.Value
	}
	if metric.MType == models.Counter && metric.Delta != nil {
		m.counters[metric.ID] += *metric.Delta
	}
	return nil
}

func (m *MockStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		if metric.MType == models.Gauge && metric.Value != nil {
			m.gauges[metric.ID] = *metric.Value
		}
		if metric.MType == models.Counter && metric.Delta != nil {
			m.counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}

func (m *MockStorage) GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error) {
	var result models.Metrics
	result.ID = id
	result.MType = mType

	switch mType {
	case models.Gauge:
		val, ok := m.gauges[id]
		if !ok {
			return models.Metrics{}, repository.ErrNotFound
		}
		result.Value = &val
		return result, nil
	case models.Counter:
		val, ok := m.counters[id]
		if !ok {
			return models.Metrics{}, repository.ErrNotFound
		}
		result.Delta = &val
		return result, nil
	default:
		return models.Metrics{}, repository.ErrNotFound
	}
}

func (m *MockStorage) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
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
	return all, nil
}

func (m *MockStorage) Close() error {
	return nil
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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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
	handler := NewMetricsHandler(svc, nil)

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

func newJSONRequest(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// UpdateMetricJSONHandler

func TestUpdateMetricJSONHandler_Gauge(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	h := NewMetricsHandler(svc, nil)

	body := `{"id":"cpu","type":"gauge","value":0.75}`
	req := newJSONRequest(t, http.MethodPost, "/update/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMetricJSONHandler_Counter(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	h := NewMetricsHandler(svc, nil)

	body := `{"id":"hits","type":"counter","delta":10}`
	req := newJSONRequest(t, http.MethodPost, "/update/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMetricJSONHandler_WrongContentType(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricJSONHandler_InvalidJSON(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/update/", "not json")
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricJSONHandler_MissingFields(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/update/", `{"id":"","type":""}`)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricJSONHandler_GaugeNilValue(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/update/", `{"id":"cpu","type":"gauge"}`)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricJSONHandler_CounterNilDelta(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/update/", `{"id":"hits","type":"counter"}`)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricJSONHandler_InvalidType(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/update/", `{"id":"x","type":"unknown","value":1.0}`)
	w := httptest.NewRecorder()

	h.UpdateMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// GetMetricJSONHandler

func TestGetMetricJSONHandler_Found(t *testing.T) {
	storage := NewMockStorage()
	storage.gauges["cpu"] = 0.5
	h := NewMetricsHandler(service.NewMetricsService(storage), nil)

	req := newJSONRequest(t, http.MethodPost, "/value/", `{"id":"cpu","type":"gauge"}`)
	w := httptest.NewRecorder()

	h.GetMetricJSONHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMetricJSONHandler_NotFound(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/value/", `{"id":"missing","type":"gauge"}`)
	w := httptest.NewRecorder()

	h.GetMetricJSONHandler(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestGetMetricJSONHandler_WrongContentType(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	h.GetMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGetMetricJSONHandler_InvalidJSON(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/value/", "bad json")
	w := httptest.NewRecorder()

	h.GetMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGetMetricJSONHandler_MissingFields(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/value/", `{"id":"","type":""}`)
	w := httptest.NewRecorder()

	h.GetMetricJSONHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// UpdateMetricsBatchHandler

func TestUpdateMetricsBatchHandler_Success(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)

	body := `[{"id":"cpu","type":"gauge","value":1.5},{"id":"hits","type":"counter","delta":5}]`
	req := newJSONRequest(t, http.MethodPost, "/updates/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMetricsBatchHandler_WrongContentType(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader("[]"))
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_EmptyArray(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/updates/", `[]`)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_InvalidJSON(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	req := newJSONRequest(t, http.MethodPost, "/updates/", `not json`)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_MissingID(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	body := `[{"id":"","type":"gauge","value":1.5}]`
	req := newJSONRequest(t, http.MethodPost, "/updates/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_NilGaugeValue(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	body := `[{"id":"cpu","type":"gauge"}]`
	req := newJSONRequest(t, http.MethodPost, "/updates/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_NilCounterDelta(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	body := `[{"id":"hits","type":"counter"}]`
	req := newJSONRequest(t, http.MethodPost, "/updates/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateMetricsBatchHandler_InvalidType(t *testing.T) {
	h := NewMetricsHandler(service.NewMetricsService(NewMockStorage()), nil)
	body := `[{"id":"x","type":"unknown","value":1.0}]`
	req := newJSONRequest(t, http.MethodPost, "/updates/", body)
	w := httptest.NewRecorder()

	h.UpdateMetricsBatchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestListMetricsHandlerEmpty(t *testing.T) {
	storage := NewMockStorage()
	svc := service.NewMetricsService(storage)
	handler := NewMetricsHandler(svc, nil)

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
