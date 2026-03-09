package app

import (
	"github.com/Agamariel/go-metrics/internal/handler"
	custommiddleware "github.com/Agamariel/go-metrics/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// setupRouter настраивает роутер с middleware и маршрутами
func (a *App) setupRouter(metricsHandler *handler.MetricsHandler, dbHandler *handler.DBHandler) *chi.Mux {
	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(custommiddleware.TrustedSubnetMiddleware(a.config.TrustedSubnet)) // Проверка доверенной подсети
	r.Use(custommiddleware.CryptoMiddleware(a.privateKey))                  // Расшифровка (первым — до gzip)
	r.Use(custommiddleware.GzipMiddleware)                                  // Сжатие gzip
	r.Use(custommiddleware.HashMiddleware(a.config.Key))   // Проверка и добавление хеша
	r.Use(custommiddleware.Logger(a.logger))             // Кастомное логирование
	r.Use(middleware.Recoverer)                          // Восстановление после паники
	r.Use(middleware.RequestID)                          // Добавление request ID
	r.Use(middleware.RealIP)                             // Определение реального IP клиента

	// Настраиваем маршруты
	r.Post("/update/{type}/{name}/{value}", metricsHandler.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", metricsHandler.GetMetricHandler)

	// JSON API эндпоинты
	r.Post("/update/", metricsHandler.UpdateMetricJSONHandler)
	r.Post("/updates/", metricsHandler.UpdateMetricsBatchHandler)
	r.Post("/value/", metricsHandler.GetMetricJSONHandler)

	// Проверка соединения с БД
	r.Get("/ping", dbHandler.PingDB)

	// Список всех метрик
	r.Get("/", metricsHandler.ListMetricsHandler)

	return r
}
