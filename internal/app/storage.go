package app

import (
	"context"

	"github.com/Agamariel/go-metrics/internal/config/db"
	"github.com/Agamariel/go-metrics/internal/repository"
	"go.uber.org/zap"
)

// initStorage инициализирует хранилище метрик.
// Приоритет: PostgreSQL -> File -> Memory
func (a *App) initStorage(ctx context.Context) error {
	if a.config.DatabaseDSN != "" {
		return a.initPostgresStorage(ctx)
	} else if a.config.FileStoragePath != "" {
		return a.initFileStorage()
	}
	return a.initMemStorage()
}

// initPostgresStorage инициализирует PostgreSQL хранилище
func (a *App) initPostgresStorage(ctx context.Context) error {
	database, err := db.New(ctx, db.NewPostgreSQL(a.config.DatabaseDSN), a.logger)
	if err != nil {
		return err
	}

	// Применяем миграции
	if err := repository.Migrate(ctx, database, a.logger); err != nil {
		database.Close()
		return err
	}

	a.database = database
	a.storage = repository.NewPostgresStorage(database, a.logger)
	a.logger.Info("Используется хранилище PostgreSQL")

	return nil
}

// initFileStorage инициализирует файловое хранилище
func (a *App) initFileStorage() error {
	fileStorage, err := repository.NewFileStorage(repository.FileStorageConfig{
		FilePath:      a.config.FileStoragePath,
		StoreInterval: a.config.StoreInterval,
		Restore:       a.config.Restore,
		Logger:        a.logger,
	})
	if err != nil {
		return err
	}

	a.storage = fileStorage
	a.logger.Info("Используется файловое хранилище", zap.String("path", a.config.FileStoragePath))

	return nil
}

// initMemStorage инициализирует in-memory хранилище
func (a *App) initMemStorage() error {
	a.storage = repository.NewMemStorage()
	a.logger.Info("Используется in-memory хранилище")
	return nil
}
