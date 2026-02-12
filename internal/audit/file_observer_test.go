package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileObserver(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	observer, err := NewFileObserver(filePath)
	if err != nil {
		t.Fatalf("Failed to create FileObserver: %v", err)
	}

	if observer.filePath != filePath {
		t.Errorf("Expected filePath %s, got %s", filePath, observer.filePath)
	}

	// Проверяем, что файл создан
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File should be created")
	}
}

func TestNewFileObserver_EmptyPath(t *testing.T) {
	_, err := NewFileObserver("")
	if err == nil {
		t.Error("Expected error for empty filePath")
	}
}

func TestFileObserver_Notify(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	observer, err := NewFileObserver(filePath)
	if err != nil {
		t.Fatalf("Failed to create FileObserver: %v", err)
	}

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"Alloc", "TotalAlloc"},
		IPAddress: "192.168.0.42",
	}

	ctx := context.Background()
	if err := observer.Notify(ctx, event); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	// Читаем файл и проверяем содержимое
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	var readEvent Event
	if err := json.Unmarshal([]byte(lines[0]), &readEvent); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if readEvent.IPAddress != event.IPAddress {
		t.Errorf("Expected IP %s, got %s", event.IPAddress, readEvent.IPAddress)
	}

	if len(readEvent.Metrics) != len(event.Metrics) {
		t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(readEvent.Metrics))
	}
}

func TestFileObserver_Notify_MultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	observer, err := NewFileObserver(filePath)
	if err != nil {
		t.Fatalf("Failed to create FileObserver: %v", err)
	}

	ctx := context.Background()

	// Отправляем несколько событий
	for i := 0; i < 3; i++ {
		event := Event{
			Timestamp: time.Now().Unix(),
			Metrics:   []string{"metric" + string(rune('1'+i))},
			IPAddress: "192.168.0." + string(rune('1'+i)),
		}

		if err := observer.Notify(ctx, event); err != nil {
			t.Fatalf("Notify %d failed: %v", i, err)
		}
	}

	// Читаем файл и проверяем количество строк
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Проверяем, что каждая строка - валидный JSON
	for i, line := range lines {
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestFileObserver_Notify_Append(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	// Создаем файл с предварительным содержимым
	initialContent := `{"ts":1234567890,"metrics":["initial"],"ip_address":"192.168.0.1"}` + "\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	observer, err := NewFileObserver(filePath)
	if err != nil {
		t.Fatalf("Failed to create FileObserver: %v", err)
	}

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"new_metric"},
		IPAddress: "192.168.0.2",
	}

	ctx := context.Background()
	if err := observer.Notify(ctx, event); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	// Читаем файл и проверяем, что старое содержимое сохранилось
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (initial + new), got %d", len(lines))
	}

	// Проверяем первую строку (должна остаться неизменной)
	if !strings.Contains(lines[0], "initial") {
		t.Error("Initial content should be preserved")
	}

	// Проверяем вторую строку (новое событие)
	if !strings.Contains(lines[1], "new_metric") {
		t.Error("New event should be appended")
	}
}
