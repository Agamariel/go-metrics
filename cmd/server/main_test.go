package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Создаем тестовый сервер
func setupTestServer() *chi.Mux {
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(metricsService, nil)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)
	r.Get("/", h.ListMetricsHandler)

	return r
}

func TestIntegrationFlow(t *testing.T) {
	router := setupTestServer()
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Тест 1: Добавление gauge метрики
	resp, err := http.Post(ts.URL+"/update/gauge/TestGauge/123.456", "text/plain", nil)
	if err != nil {
		t.Fatalf("Failed to post gauge: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for gauge update, got %d", http.StatusOK, resp.StatusCode)
	}
	resp.Body.Close()

	// Тест 2: Добавление counter метрики
	resp, err = http.Post(ts.URL+"/update/counter/TestCounter/42", "text/plain", nil)
	if err != nil {
		t.Fatalf("Failed to post counter: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for counter update, got %d", http.StatusOK, resp.StatusCode)
	}
	resp.Body.Close()

	// Тест 3: Получение gauge метрики
	resp, err = http.Get(ts.URL + "/value/gauge/TestGauge")
	if err != nil {
		t.Fatalf("Failed to get gauge: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for gauge get, got %d", http.StatusOK, resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "123.456" {
		t.Errorf("Expected gauge value '123.456', got '%s'", string(body))
	}
	resp.Body.Close()

	// Тест 4: Получение counter метрики
	resp, err = http.Get(ts.URL + "/value/counter/TestCounter")
	if err != nil {
		t.Fatalf("Failed to get counter: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for counter get, got %d", http.StatusOK, resp.StatusCode)
	}
	body, _ = io.ReadAll(resp.Body)
	if string(body) != "42" {
		t.Errorf("Expected counter value '42', got '%s'", string(body))
	}
	resp.Body.Close()

	// Тест 5: Получение несуществующей метрики
	resp, err = http.Get(ts.URL + "/value/gauge/NonExistent")
	if err != nil {
		t.Fatalf("Failed to get non-existent metric: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d for non-existent metric, got %d", http.StatusNotFound, resp.StatusCode)
	}
	resp.Body.Close()

	// Тест 6: Получение HTML страницы со всеми метриками
	resp, err = http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("Failed to get metrics list: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d for metrics list, got %d", http.StatusOK, resp.StatusCode)
	}
	body, _ = io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "TestGauge") {
		t.Error("Metrics list should contain TestGauge")
	}
	if !strings.Contains(bodyStr, "TestCounter") {
		t.Error("Metrics list should contain TestCounter")
	}
	// Проверяем значение gauge (может быть в формате 123.456000)
	if !strings.Contains(bodyStr, "123.456") && !strings.Contains(bodyStr, "123.4560") {
		t.Errorf("Metrics list should contain gauge value, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "42") {
		t.Error("Metrics list should contain counter value")
	}
	resp.Body.Close()

	// Тест 7: Накопление counter метрики
	resp, err = http.Post(ts.URL+"/update/counter/TestCounter/10", "text/plain", nil)
	if err != nil {
		t.Fatalf("Failed to update counter: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/value/counter/TestCounter")
	if err != nil {
		t.Fatalf("Failed to get accumulated counter: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	if string(body) != "52" { // 42 + 10
		t.Errorf("Expected accumulated counter value '52', got '%s'", string(body))
	}
	resp.Body.Close()
}

func TestInvalidRequests(t *testing.T) {
	router := setupTestServer()
	ts := httptest.NewServer(router)
	defer ts.Close()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{"Invalid metric type", ts.URL + "/update/invalid/Test/123", http.StatusBadRequest},
		{"Invalid gauge value", ts.URL + "/update/gauge/Test/notanumber", http.StatusBadRequest},
		{"Invalid counter value", ts.URL + "/update/counter/Test/notanumber", http.StatusBadRequest},
		{"Counter with float", ts.URL + "/update/counter/Test/123.456", http.StatusBadRequest},
		{"Get invalid type", ts.URL + "/value/invalid/Test", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			if strings.Contains(tt.url, "/update/") {
				resp, err = http.Post(tt.url, "text/plain", nil)
			} else {
				resp, err = http.Get(tt.url)
			}
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer func() {
				if resp != nil {
					resp.Body.Close()
				}
			}()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}
