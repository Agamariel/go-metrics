package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Определяем флаги командной строки
	serverAddress := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		log.Fatalf("Ошибка: неизвестные аргументы: %v", flag.Args())
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

	log.Printf("Server started at http://%s", *serverAddress)
	if err := http.ListenAndServe(*serverAddress, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
