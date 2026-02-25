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

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run содержит основную логику приложения и возвращает ошибку вместо вызова os.Exit
func run() error {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	// Создаем и инициализируем приложение
	application, err := app.NewApplication()
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	// Канал для сигналов остановки
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Запускаем приложение в отдельной горутине
	errChan := make(chan error, 1)
	go func() {
		errChan <- application.Run()
	}()

	// Ожидаем сигнал остановки или ошибку запуска
	select {
	case <-stop:
		// Graceful shutdown с таймаутом 30 секунд
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
