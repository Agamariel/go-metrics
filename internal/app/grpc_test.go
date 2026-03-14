package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	grpcserver "github.com/Agamariel/go-metrics/internal/grpc"
	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/internal/handler"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
	"google.golang.org/grpc"
)

// freePort возвращает свободный локальный адрес.
func freePort(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	lis.Close()
	return addr
}

func TestApp_Run_WithGRPC(t *testing.T) {
	grpcAddr := freePort(t)
	httpAddr := freePort(t)

	app := &App{}
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger: %v", err)
	}

	app.config = &config.ServerConfig{
		Address:     httpAddr,
		GRPCAddress: grpcAddr,
	}
	app.storage = repository.NewMemStorage()

	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, nil)
	dbHandler := handler.NewDBHandler(nil, app.logger)
	router := app.setupRouter(metricsHandler, dbHandler)

	app.server = &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	interceptor := grpcserver.TrustedSubnetInterceptor("")
	app.grpcServer = grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	grpcserver.RegisterMetricsServer(app.grpcServer, metricsService)

	go func() { _ = app.Run() }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown with gRPC failed: %v", err)
	}
}

func TestApp_Shutdown_WithGRPC_NilGRPCServer(t *testing.T) {
	httpAddr := freePort(t)

	app := &App{}
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger: %v", err)
	}

	app.config = &config.ServerConfig{Address: httpAddr}
	app.storage = repository.NewMemStorage()
	app.grpcServer = nil

	metricsService := service.NewMetricsService(app.storage)
	metricsHandler := handler.NewMetricsHandler(metricsService, nil)
	dbHandler := handler.NewDBHandler(nil, app.logger)
	router := app.setupRouter(metricsHandler, dbHandler)

	app.server = &http.Server{
		Addr:    httpAddr,
		Handler: router,
	}

	go func() { _ = app.Run() }()
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown without gRPC failed: %v", err)
	}
}
