package repository

import (
	"os"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/models"
	"go.uber.org/zap"
)

// mockTicker реализует интерфейс Ticker для тестирования
type mockTicker struct {
	ch       chan time.Time
	stopChan chan struct{}
}

func newMockTicker() *mockTicker {
	return &mockTicker{
		ch:       make(chan time.Time, 1),
		stopChan: make(chan struct{}),
	}
}

func (m *mockTicker) C() <-chan time.Time {
	return m.ch
}

func (m *mockTicker) Stop() {
	close(m.stopChan)
}

func (m *mockTicker) Tick() {
	m.ch <- time.Now()
}

func TestFileStorageWithMockTicker(t *testing.T) {
	// Создаем временный файл для тестов
	tmpFile, err := os.CreateTemp("", "test-metrics-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger, _ := zap.NewDevelopment()
	mockTick := newMockTicker()

	// Создаем FileStorage
	fs, err := NewFileStorage(FileStorageConfig{
		FilePath:      tmpFile.Name(),
		StoreInterval: 1,
		Restore:       false,
		Logger:        logger,
		Ticker:        mockTick,
	})
	if err != nil {
		t.Fatalf("Не удалось создать FileStorage: %v", err)
	}
	defer fs.Close()

	// Добавляем метрику
	val := 42.0 // Автостопом по галактике :)
	metric := models.Metrics{
		ID:    "test_metric",
		MType: models.Gauge,
		Value: &val,
	}

	if err := fs.UpdateMetric(metric); err != nil {
		t.Fatalf("Не удалось обновить метрику: %v", err)
	}

	// Проверяем, что сохранение еще не произошло (счетчик = 0, т.к. интервал > 0)
	if count := fs.GetSaveCount(); count != 0 {
		t.Errorf("Ожидали 0 сохранений, получили %d", count)
	}

	// Вручную триггерим тик
	mockTick.Tick()

	// Даем время горутине обработать событие
	time.Sleep(10 * time.Millisecond)

	// Проверяем, что сохранение произошло
	if count := fs.GetSaveCount(); count != 1 {
		t.Errorf("Ожидали 1 сохранение после тика, получили %d", count)
	}

	// Триггерим еще один тик
	mockTick.Tick()
	time.Sleep(10 * time.Millisecond)

	// Проверяем, что счетчик увеличился
	if count := fs.GetSaveCount(); count != 2 {
		t.Errorf("Ожидали 2 сохранения после второго тика, получили %d", count)
	}
}

// TestFileStorageSyncMode проверяет синхронный режим (STORE_INTERVAL=0)
func TestFileStorageSyncMode(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-metrics-*.json")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger, _ := zap.NewDevelopment()

	// В синхронном режиме ticker не используется
	fs, err := NewFileStorage(FileStorageConfig{
		FilePath:      tmpFile.Name(),
		StoreInterval: 0, // Синхронный режим
		Restore:       false,
		Logger:        logger,
	})
	if err != nil {
		t.Fatalf("Не удалось создать FileStorage: %v", err)
	}
	defer fs.Close()

	// Добавляем метрику
	val := 42.0 // Автостопом по галактике :)
	metric := models.Metrics{
		ID:    "test_metric",
		MType: models.Gauge,
		Value: &val,
	}

	if err := fs.UpdateMetric(metric); err != nil {
		t.Fatalf("Не удалось обновить метрику: %v", err)
	}

	// В синхронном режиме сохранение должно произойти сразу
	if count := fs.GetSaveCount(); count != 1 {
		t.Errorf("Ожидали 1 сохранение в синхронном режиме, получили %d", count)
	}
}
