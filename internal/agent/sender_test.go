package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

func TestNewMetricsSender(t *testing.T) {
	sender, err := NewMetricsSender("http://localhost:8080", "", "")
	if err != nil {
		t.Fatalf("NewMetricsSender returned error: %v", err)
	}

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

	sender, err := NewMetricsSender(server.URL, "", "")
	if err != nil {
		t.Fatalf("NewMetricsSender returned error: %v", err)
	}

	err = sender.SendMetric("gauge", "TestMetric", "123.456")
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

	sender, err := NewMetricsSender(server.URL, "", "")
	if err != nil {
		t.Fatalf("NewMetricsSender returned error: %v", err)
	}

	err = sender.SendMetric("gauge", "TestMetric", "123.456")
	if err == nil {
		t.Error("Expected error for server error, got nil")
	}
}

func TestSendAllMetrics(t *testing.T) {
	receivedMetrics := make(map[string]models.Metrics)

	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что это батч API
		if r.URL.Path == "/updates/" {
			// Читаем тело запроса
			var reader io.Reader = r.Body

			// Если данные сжаты, распаковываем
			if r.Header.Get("Content-Encoding") == "gzip" {
				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					t.Errorf("Failed to create gzip reader: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer gz.Close()
				reader = gz
			}

			// Декодируем JSON массив метрик
			body, _ := io.ReadAll(reader)
			var metrics []models.Metrics
			if err := json.Unmarshal(body, &metrics); err != nil {
				t.Errorf("Failed to decode JSON: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Сохраняем все полученные метрики
			for _, metric := range metrics {
				receivedMetrics[metric.ID] = metric
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender, err := NewMetricsSender(server.URL, "", "")
	if err != nil {
		t.Fatalf("NewMetricsSender returned error: %v", err)
	}

	gauges := map[string]float64{
		"Metric1": 123.456,
		"Metric2": 789.012,
	}

	counters := map[string]int64{
		"Counter1": 100,
		"Counter2": 200,
	}

	ctx := context.Background()
	err = sender.SendAllMetrics(ctx, gauges, counters)
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
		// Проверяем, что запрос идет на правильный эндпоинт
		if r.URL.Path != "/updates/" {
			t.Errorf("Expected path '/updates/', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	sender, err := NewMetricsSender(server.URL, "", "")
	if err != nil {
		t.Fatalf("NewMetricsSender returned error: %v", err)
	}

	gauges := map[string]float64{
		"Metric1": 123.456,
	}

	counters := map[string]int64{}

	ctx := context.Background()
	err = sender.SendAllMetrics(ctx, gauges, counters)
	if err == nil {
		t.Error("Expected error when server returns error, got nil")
	}
}
