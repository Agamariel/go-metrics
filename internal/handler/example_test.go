package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

// Example демонстрирует создание и использование MetricsHandler.
func Example() {
	// Создаём хранилище и сервис
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)

	// Создаём обработчик без аудита
	h := handler.NewMetricsHandler(svc, nil)

	// Настраиваем роутер
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)
	r.Post("/update", h.UpdateMetricJSONHandler)
	r.Post("/value", h.GetMetricJSONHandler)
	r.Post("/updates/", h.UpdateMetricsBatchHandler)
	r.Get("/", h.ListMetricsHandler)

	// Запускаем тестовый сервер
	ts := httptest.NewServer(r)
	defer ts.Close()

	fmt.Println("Сервер запущен")
	// Output: Сервер запущен
}

// ExampleMetricsHandler_UpdateMetricHandler демонстрирует обновление метрики через URL.
func ExampleMetricsHandler_UpdateMetricHandler() {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(svc, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Обновляем gauge-метрику
	resp, _ := http.Post(ts.URL+"/update/gauge/temperature/36.6", "", nil)
	fmt.Printf("Gauge update: %d\n", resp.StatusCode)
	resp.Body.Close()

	// Обновляем counter-метрику
	resp, _ = http.Post(ts.URL+"/update/counter/requests/1", "", nil)
	fmt.Printf("Counter update: %d\n", resp.StatusCode)
	resp.Body.Close()

	// Output:
	// Gauge update: 200
	// Counter update: 200
}

// ExampleMetricsHandler_GetMetricHandler демонстрирует получение значения метрики.
func ExampleMetricsHandler_GetMetricHandler() {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(svc, nil)

	// Предварительно сохраняем метрику
	value := 42.5
	storage.UpdateMetric(context.Background(), models.Metrics{
		ID:    "cpu",
		MType: models.Gauge,
		Value: &value,
	})

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := http.Get(ts.URL + "/value/gauge/cpu")
	defer resp.Body.Close()

	var body bytes.Buffer
	body.ReadFrom(resp.Body)
	fmt.Printf("Value: %s\n", body.String())

	// Output:
	// Value: 42.5
}

// ExampleMetricsHandler_UpdateMetricJSONHandler демонстрирует обновление метрики через JSON.
func ExampleMetricsHandler_UpdateMetricJSONHandler() {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(svc, nil)

	r := chi.NewRouter()
	r.Post("/update", h.UpdateMetricJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Отправляем gauge-метрику
	value := 98.6
	metric := models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	}
	body, _ := json.Marshal(metric)

	resp, _ := http.Post(ts.URL+"/update", "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)

	// Output:
	// Status: 200
}

// ExampleMetricsHandler_UpdateMetricsBatchHandler демонстрирует пакетное обновление метрик.
func ExampleMetricsHandler_UpdateMetricsBatchHandler() {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(svc, nil)

	r := chi.NewRouter()
	r.Post("/updates/", h.UpdateMetricsBatchHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Подготавливаем пакет метрик
	value1 := 42.5
	value2 := 36.6
	delta := int64(100)

	metrics := []models.Metrics{
		{ID: "cpu", MType: models.Gauge, Value: &value1},
		{ID: "temperature", MType: models.Gauge, Value: &value2},
		{ID: "requests", MType: models.Counter, Delta: &delta},
	}
	body, _ := json.Marshal(metrics)

	resp, _ := http.Post(ts.URL+"/updates/", "application/json", bytes.NewReader(body))
	defer resp.Body.Close()

	fmt.Printf("Batch update status: %d\n", resp.StatusCode)

	// Output:
	// Batch update status: 200
}

// ExampleMetricsHandler_GetMetricJSONHandler демонстрирует получение метрики через JSON API.
func ExampleMetricsHandler_GetMetricJSONHandler() {
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(svc, nil)

	// Предварительно сохраняем метрику
	delta := int64(42)
	storage.UpdateMetric(context.Background(), models.Metrics{
		ID:    "requests",
		MType: models.Counter,
		Delta: &delta,
	})

	r := chi.NewRouter()
	r.Post("/value", h.GetMetricJSONHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Запрашиваем метрику
	request := models.Metrics{
		ID:    "requests",
		MType: models.Counter,
	}
	reqBody, _ := json.Marshal(request)

	resp, _ := http.Post(ts.URL+"/value", "application/json", bytes.NewReader(reqBody))
	defer resp.Body.Close()

	var result models.Metrics
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("Metric: %s, Delta: %d\n", result.ID, *result.Delta)

	// Output:
	// Metric: requests, Delta: 42
}
