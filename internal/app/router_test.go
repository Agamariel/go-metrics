package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
)

func TestApp_SetupRouter(t *testing.T) {
	app := &App{}

	// Инициализируем необходимые компоненты
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	app.config = &config.ServerConfig{
		Address: "localhost:8080",
		Key:     "test-key",
	}

	app.storage = repository.NewMemStorage()
	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, nil)
	dbHandler := handler.NewDBHandler(nil, app.logger)

	// Настраиваем роутер
	router := app.setupRouter(metricsHandler, dbHandler)

	if router == nil {
		t.Fatal("setupRouter returned nil")
	}

	// Проверяем, что маршруты настроены
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"update path", "POST", "/update/gauge/test/1.5"},
		{"get value", "GET", "/value/gauge/test"},
		{"update json", "POST", "/update/"},
		{"batch update", "POST", "/updates/"},
		{"get json", "POST", "/value/"},
		{"ping", "GET", "/ping"},
		{"list metrics", "GET", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Проверяем, что роутер обработал запрос (не 404)
			// Может быть ошибка валидации, но не 404
			if rr.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not found", tt.method, tt.path)
			}
		})
	}
}
