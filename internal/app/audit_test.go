package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Agamariel/go-metrics/internal/config"
)

func TestApp_InitAudit_FileObserver(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Создаём временный файл для аудита
	tmpDir := t.TempDir()
	auditFile := filepath.Join(tmpDir, "audit.log")

	app.config = &config.ServerConfig{
		AuditFile: auditFile,
	}

	if err := app.initAudit(); err != nil {
		t.Fatalf("initAudit failed: %v", err)
	}

	if app.publisher == nil {
		t.Error("Publisher should not be nil when file audit is enabled")
	}

	// Проверяем, что файл был создан
	if _, err := os.Stat(auditFile); os.IsNotExist(err) {
		t.Error("Audit file was not created")
	}

	// Закрываем publisher
	if app.publisher != nil {
		app.publisher.Close()
	}
}

func TestApp_InitAudit_HTTPObserver(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	app.config = &config.ServerConfig{
		AuditURL: "http://localhost:9999/audit",
	}

	if err := app.initAudit(); err != nil {
		t.Fatalf("initAudit failed: %v", err)
	}

	if app.publisher == nil {
		t.Error("Publisher should not be nil when HTTP audit is enabled")
	}

	// Закрываем publisher
	if app.publisher != nil {
		app.publisher.Close()
	}
}

func TestApp_InitAudit_BothObservers(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Создаём временный файл для аудита
	tmpDir := t.TempDir()
	auditFile := filepath.Join(tmpDir, "audit.log")

	app.config = &config.ServerConfig{
		AuditFile: auditFile,
		AuditURL:  "http://localhost:9999/audit",
	}

	if err := app.initAudit(); err != nil {
		t.Fatalf("initAudit failed: %v", err)
	}

	if app.publisher == nil {
		t.Error("Publisher should not be nil when both audits are enabled")
	}

	// Закрываем publisher
	if app.publisher != nil {
		app.publisher.Close()
	}
}

func TestApp_InitAudit_InvalidFile(t *testing.T) {
	app := &App{}

	// Инициализируем логгер
	if err := app.initLogger(); err != nil {
		t.Fatalf("initLogger failed: %v", err)
	}

	// Используем недопустимый путь
	app.config = &config.ServerConfig{
		AuditFile: "/invalid/path/that/does/not/exist/audit.log",
	}

	err := app.initAudit()
	if err == nil {
		t.Error("Expected error with invalid audit file path")
	}
}
