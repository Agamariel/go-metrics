package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/handler"
	custommiddleware "github.com/Agamariel/go-metrics/internal/middleware"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Config содержит конфигурацию сервера
type Config struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

func main() {
	// Значения по умолчанию
	cfg := Config{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "metrics-db.json",
		Restore:         true,
	}

	// Определяем флаги командной строки
	serverAddress := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	storeInterval := flag.Int("i", cfg.StoreInterval, "интервал сохранения метрик в секундах (0 = синхронное сохранение)")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "путь к файлу для сохранения метрик")
	restore := flag.Bool("r", cfg.Restore, "загружать ли ранее сохранённые метрики при старте")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		log.Fatalf("Ошибка: неизвестные аргументы: %v", flag.Args())
	}

	// Применяем значения из флагов
	cfg.Address = *serverAddress
	cfg.StoreInterval = *storeInterval
	cfg.FileStoragePath = *fileStoragePath
	cfg.Restore = *restore

	// Парсим переменные окружения (приоритет выше флагов)
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Ошибка при парсинге переменных окружения: %v", err)
	}

	// Инициализируем zap логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Ошибка при инициализации логгера: %v", err)
	}
	defer logger.Sync()

	// Инициализируем хранилище с файловой персистентностью
	storage, err := repository.NewFileStorage(repository.FileStorageConfig{
		FilePath:      cfg.FileStoragePath,
		StoreInterval: cfg.StoreInterval,
		Restore:       cfg.Restore,
		Logger:        logger,
	})
	if err != nil {
		log.Fatalf("Ошибка при инициализации хранилища: %v", err)
	}

	// Создаём сервис с бизнес-логикой
	metricsService := service.NewMetricsService(storage)

	h := handler.NewMetricsHandler(metricsService)
	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(custommiddleware.GzipMiddleware) // Сжатие gzip для всех эндпоинтов через нашу middleware
	// У роутера есть встроенная middleware для сжатия ответов gzip
	//r.Use(middleware.Compress(1)) // уровень сжатия 1, сжимаются типы из дефолтного списка
	r.Use(custommiddleware.Logger(logger)) // Кастомное логирование с zap
	r.Use(middleware.Recoverer)            // Восстановление после паники
	r.Use(middleware.RequestID)            // Добавление request ID
	r.Use(middleware.RealIP)               // Определение реального IP клиента

	// Настраиваем маршруты с использованием chi
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	// JSON API эндпоинты
	r.Post("/update/", h.UpdateMetricJSONHandler)
	r.Post("/value/", h.GetMetricJSONHandler)

	// Список всех метрик
	r.Get("/", h.ListMetricsHandler)

	// Настройка graceful shutdown
	server := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}

	// Канал для получения сигналов остановки
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Server started at http://%s", cfg.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидаем сигнал остановки
	<-stop
	logger.Info("Получен сигнал остановки, завершаем работу...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при остановке сервера", zap.Error(err))
	}

	// Закрываем хранилище (сохраняет метрики)
	if err := storage.Close(); err != nil {
		logger.Error("Ошибка при закрытии хранилища", zap.Error(err))
	}

	logger.Info("Сервер остановлен")
}
