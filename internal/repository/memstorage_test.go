package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

func TestNewMemStorage(t *testing.T) {
	storage := NewMemStorage()

	if storage == nil {
		t.Fatal("NewMemStorage returned nil")
	}

	if storage.gauges == nil {
		t.Error("gauges map is nil")
	}

	if storage.counters == nil {
		t.Error("counters map is nil")
	}
}

func TestUpdateMetricGauge(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	value := 123.456
	metric := models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
		Value: &value,
	}

	err := storage.UpdateMetric(ctx, metric)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что метрика сохранена
	result, err := storage.GetMetric(ctx, "TestGauge", models.Gauge)
	if err != nil {
		t.Fatal("Metric not found after update")
	}

	if result.Value == nil {
		t.Fatal("Metric value is nil")
	}

	if *result.Value != value {
		t.Errorf("Expected value %f, got %f", value, *result.Value)
	}
}

func TestUpdateMetricCounter(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	delta1 := int64(10)
	metric1 := models.Metrics{
		ID:    "TestCounter",
		MType: models.Counter,
		Delta: &delta1,
	}

	err := storage.UpdateMetric(ctx, metric1)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что счетчик = 10
	result, err := storage.GetMetric(ctx, "TestCounter", models.Counter)
	if err != nil {
		t.Fatal("Counter not found after update")
	}

	if result.Delta == nil {
		t.Fatal("Counter delta is nil")
	}

	if *result.Delta != 10 {
		t.Errorf("Expected delta 10, got %d", *result.Delta)
	}

	// Обновляем еще раз
	delta2 := int64(5)
	metric2 := models.Metrics{
		ID:    "TestCounter",
		MType: models.Counter,
		Delta: &delta2,
	}

	err = storage.UpdateMetric(ctx, metric2)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что счетчик = 15 (накопительный)
	result, err = storage.GetMetric(ctx, "TestCounter", models.Counter)
	if err != nil {
		t.Fatal("Counter not found after second update")
	}

	if *result.Delta != 15 {
		t.Errorf("Expected accumulated delta 15, got %d", *result.Delta)
	}
}

func TestGetMetricNotFound(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	_, err := storage.GetMetric(ctx, "NonExistent", models.Gauge)
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected ErrNotFound for non-existent metric")
	}
}

func TestUpdateMetrics_Batch(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	v1, v2 := 1.1, 2.2
	d1 := int64(5)
	metrics := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: &v1},
		{ID: "g2", MType: models.Gauge, Value: &v2},
		{ID: "c1", MType: models.Counter, Delta: &d1},
	}

	if err := storage.UpdateMetrics(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetrics failed: %v", err)
	}

	m, err := storage.GetMetric(ctx, "g1", models.Gauge)
	if err != nil || m.Value == nil || *m.Value != 1.1 {
		t.Errorf("Expected g1=1.1, got %v", m.Value)
	}

	m, err = storage.GetMetric(ctx, "c1", models.Counter)
	if err != nil || m.Delta == nil || *m.Delta != 5 {
		t.Errorf("Expected c1=5, got %v", m.Delta)
	}
}

func TestUpdateMetrics_NilFieldsSkipped(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	// Метрики без Value/Delta не должны паниковать
	metrics := []models.Metrics{
		{ID: "g_nil", MType: models.Gauge, Value: nil},
		{ID: "c_nil", MType: models.Counter, Delta: nil},
	}
	if err := storage.UpdateMetrics(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetrics failed: %v", err)
	}

	// Метрики не были записаны (nil-значения пропускаются)
	_, errG := storage.GetMetric(ctx, "g_nil", models.Gauge)
	if !errors.Is(errG, ErrNotFound) {
		t.Error("Expected ErrNotFound for nil gauge")
	}
}

func TestClose_NoError(t *testing.T) {
	storage := NewMemStorage()
	if err := storage.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestConcurrentUpdates(t *testing.T) {
	storage := NewMemStorage()
	ctx := context.Background()

	// Запускаем 100 горутин, каждая обновляет счетчик
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			delta := int64(1)
			storage.UpdateMetric(ctx, models.Metrics{
				ID:    "TestCounter",
				MType: models.Counter,
				Delta: &delta,
			})
			done <- true
		}()
	}

	// Ждем завершения всех горутин
	for i := 0; i < 100; i++ {
		<-done
	}

}
