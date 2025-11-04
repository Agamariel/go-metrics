package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/handler"
	custommiddleware "github.com/Agamariel/go-metrics/internal/middleware"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем zap логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Загружаем конфигурацию
	cfg, err := config.LoadServerConfig()
	if err != nil {
		logger.Fatal("Ошибка при загрузке конфигурации", zap.Error(err))
	}

	// Инициализируем хранилище с файловой персистентностью
	storage, err := repository.NewFileStorage(repository.FileStorageConfig{
		FilePath:      cfg.FileStoragePath,
		StoreInterval: cfg.StoreInterval,
		Restore:       cfg.Restore,
		Logger:        logger,
	})
	if err != nil {
		logger.Fatal("Ошибка при инициализации хранилища", zap.Error(err))
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
		logger.Info("Сервер запущен", zap.String("address", cfg.Address))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Ошибка сервера", zap.Error(err))
		}
	}()

	// Ожидаем сигнал остановки
	<-stop
	logger.Info("Получен сигнал остановки, завершаем работу...")

	// Graceful shutdown с настраиваемым таймаутом
	shutdownTimeout := time.Duration(cfg.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	logger.Info("Начинаем graceful shutdown", zap.Int("timeout_sec", cfg.ShutdownTimeout))
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Ошибка при остановке сервера", zap.Error(err))
	}

	// Закрываем хранилище (сохраняет метрики)
	if err := storage.Close(); err != nil {
		logger.Error("Ошибка при закрытии хранилища", zap.Error(err))
	}

	logger.Info("Сервер остановлен")
}
