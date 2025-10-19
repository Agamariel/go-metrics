package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
	receivedMetrics := make(map[string]string)

	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMetrics[r.URL.Path] = r.Method
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
	expectedPaths := []string{
		"/update/gauge/Metric1/123.456000",
		"/update/gauge/Metric2/789.012000",
		"/update/counter/Counter1/100",
		"/update/counter/Counter2/200",
	}

	for _, path := range expectedPaths {
		if _, exists := receivedMetrics[path]; !exists {
			t.Errorf("Metric not received: %s", path)
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
