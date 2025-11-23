package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
	"github.com/Agamariel/go-metrics/internal/config"
	"go.uber.org/zap"
)

func main() {
	// Инициализируем zap логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Загружаем конфигурацию
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		logger.Fatal("Ошибка при загрузке конфигурации", zap.Error(err))
	}

	// Преобразуем интервалы в time.Duration
	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	reportDuration := time.Duration(cfg.ReportInterval) * time.Second

	// Формируем полный URL сервера
	serverURL := fmt.Sprintf("http://%s", cfg.Address)

	logger.Info("Конфигурация агента",
		zap.String("server_address", serverURL),
		zap.Int("poll_interval_sec", cfg.PollInterval),
		zap.Int("report_interval_sec", cfg.ReportInterval),
		zap.Bool("hash_enabled", cfg.Key != ""),
	)

	// Создаем коллектор метрик
	collector := agent.NewMetricsCollector()

	// Создаем клиент для отправки метрик
	sender := agent.NewMetricsSender(serverURL, cfg.Key)

	// Запускаем горутину для сбора метрик
	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()

		for range ticker.C {
			collector.CollectMetrics()
			logger.Info("Метрики собраны")
		}
	}()

	// Запускаем горутину для отправки метрик
	go func() {
		ticker := time.NewTicker(reportDuration)
		defer ticker.Stop()

		for range ticker.C {
			gauges := collector.GetGauges()
			counters := collector.GetCounters()

			ctx := context.Background()
			if err := sender.SendAllMetrics(ctx, gauges, counters); err != nil {
				logger.Error("Ошибка при отправке метрик", zap.Error(err))
			} else {
				logger.Info("Метрики успешно отправлены")
			}
		}
	}()

	// Блокируем main
	select {}
}
