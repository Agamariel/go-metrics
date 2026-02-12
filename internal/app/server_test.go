package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
)

func TestApp_Shutdown(t *testing.T) {
	app := &App{}

	// Инициализируем необходимые компоненты
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	app.config = &config.ServerConfig{
		Address: "localhost:0", // Используем динамический порт
	}

	// Инициализируем хранилище
	app.storage = repository.NewMemStorage()

	// Создаём сервис и handler
	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, nil)
	dbHandler := handler.NewDBHandler(nil, app.logger)

	// Настраиваем роутер
	router := app.setupRouter(metricsHandler, dbHandler)

	// Создаём HTTP сервер
	app.server = &http.Server{
		Addr:    app.config.Address,
		Handler: router,
	}

	// Запускаем сервер в горутине
	go func() {
		app.Run()
	}()

	// Даём серверу немного времени на запуск
	time.Sleep(100 * time.Millisecond)

	// Выполняем shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestApp_Shutdown_WithAudit(t *testing.T) {
	app := &App{}

	// Инициализируем необходимые компоненты
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	tmpDir := t.TempDir()
	auditFile := tmpDir + "/audit.log"

	app.config = &config.ServerConfig{
		Address:   "localhost:0",
		AuditFile: auditFile,
	}

	// Инициализируем аудит
	if err := app.initAudit(); err != nil {
		t.Fatalf("initAudit failed: %v", err)
	}

	// Инициализируем хранилище
	app.storage = repository.NewMemStorage()

	// Создаём сервис и handler
	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, app.publisher)
	dbHandler := handler.NewDBHandler(nil, app.logger)

	// Настраиваем роутер
	router := app.setupRouter(metricsHandler, dbHandler)

	// Создаём HTTP сервер
	app.server = &http.Server{
		Addr:    app.config.Address,
		Handler: router,
	}

	// Запускаем сервер в горутине
	go func() {
		app.Run()
	}()

	// Даём серверу немного времени на запуск
	time.Sleep(100 * time.Millisecond)

	// Выполняем shutdown с аудитом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown with audit failed: %v", err)
	}
}
