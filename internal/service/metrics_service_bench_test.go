package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
)

// BenchmarkMetricsService_UpdateMetricByPath измеряет скорость парсинга и обновления метрики из пути
func BenchmarkMetricsService_UpdateMetricByPath(b *testing.B) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	b.Run("Gauge", func(b *testing.B) {
		path := "gauge/test_metric/123.456"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.UpdateMetricByPath(ctx, path)
		}
	})

	b.Run("Counter", func(b *testing.B) {
		path := "counter/test_counter/100"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = service.UpdateMetricByPath(ctx, path)
		}
	})
}

// BenchmarkMetricsService_UpdateMetrics измеряет скорость пакетного обновления метрик
func BenchmarkMetricsService_UpdateMetrics(b *testing.B) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)
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
				_ = service.UpdateMetrics(ctx, metrics)
			}
		})
	}
}

// BenchmarkMetricsService_GetMetric измеряет скорость получения метрики
func BenchmarkMetricsService_GetMetric(b *testing.B) {
	storage := repository.NewMemStorage()
	service := NewMetricsService(storage)
	ctx := context.Background()

	// Подготовка
	value := 123.456
	service.UpdateGauge(ctx, "test_gauge", value)
	service.UpdateCounter(ctx, "test_counter", 100)

	b.Run("Gauge", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetMetric(ctx, "test_gauge", models.Gauge)
		}
	})

	b.Run("Counter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = service.GetMetric(ctx, "test_counter", models.Counter)
		}
	})
}

// BenchmarkMetricsService_GetAllMetrics измеряет скорость получения всех метрик
func BenchmarkMetricsService_GetAllMetrics(b *testing.B) {
	ctx := context.Background()

	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			storage := repository.NewMemStorage()
			service := NewMetricsService(storage)

			// Подготовка: добавляем метрики
			for i := 0; i < size; i++ {
				if i%2 == 0 {
					service.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i))
				} else {
					service.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = service.GetAllMetrics(ctx)
			}
		})
	}
}
