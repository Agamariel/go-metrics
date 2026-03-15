// Package app содержит логику инициализации и запуска приложения.
package app

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/audit"
	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/grpcserver"
	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/Agamariel/go-metrics/pkg/crypto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// App представляет собой приложение сервера метрик.
type App struct {
	config     *config.ServerConfig
	logger     logger.Logger
	storage    repository.Storage
	database   *sql.DB
	publisher  *audit.Publisher
	server     *http.Server
	grpcServer *grpc.Server
	privateKey *rsa.PrivateKey
}

// NewApplication создает и полностью инициализирует приложение.
func NewApplication() (*App, error) {
	app := &App{}

	// Инициализируем логгер (первым, т.к. нужен для всех остальных компонентов)
	if err := app.initLogger(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации логгера: %w", err)
	}

	// Загружаем конфигурацию
	if err := app.initConfig(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Загружаем приватный ключ (если задан)
	if err := app.initCrypto(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации шифрования: %w", err)
	}

	// Инициализируем хранилище (PostgreSQL -> File -> Memory)
	if err := app.initStorage(context.Background()); err != nil {
		return nil, fmt.Errorf("ошибка инициализации хранилища: %w", err)
	}

	// Инициализируем систему аудита (опционально)
	if err := app.initAudit(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации аудита: %w", err)
	}

	// Создаем сервис и handlers
	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, app.publisher)
	dbHandler := handler.NewDBHandler(app.database, app.logger)

	// Настраиваем роутер
	router := app.setupRouter(metricsHandler, dbHandler)

	// Создаем HTTP сервер
	app.server = &http.Server{
		Addr:    app.config.Address,
		Handler: router,
	}

	// Создаем gRPC-сервер (если задан адрес)
	if app.config.GRPCAddress != "" {
		interceptor := grpcserver.TrustedSubnetInterceptor(app.config.TrustedSubnet)
		app.grpcServer = grpc.NewServer(grpc.UnaryInterceptor(interceptor))
		grpcserver.RegisterMetricsServer(app.grpcServer, metricsService)
	}

	app.logger.Info("Приложение успешно инициализировано",
		zap.String("address", app.config.Address),
		zap.String("grpc_address", app.config.GRPCAddress),
		zap.Bool("crypto_enabled", app.privateKey != nil),
	)

	return app, nil
}

// initCrypto загружает приватный RSA ключ из файла, если путь задан в конфигурации.
func (a *App) initCrypto() error {
	if a.config.CryptoKey == "" {
		return nil
	}

	key, err := crypto.LoadPrivateKey(a.config.CryptoKey)
	if err != nil {
		return err
	}

	a.privateKey = key
	return nil
}

// initLogger инициализирует zap логгер
func (a *App) initLogger() error {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}

	a.logger = logger.NewZapAdapter(zapLogger)
	return nil
}

// initConfig загружает конфигурацию из флагов и переменных окружения
func (a *App) initConfig() error {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		return err
	}

	a.config = &cfg
	return nil
}
