package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	ctx := context.Background()
	if err := fs.UpdateMetric(ctx, metric); err != nil {
		t.Fatalf("Не удалось обновить метрику: %v", err)
	}

	// Проверяем, что сохранение еще не произошло (счетчик = 0, т.к. интервал > 0)
	if count := fs.GetSaveCount(); count != 0 {
		t.Errorf("Ожидали 0 сохранений, получили %d", count)
	}

	// Вручную триггерим тик и ждем подтверждения сохранения
	mockTick.Tick()
	waitForSaveCount(t, fs, 1, time.Second)

	// Триггерим еще один тик
	mockTick.Tick()
	waitForSaveCount(t, fs, 2, time.Second)
}

// waitForSaveCount ожидает, пока счётчик сохранений не достигнет expected,
// опрашивая с шагом 5ms. Используется вместо time.Sleep для стабильности теста.
func waitForSaveCount(t *testing.T, fs *FileStorage, expected int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fs.GetSaveCount() == expected {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("Ожидали %d сохранений, получили %d (таймаут %s)", expected, fs.GetSaveCount(), timeout)
}

// TestFileStorageGetMetric проверяет GetMetric через FileStorage
func TestFileStorageGetMetric(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-metrics-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger, _ := zap.NewDevelopment()
	fs, err := NewFileStorage(FileStorageConfig{
		FilePath:      tmpFile.Name(),
		StoreInterval: 0,
		Logger:        logger,
	})
	require.NoError(t, err)
	defer fs.Close()

	ctx := context.Background()
	val := 55.5
	require.NoError(t, fs.UpdateMetric(ctx, models.Metrics{
		ID: "cpu", MType: models.Gauge, Value: &val,
	}))

	m, err := fs.GetMetric(ctx, "cpu", models.Gauge)
	require.NoError(t, err)
	require.NotNil(t, m.Value)
	assert.Equal(t, 55.5, *m.Value)
}

// TestFileStorageGetAllMetrics проверяет GetAllMetrics через FileStorage
func TestFileStorageGetAllMetrics(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-metrics-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger, _ := zap.NewDevelopment()
	fs, err := NewFileStorage(FileStorageConfig{
		FilePath:      tmpFile.Name(),
		StoreInterval: 0,
		Logger:        logger,
	})
	require.NoError(t, err)
	defer fs.Close()

	ctx := context.Background()
	v1, d1 := 1.1, int64(10)
	require.NoError(t, fs.UpdateMetric(ctx, models.Metrics{ID: "g1", MType: models.Gauge, Value: &v1}))
	require.NoError(t, fs.UpdateMetric(ctx, models.Metrics{ID: "c1", MType: models.Counter, Delta: &d1}))

	all, err := fs.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

// TestFileStorageUpdateMetrics проверяет пакетное обновление
func TestFileStorageUpdateMetrics(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-metrics-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	logger, _ := zap.NewDevelopment()
	fs, err := NewFileStorage(FileStorageConfig{
		FilePath:      tmpFile.Name(),
		StoreInterval: 0,
		Logger:        logger,
	})
	require.NoError(t, err)
	defer fs.Close()

	ctx := context.Background()
	v1, v2 := 10.0, 20.0
	metrics := []models.Metrics{
		{ID: "m1", MType: models.Gauge, Value: &v1},
		{ID: "m2", MType: models.Gauge, Value: &v2},
	}
	require.NoError(t, fs.UpdateMetrics(ctx, metrics))

	all, err := fs.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	// UpdateMetrics делает одно пакетное сохранение
	assert.Equal(t, int64(1), fs.GetSaveCount())
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

	ctx := context.Background()
	if err := fs.UpdateMetric(ctx, metric); err != nil {
		t.Fatalf("Не удалось обновить метрику: %v", err)
	}

	// В синхронном режиме сохранение должно произойти сразу
	if count := fs.GetSaveCount(); count != 1 {
		t.Errorf("Ожидали 1 сохранение в синхронном режиме, получили %d", count)
	}
}
