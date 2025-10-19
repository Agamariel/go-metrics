package main

import (
	"log"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
)

func main() {
	// Инициализируем in-memory хранилище
	storage := repository.NewMemStorage()

	// Создаём сервис с бизнес-логикой (если нужно — пока просто передаем storage)
	metricsService := service.NewMetricsService(storage)

	// Создаём HTTP-хэндлер
	h := handler.NewMetricsHandler(metricsService)

	// Настраиваем маршруты
	http.HandleFunc("/update/", h.UpdateMetricHandler)

	log.Println("Server started at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
