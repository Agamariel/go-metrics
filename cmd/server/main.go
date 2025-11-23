package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/config/db"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/logger"
	custommiddleware "github.com/Agamariel/go-metrics/internal/middleware"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем zap логгер
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer zapLogger.Sync()

	// Создаем адаптер для нашего интерфейса Logger
	log := logger.NewZapAdapter(zapLogger)

	// Загружаем конфигурацию
	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatal("Ошибка при загрузке конфигурации", zap.Error(err))
	}

	// Инициализируем хранилище
	var storage repository.Storage
	var database *sql.DB

	// PostgreSQL -> File -> Memory
	if cfg.DatabaseDSN != "" {
		database, err = db.New(context.Background(), db.NewPostgreSQL(cfg.DatabaseDSN), log)
		if err != nil {
			log.Fatal("Ошибка при подключении к базе данных", zap.Error(err))
		}
		// Применяем миграции
		if err := repository.Migrate(context.Background(), database, log); err != nil {
			database.Close()
			log.Fatal("Ошибка при применении миграций", zap.Error(err))
		}
		storage = repository.NewPostgresStorage(database, log)
		log.Info("Используется хранилище PostgreSQL")
	} else if cfg.FileStoragePath != "" {
		fileStorage, err := repository.NewFileStorage(repository.FileStorageConfig{
			FilePath:      cfg.FileStoragePath,
			StoreInterval: cfg.StoreInterval,
			Restore:       cfg.Restore,
			Logger:        log,
		})
		if err != nil {
			log.Fatal("Ошибка при инициализации файлового хранилища", zap.Error(err))
		}
		storage = fileStorage
		log.Info("Используется файловое хранилище", zap.String("path", cfg.FileStoragePath))
	} else {
		storage = repository.NewMemStorage()
		log.Info("Используется in-memory хранилище")
	}

	// Создаём сервис с бизнес-логикой
	metricsService := service.NewMetricsService(storage)

	h := handler.NewMetricsHandler(metricsService)

	// Создаём ping handler для проверки БД
	dbHandler := handler.NewDBHandler(database, log)

	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(custommiddleware.GzipMiddleware) // Сжатие gzip для всех эндпоинтов через нашу middleware
	// У роутера есть встроенная middleware для сжатия ответов gzip
	//r.Use(middleware.Compress(1)) // уровень сжатия 1, сжимаются типы из дефолтного списка
	r.Use(custommiddleware.HashMiddleware(cfg.Key)) // Проверка и добавление хеша
	r.Use(custommiddleware.Logger(log))             // Кастомное логирование через интерфейс
	r.Use(middleware.Recoverer)                     // Восстановление после паники
	r.Use(middleware.RequestID)                     // Добавление request ID
	r.Use(middleware.RealIP)                        // Определение реального IP клиента

	// Настраиваем маршруты с использованием chi
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	// JSON API эндпоинты
	r.Post("/update/", h.UpdateMetricJSONHandler)
	r.Post("/updates/", h.UpdateMetricsBatchHandler)
	r.Post("/value/", h.GetMetricJSONHandler)

	// Проверка соединения с БД
	r.Get("/ping", dbHandler.PingDB)

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
		log.Info("Сервер запущен", zap.String("address", cfg.Address))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Ошибка сервера", zap.Error(err))
		}
	}()

	// Ожидаем сигнал остановки
	<-stop
	log.Info("Получен сигнал остановки, завершаем работу...")

	// Graceful shutdown с настраиваемым таймаутом
	shutdownTimeout := time.Duration(cfg.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Info("Начинаем graceful shutdown", zap.Int("timeout_sec", cfg.ShutdownTimeout))
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Ошибка при остановке сервера", zap.Error(err))
	}

	// Закрываем хранилище (сохраняет метрики)
	if err := storage.Close(); err != nil {
		log.Error("Ошибка при закрытии хранилища", zap.Error(err))
	}

	// Закрываем подключение к базе данных
	if database != nil {
		if err := database.Close(); err != nil {
			log.Error("Ошибка при закрытии подключения к БД", zap.Error(err))
		}
	}

	log.Info("Сервер остановлен")
}
