package app

import (
	"context"
	"errors"
	"net"
	"net/http"

	"go.uber.org/zap"
)

// Run запускает HTTP и (при наличии конфигурации) gRPC сервер.
func (a *App) Run() error {
	a.logger.Info("HTTP-сервер запущен", zap.String("address", a.config.Address))

	if a.grpcServer != nil {
		go func() {
			lis, err := net.Listen("tcp", a.config.GRPCAddress)
			if err != nil {
				a.logger.Error("Ошибка запуска gRPC listener", zap.Error(err))
				return
			}
			a.logger.Info("gRPC-сервер запущен", zap.String("address", a.config.GRPCAddress))
			if err := a.grpcServer.Serve(lis); err != nil {
				a.logger.Error("Ошибка gRPC-сервера", zap.Error(err))
			}
		}()
	}

	err := a.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown выполняет graceful shutdown сервера и всех компонентов приложения.
// Порядок остановки:
// 1. HTTP сервер (прекращает прием новых запросов)
// 2. Audit publisher (дожидается завершения операций аудита)
// 3. Storage (сохраняет данные)
// 4. Database (закрывает соединения)
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("Получен сигнал остановки, завершаем работу...")

	// Останавливаем gRPC-сервер
	if a.grpcServer != nil {
		a.logger.Info("Останавливаем gRPC-сервер")
		a.grpcServer.GracefulStop()
	}

	// Останавливаем HTTP сервер
	a.logger.Info("Начинаем graceful shutdown")
	if err := a.server.Shutdown(ctx); err != nil {
		a.logger.Error("Ошибка при остановке сервера", zap.Error(err))
		return err
	}

	// Закрываем Publisher аудита (ожидаем завершения всех операций)
	if a.publisher != nil {
		a.logger.Info("Ожидаем завершения операций аудита...")
		a.publisher.Close()
	}

	// Закрываем хранилище (сохраняет метрики)
	if err := a.storage.Close(); err != nil {
		a.logger.Error("Ошибка при закрытии хранилища", zap.Error(err))
		return err
	}

	// Закрываем подключение к базе данных
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			a.logger.Error("Ошибка при закрытии подключения к БД", zap.Error(err))
			return err
		}
	}

	a.logger.Info("Сервер остановлен")
	return nil
}
