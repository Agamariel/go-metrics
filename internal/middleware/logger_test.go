package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger(t *testing.T) {
	// Создаём логгер для тестирования
	core, recorded := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	// Создаём тестовый обработчик
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Оборачиваем обработчик в middleware
	middleware := Logger(logger)
	wrappedHandler := middleware(handler)

	// Создаём тестовый запрос
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Выполняем запрос
	wrappedHandler.ServeHTTP(w, req)

	// Проверяем результаты
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Проверяем, что лог был записан
	logs := recorded.All()
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	logEntry := logs[0]

	// Проверяем сообщение
	if logEntry.Message != "HTTP request" {
		t.Errorf("Expected message 'HTTP request', got '%s'", logEntry.Message)
	}

	// Проверяем наличие всех необходимых полей
	fields := logEntry.ContextMap()

	if method, ok := fields["method"].(string); !ok || method != "GET" {
		t.Errorf("Expected method 'GET', got '%v'", fields["method"])
	}

	if uri, ok := fields["uri"].(string); !ok || uri != "/test" {
		t.Errorf("Expected uri '/test', got '%v'", fields["uri"])
	}

	if status, ok := fields["status"].(int64); !ok || status != 200 {
		t.Errorf("Expected status 200, got '%v'", fields["status"])
	}

	if size, ok := fields["size"].(int64); !ok || size != 13 {
		t.Errorf("Expected size 13, got '%v'", fields["size"])
	}

	if _, ok := fields["duration"]; !ok {
		t.Error("Expected duration field to be present")
	}
}

func TestLoggerWithStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{"200 OK", http.StatusOK, http.StatusOK},
		{"404 Not Found", http.StatusNotFound, http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError, http.StatusInternalServerError},
		{"503 Service Unavailable", http.StatusServiceUnavailable, http.StatusServiceUnavailable},
		{"201 Created", http.StatusCreated, http.StatusCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём логгер
			core, recorded := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			// Создаём тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			// Оборачиваем обработчик в middleware
			middleware := Logger(logger)
			wrappedHandler := middleware(handler)

			// Создаём тестовый запрос
			req := httptest.NewRequest("POST", "/test", nil)
			w := httptest.NewRecorder()

			// Выполняем запрос
			wrappedHandler.ServeHTTP(w, req)

			// Проверяем статус код
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Проверяем лог
			logs := recorded.All()
			if len(logs) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(logs))
			}

			fields := logs[0].ContextMap()
			if status, ok := fields["status"].(int64); !ok || int(status) != tt.expectedStatus {
				t.Errorf("Expected logged status %d, got '%v'", tt.expectedStatus, fields["status"])
			}
		})
	}
}

func TestLoggerResponseSize(t *testing.T) {
	testCases := []struct {
		name         string
		responseBody string
		expectedSize int
	}{
		{"Empty response", "", 0},
		{"Small response", "OK", 2},
		{"Medium response", "This is a test", 14},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Создаём наблюдаемый логгер
			core, recorded := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			// Создаём тестовый обработчик
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.responseBody))
			})

			// Оборачиваем обработчик в middleware
			middleware := Logger(logger)
			wrappedHandler := middleware(handler)

			// Создаём тестовый запрос
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Выполняем запрос
			wrappedHandler.ServeHTTP(w, req)

			// Проверяем лог
			logs := recorded.All()
			if len(logs) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(logs))
			}

			fields := logs[0].ContextMap()
			if size, ok := fields["size"].(int64); !ok || int(size) != tc.expectedSize {
				t.Errorf("Expected size %d, got '%v'", tc.expectedSize, fields["size"])
			}
		})
	}
}
