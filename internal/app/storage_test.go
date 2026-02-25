package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Agamariel/go-metrics/internal/config"
)

func TestApp_InitStorage_Memory(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Конфигурация без DatabaseDSN и FileStoragePath
	app.config = &config.ServerConfig{}

	ctx := context.Background()
	if err := app.initStorage(ctx); err != nil {
		t.Fatalf("initStorage failed: %v", err)
	}

	if app.storage == nil {
		t.Error("Storage should not be nil")
	}

	// Закрываем storage
	app.storage.Close()
}

func TestApp_InitStorage_File(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	tmpDir := t.TempDir()
	storageFile := filepath.Join(tmpDir, "metrics.json")

	// Конфигурация с FileStoragePath
	app.config = &config.ServerConfig{
		FileStoragePath: storageFile,
		StoreInterval:   300,
		Restore:         true,
	}

	ctx := context.Background()
	if err := app.initStorage(ctx); err != nil {
		t.Fatalf("initStorage failed: %v", err)
	}

	if app.storage == nil {
		t.Error("Storage should not be nil")
	}

	// Закрываем storage
	app.storage.Close()
}

func TestApp_InitStorage_Postgres_Invalid(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Невалидный DSN
	app.config = &config.ServerConfig{
		DatabaseDSN: "invalid-dsn",
	}

	ctx := context.Background()
	err := app.initStorage(ctx)
	if err == nil {
		t.Error("Expected error with invalid database DSN")
	}
}

func TestApp_InitFileStorage_WithRestore(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	tmpDir := t.TempDir()
	storageFile := filepath.Join(tmpDir, "metrics-restore.json")

	// Конфигурация с восстановлением из файла
	app.config = &config.ServerConfig{
		FileStoragePath: storageFile,
		StoreInterval:   0, // Синхронная запись
		Restore:         true,
	}

	err := app.initFileStorage()
	if err != nil {
		t.Fatalf("initFileStorage with restore failed: %v", err)
	}

	if app.storage == nil {
		t.Error("Storage should not be nil")
	}

	// Закрываем storage
	app.storage.Close()
}
