package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

// BenchmarkMemStorage_UpdateMetric измеряет скорость обновления одной метрики
func BenchmarkMemStorage_UpdateMetric(b *testing.B) {
	storage := NewMemStorage()
	ctx := context.Background()

	b.Run("Gauge", func(b *testing.B) {
		value := 123.456
		metric := models.Metrics{
			ID:    "test_gauge",
			MType: models.Gauge,
			Value: &value,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.UpdateMetric(ctx, metric)
		}
	})

	b.Run("Counter", func(b *testing.B) {
		delta := int64(10)
		metric := models.Metrics{
			ID:    "test_counter",
			MType: models.Counter,
			Delta: &delta,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = storage.UpdateMetric(ctx, metric)
		}
	})
}

// BenchmarkMemStorage_UpdateMetrics измеряет скорость пакетного обновления метрик
func BenchmarkMemStorage_UpdateMetrics(b *testing.B) {
	storage := NewMemStorage()
	ctx := context.Background()

	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			metrics := make([]models.Metrics, size)
			for i := 0; i < size; i++ {
				if i%2 == 0 {
					value := float64(i)
					metrics[i] = models.Metrics{
						ID:    fmt.Sprintf("gauge_%d", i),
						MType: models.Gauge,
						Value: &value,
					}
				} else {
					delta := int64(i)
					metrics[i] = models.Metrics{
						ID:    fmt.Sprintf("counter_%d", i),
						MType: models.Counter,
						Delta: &delta,
					}
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = storage.UpdateMetrics(ctx, metrics)
			}
		})
	}
}

// BenchmarkMemStorage_GetMetric измеряет скорость получения метрики
func BenchmarkMemStorage_GetMetric(b *testing.B) {
	storage := NewMemStorage()
	ctx := context.Background()

	// Подготовка: добавляем метрики
	value := 123.456
	storage.UpdateMetric(ctx, models.Metrics{
		ID:    "test_gauge",
		MType: models.Gauge,
		Value: &value,
	})

	delta := int64(100)
	storage.UpdateMetric(ctx, models.Metrics{
		ID:    "test_counter",
		MType: models.Counter,
		Delta: &delta,
	})

	b.Run("Gauge", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = storage.GetMetric(ctx, "test_gauge", models.Gauge)
		}
	})

	b.Run("Counter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = storage.GetMetric(ctx, "test_counter", models.Counter)
		}
	})
}

// BenchmarkMemStorage_GetAllMetrics измеряет скорость получения всех метрик
func BenchmarkMemStorage_GetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			storage := NewMemStorage()
			// Подготовка: добавляем метрики
			for i := 0; i < size; i++ {
				if i%2 == 0 {
					value := float64(i)
					storage.UpdateMetric(ctx, models.Metrics{
						ID:    fmt.Sprintf("gauge_%d", i),
						MType: models.Gauge,
						Value: &value,
					})
				} else {
					delta := int64(i)
					storage.UpdateMetric(ctx, models.Metrics{
						ID:    fmt.Sprintf("counter_%d", i),
						MType: models.Counter,
						Delta: &delta,
					})
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = storage.GetAllMetrics(ctx)
			}
		})
	}
}
