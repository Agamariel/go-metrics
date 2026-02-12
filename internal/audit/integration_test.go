package audit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/audit"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Интеграционный тест: проверяем, что аудит работает end-to-end
func TestAudit_Integration_FileObserver(t *testing.T) {
	// Создаем временный файл для аудита
	tmpDir := t.TempDir()
	auditFile := filepath.Join(tmpDir, "audit.log")

	// Создаем logger
	zapLogger, _ := zap.NewDevelopment()
	log := logger.NewZapAdapter(zapLogger)

	// Создаем систему аудита
	publisher := audit.NewPublisher(log)
	fileObserver, err := audit.NewFileObserver(auditFile)
	if err != nil {
		t.Fatalf("Failed to create FileObserver: %v", err)
	}
	publisher.Attach(fileObserver)

	// Создаем сервис метрик и handler
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(metricsService, publisher)

	// Создаем роутер
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Post("/update/", h.UpdateMetricJSONHandler)
	r.Post("/updates/", h.UpdateMetricsBatchHandler)

	// Отправляем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/123.456", nil)
	req.RemoteAddr = "192.168.0.1:12345"

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "TestMetric")
	rctx.URLParams.Add("value", "123.456")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Закрываем publisher и ждем завершения всех операций
	publisher.Close()

	// Читаем файл аудита
	data, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(lines))
	}

	// Проверяем содержимое события
	var event audit.Event
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("Failed to unmarshal audit event: %v", err)
	}

	if len(event.Metrics) != 1 || event.Metrics[0] != "TestMetric" {
		t.Errorf("Expected metric 'TestMetric', got %v", event.Metrics)
	}

	if !strings.Contains(event.IPAddress, "192.168.0.1") {
		t.Errorf("Expected IP containing '192.168.0.1', got '%s'", event.IPAddress)
	}

	if event.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
}

// Интеграционный тест: проверяем HTTP аудит
func TestAudit_Integration_HTTPObserver(t *testing.T) {
	// Создаем тестовый HTTP сервер для приема событий аудита
	receivedEvents := []audit.Event{}
	auditServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event audit.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("Failed to decode audit event: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedEvents = append(receivedEvents, event)
		w.WriteHeader(http.StatusOK)
	}))
	defer auditServer.Close()

	// Создаем logger
	zapLogger, _ := zap.NewDevelopment()
	log := logger.NewZapAdapter(zapLogger)

	// Создаем систему аудита с HTTP observer
	publisher := audit.NewPublisher(log)
	httpObserver := audit.NewHTTPObserver(auditServer.URL)
	publisher.Attach(httpObserver)

	// Создаем сервис метрик и handler
	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	h := handler.NewMetricsHandler(metricsService, publisher)

	// Создаем роутер
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)

	// Отправляем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/update/counter/TestCounter/42", nil)
	req.RemoteAddr = "10.0.0.1:54321"

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "counter")
	rctx.URLParams.Add("name", "TestCounter")
	rctx.URLParams.Add("value", "42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.UpdateMetricHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Закрываем publisher и ждем завершения всех операций
	publisher.Close()

	// Даем небольшую задержку на доставку
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что событие получено
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 audit event, got %d", len(receivedEvents))
	}

	event := receivedEvents[0]
	if len(event.Metrics) != 1 || event.Metrics[0] != "TestCounter" {
		t.Errorf("Expected metric 'TestCounter', got %v", event.Metrics)
	}

	if !strings.Contains(event.IPAddress, "10.0.0.1") {
		t.Errorf("Expected IP containing '10.0.0.1', got '%s'", event.IPAddress)
	}
}
