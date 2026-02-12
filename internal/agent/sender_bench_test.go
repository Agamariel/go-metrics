package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Agamariel/go-metrics/internal/models"
)

// BenchmarkCompressJSON измеряет скорость сжатия JSON
func BenchmarkCompressJSON(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Metrics_%d", size), func(b *testing.B) {
			// Создаем тестовые метрики
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

			jsonData, _ := json.Marshal(metrics)
			b.ResetTimer()
			b.SetBytes(int64(len(jsonData)))

			for i := 0; i < b.N; i++ {
				_, _ = compressJSON(jsonData)
			}
		})
	}
}

// BenchmarkJSONMarshal измеряет скорость сериализации метрик в JSON
func BenchmarkJSONMarshal(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Metrics_%d", size), func(b *testing.B) {
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
				_, _ = json.Marshal(metrics)
			}
		})
	}
}

// BenchmarkGzipCompression сравнивает различные уровни сжатия gzip
func BenchmarkGzipCompression(b *testing.B) {
	// Подготовка данных
	metrics := make([]models.Metrics, 100)
	for i := 0; i < 100; i++ {
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
	jsonData, _ := json.Marshal(metrics)

	levels := []int{
		gzip.BestSpeed,
		gzip.DefaultCompression,
		gzip.BestCompression,
	}

	levelNames := map[int]string{
		gzip.BestSpeed:          "BestSpeed",
		gzip.DefaultCompression: "Default",
		gzip.BestCompression:    "BestCompression",
	}

	for _, level := range levels {
		b.Run(levelNames[level], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				gz, _ := gzip.NewWriterLevel(&buf, level)
				gz.Write(jsonData)
				gz.Close()
			}
		})
	}
}
