package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/app"
	"github.com/Agamariel/go-metrics/pkg/buildinfo"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// appRunner — минимальный интерфейс, необходимый для запуска и остановки сервера.
// Позволяет подменять реализацию в тестах.
type appRunner interface {
	Run() error
	Shutdown(ctx context.Context) error
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run содержит основную логику приложения и возвращает ошибку вместо вызова os.Exit
func run() error {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	application, err := app.NewApplication()
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	return runApp(application)
}

// runApp регистрирует обработчики сигналов и передаёт управление в runWithStop.
func runApp(application appRunner) error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer signal.Stop(stop)

	return runWithStop(application, stop)
}

// runWithStop содержит логику ожидания сигнала и graceful shutdown.
// Принимает канал сигналов явно — это позволяет тестам управлять
// жизненным циклом напрямую без использования системных сигналов.
func runWithStop(application appRunner, stop <-chan os.Signal) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- application.Run()
	}()

	select {
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := application.Shutdown(ctx); err != nil {
			return fmt.Errorf("ошибка при остановке: %w", err)
		}
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("ошибка запуска: %w", err)
		}
	}

	return nil
}
