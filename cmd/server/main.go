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

	// Инициализируем подключение к базе данных (если указан DSN)
	var database *db.DB
	if cfg.DatabaseDSN != "" {
		database, err = db.New(context.Background(), db.Config{
			DSN: cfg.DatabaseDSN,
		}, log)
		if err != nil {
			log.Fatal("Ошибка при подключении к базе данных", zap.Error(err))
		}
		log.Info("База данных подключена")
	} else {
		log.Info("База данных не настроена (DSN не указан)")
	}

	// Инициализируем хранилище с файловой персистентностью
	storage, err := repository.NewFileStorage(repository.FileStorageConfig{
		FilePath:      cfg.FileStoragePath,
		StoreInterval: cfg.StoreInterval,
		Restore:       cfg.Restore,
		Logger:        log,
	})
	if err != nil {
		log.Fatal("Ошибка при инициализации хранилища", zap.Error(err))
	}

	// Создаём сервис с бизнес-логикой
	metricsService := service.NewMetricsService(storage)

	h := handler.NewMetricsHandler(metricsService)

	// Создаём ping handler для проверки БД
	dbHandler := handler.NewDbHandler(database, log)

	r := chi.NewRouter()

	// Добавляем middleware
	r.Use(custommiddleware.GzipMiddleware) // Сжатие gzip для всех эндпоинтов через нашу middleware
	// У роутера есть встроенная middleware для сжатия ответов gzip
	//r.Use(middleware.Compress(1)) // уровень сжатия 1, сжимаются типы из дефолтного списка
	r.Use(custommiddleware.Logger(log)) // Кастомное логирование через интерфейс
	r.Use(middleware.Recoverer)         // Восстановление после паники
	r.Use(middleware.RequestID)         // Добавление request ID
	r.Use(middleware.RealIP)            // Определение реального IP клиента

	// Настраиваем маршруты с использованием chi
	r.Post("/update/{type}/{name}/{value}", h.UpdateMetricHandler)
	r.Get("/value/{type}/{name}", h.GetMetricHandler)

	// JSON API эндпоинты
	r.Post("/update/", h.UpdateMetricJSONHandler)
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

	// Закрываем подключение к базе
	if database != nil {
		if err := database.Close(log); err != nil {
			log.Error("Ошибка при закрытии подключения к БД", zap.Error(err))
		}
	}

	// Закрываем хранилище (сохраняет метрики)
	if err := storage.Close(); err != nil {
		log.Error("Ошибка при закрытии хранилища", zap.Error(err))
	}

	log.Info("Сервер остановлен")
}
