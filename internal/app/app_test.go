package app

import (
	"os"
	"testing"

	"github.com/Agamariel/go-metrics/internal/config"
)

func TestApp_InitLogger(t *testing.T) {
	app := &App{}

	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	if app.logger == nil {
		t.Error("Logger should not be nil after initialization")
	}
}

func TestApp_InitMemStorage(t *testing.T) {
	app := &App{}

	// Инициализируем логгер (требуется для storage)
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	if err := app.initMemStorage(); err != nil {
		t.Fatalf("initMemStorage failed: %v", err)
	}

	if app.storage == nil {
		t.Error("Storage should not be nil after initialization")
	}
}

func TestApp_InitAudit_Disabled(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Устанавливаем конфигурацию без параметров аудита
	app.config = &config.ServerConfig{
		AuditFile: "",
		AuditURL:  "",
	}

	// Аудит должен остаться nil, если не указаны параметры
	if err := app.initAudit(); err != nil {
		t.Fatalf("initAudit failed: %v", err)
	}

	// Publisher должен быть nil, так как аудит отключен
	if app.publisher != nil {
		t.Error("Publisher should be nil when audit is disabled")
	}
}

func TestApp_InitCrypto_EmptyPath(t *testing.T) {
	app := &App{}
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger: %v", err)
	}
	app.config = &config.ServerConfig{CryptoKey: ""}

	if err := app.initCrypto(); err != nil {
		t.Errorf("initCrypto with empty path should not fail: %v", err)
	}
	if app.privateKey != nil {
		t.Error("privateKey should be nil when CryptoKey is empty")
	}
}

func TestApp_InitCrypto_InvalidPath(t *testing.T) {
	app := &App{}
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger: %v", err)
	}
	app.config = &config.ServerConfig{CryptoKey: "/nonexistent/key.pem"}

	if err := app.initCrypto(); err == nil {
		t.Error("initCrypto with invalid path should fail")
	}
}

func TestApp_InitCrypto_InvalidKeyContent(t *testing.T) {
	f, err := os.CreateTemp("", "bad-key-*.pem")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("this is not a valid PEM key")
	f.Close()

	app := &App{}
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger: %v", err)
	}
	app.config = &config.ServerConfig{CryptoKey: f.Name()}

	if err := app.initCrypto(); err == nil {
		t.Error("initCrypto with invalid key file content should fail")
	}
}

func TestApp_InitFileStorage(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Устанавливаем конфигурацию для файлового хранилища
	app.config = &config.ServerConfig{
		FileStoragePath: "test-metrics.json",
		StoreInterval:   300,
		Restore:         true,
	}

	if err := app.initFileStorage(); err != nil {
		t.Fatalf("initFileStorage failed: %v", err)
	}

	if app.storage == nil {
		t.Error("Storage should not be nil after initialization")
	}

	// Закрываем storage
	app.storage.Close()
}
