package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
	"github.com/Agamariel/go-metrics/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <output-file>")
		fmt.Println("Example: go run main.go profiles/base.pprof")
		os.Exit(1)
	}

	outputFile := os.Args[1]

	// Запускаем профилирование
	if err := profileMemory(outputFile); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Memory profile saved to %s\n", outputFile)
}

func profileMemory(outputFile string) error {
	// Создаем файл для профиля
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	// Выполняем операции, которые мы хотим профилировать
	workload()

	// Принудительно запускаем сборщик мусора перед снятием профиля
	runtime.GC()

	// Записываем профиль heap
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}

	return nil
}

// workload симулирует типичную нагрузку на систему
func workload() {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	svc := service.NewMetricsService(storage)

	// 1. Обновляем одиночные метрики
	for i := 0; i < 1000; i++ {
		gaugeName := fmt.Sprintf("gauge_%d", i)
		counterName := fmt.Sprintf("counter_%d", i)

		svc.UpdateGauge(ctx, gaugeName, float64(i)*1.5)
		svc.UpdateCounter(ctx, counterName, int64(i))
	}

	// 2. Пакетное обновление метрик
	for batch := 0; batch < 10; batch++ {
		metrics := make([]models.Metrics, 100)
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				value := float64(batch*100 + i)
				metrics[i] = models.Metrics{
					ID:    fmt.Sprintf("batch_gauge_%d_%d", batch, i),
					MType: models.Gauge,
					Value: &value,
				}
			} else {
				delta := int64(batch*100 + i)
				metrics[i] = models.Metrics{
					ID:    fmt.Sprintf("batch_counter_%d_%d", batch, i),
					MType: models.Counter,
					Delta: &delta,
				}
			}
		}
		svc.UpdateMetrics(ctx, metrics)
	}

	// 3. Чтение метрик
	for i := 0; i < 100; i++ {
		gaugeName := fmt.Sprintf("gauge_%d", i)
		counterName := fmt.Sprintf("counter_%d", i)

		svc.GetMetric(ctx, gaugeName, models.Gauge)
		svc.GetMetric(ctx, counterName, models.Counter)
	}

	// 4. Получение всех метрик (множественные вызовы)
	for i := 0; i < 50; i++ {
		svc.GetAllMetrics(ctx)
	}

	// 5. UpdateMetricByPath (парсинг строк)
	for i := 0; i < 500; i++ {
		path := fmt.Sprintf("gauge/parsed_metric_%d/%.2f", i, float64(i)*2.5)
		svc.UpdateMetricByPath(ctx, path)
	}
}
