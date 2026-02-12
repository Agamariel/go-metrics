package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/app"
)

func main() {
	// Создаем и инициализируем приложение
	application, err := app.NewApplication()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации: %v\n", err)
		os.Exit(1)
	}

	// Канал для сигналов остановки
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

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
			fmt.Fprintf(os.Stderr, "Ошибка при остановке: %v\n", err)
			os.Exit(1)
		}
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
			os.Exit(1)
		}
	}
}
