package agent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

func TestNewMetricsSender(t *testing.T) {
	sender := NewMetricsSender("http://localhost:8080")

	if sender == nil {
		t.Fatal("NewMetricsSender returned nil")
	}

	if sender.serverURL != "http://localhost:8080" {
		t.Errorf("serverURL expected 'http://localhost:8080', got '%s'", sender.serverURL)
	}

	if sender.client == nil {
		t.Error("HTTP client is nil")
	}
}

func TestSendMetric(t *testing.T) {
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Проверяем Content-Type
		contentType := r.Header.Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("Expected Content-Type 'text/plain', got '%s'", contentType)
		}

		// Проверяем URL
		expectedPath := "/update/gauge/TestMetric/123.456"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewMetricsSender(server.URL)

	err := sender.SendMetric("gauge", "TestMetric", "123.456")
	if err != nil {
		t.Errorf("SendMetric failed: %v", err)
	}
}

func TestSendMetricServerError(t *testing.T) {
	// Создаем тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := NewMetricsSender(server.URL)

	err := sender.SendMetric("gauge", "TestMetric", "123.456")
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestSendAllMetrics(t *testing.T) {
	receivedMetrics := make(map[string]models.Metrics)

	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что это JSON API
		if r.URL.Path == "/update/" {
			// Декодируем JSON
			body, _ := io.ReadAll(r.Body)
			var metric models.Metrics
			if err := json.Unmarshal(body, &metric); err != nil {
				t.Errorf("Failed to decode JSON: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			receivedMetrics[metric.ID] = metric
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewMetricsSender(server.URL)

	gauges := map[string]float64{
		"Metric1": 123.456,
		"Metric2": 789.012,
	}

	counters := map[string]int64{
		"Counter1": 100,
		"Counter2": 200,
	}

	err := sender.SendAllMetrics(gauges, counters)
	if err != nil {
		t.Errorf("SendAllMetrics failed: %v", err)
	}

	// Проверяем, что все метрики были отправлены
	expectedMetrics := map[string]string{
		"Metric1":  models.Gauge,
		"Metric2":  models.Gauge,
		"Counter1": models.Counter,
		"Counter2": models.Counter,
	}

	for name, expectedType := range expectedMetrics {
		metric, exists := receivedMetrics[name]
		if !exists {
			t.Errorf("Metric not received: %s", name)
			continue
		}
		if metric.MType != expectedType {
			t.Errorf("Metric %s: expected type %s, got %s", name, expectedType, metric.MType)
		}
	}
}

func TestSendAllMetricsWithError(t *testing.T) {
	// Создаем тестовый сервер, который возвращает ошибку
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewMetricsSender(server.URL)

	gauges := map[string]float64{
		"Metric1": 123.456,
	}

	counters := map[string]int64{}

	err := sender.SendAllMetrics(gauges, counters)
	if err == nil {
		t.Error("Expected error when server returns error, got nil")
	}
}
