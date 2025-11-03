package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config содержит конфигурацию сервера
type Config struct {
	Address string `env:"ADDRESS"`
}

func main() {
	// Значения по умолчанию
	cfg := Config{
		Address: "localhost:8080",
	}

	// Определяем флаги командной строки
	serverAddress := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		log.Fatalf("Ошибка: неизвестные аргументы: %v", flag.Args())
	}

	// Применяем значение из флага
	cfg.Address = *serverAddress

	// Парсим переменные окружения (приоритет выше флагов)
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Ошибка при парсинге переменных окружения: %v", err)
	}

	// Инициализируем in-memory хранилище
	storage := repository.NewMemStorage()

	// Создаём сервис с бизнес-логикой
	metricsService := service.NewMetricsService(storage)

	h := handler.NewMetricsHandler(metricsService)
	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(middleware.Logger)    // Логирование запросов
	r.Use(middleware.Recoverer) // Восстановление после паники
	r.Use(middleware.RequestID) // Добавление request ID
	r.Use(middleware.RealIP)    // Определение реального IP клиента

	// Настраиваем маршруты с использованием chi
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)
	r.Get("/", h.ListMetricsHandler)

	log.Printf("Server started at http://%s", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
