package app

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

// Run запускает HTTP сервер.
func (a *App) Run() error {
	a.logger.Info("Сервер запущен", zap.String("address", a.config.Address))

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
