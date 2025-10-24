package repository

import (
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

	value := 123.456
	metric := models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
		Value: &value,
	}

	err := storage.UpdateMetric(metric)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что метрика сохранена
	result, ok := storage.GetMetric("TestGauge", models.Gauge)
	if !ok {
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

	delta1 := int64(10)
	metric1 := models.Metrics{
		ID:    "TestCounter",
		MType: models.Counter,
		Delta: &delta1,
	}

	err := storage.UpdateMetric(metric1)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что счетчик = 10
	result, ok := storage.GetMetric("TestCounter", models.Counter)
	if !ok {
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

	err = storage.UpdateMetric(metric2)
	if err != nil {
		t.Errorf("UpdateMetric failed: %v", err)
	}

	// Проверяем, что счетчик = 15 (накопительный)
	result, ok = storage.GetMetric("TestCounter", models.Counter)
	if !ok {
		t.Fatal("Counter not found after second update")
	}

	if *result.Delta != 15 {
		t.Errorf("Expected accumulated delta 15, got %d", *result.Delta)
	}
}

func TestGetMetricNotFound(t *testing.T) {
	storage := NewMemStorage()

	_, ok := storage.GetMetric("NonExistent", models.Gauge)
	if ok {
		t.Error("Expected false for non-existent metric")
	}
}

func TestConcurrentUpdates(t *testing.T) {
	storage := NewMemStorage()

	// Запускаем 100 горутин, каждая обновляет счетчик
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			delta := int64(1)
			storage.UpdateMetric(models.Metrics{
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
