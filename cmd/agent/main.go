package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/caarlos0/env/v6"
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

	// Значения по умолчанию
	cfg := config.NewAgentConfig()

	// Определяем флаги командной строки
	serverAddress := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", cfg.ReportInterval, "частота отправки метрик на сервер (в секундах)")
	pollInterval := flag.Int("p", cfg.PollInterval, "частота опроса метрик из пакета runtime (в секундах)")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		logger.Fatal("Ошибка: неизвестные аргументы", zap.Strings("args", flag.Args()))
	}

	// Применяем значения из флагов
	cfg.Address = *serverAddress
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval

	// Парсим переменные окружения (приоритет выше флагов)
	if err := env.Parse(&cfg); err != nil {
		logger.Fatal("Ошибка при парсинге переменных окружения", zap.Error(err))
	}

	// Минимальные проверки значений
	if cfg.ReportInterval <= 0 {
		logger.Fatal("Ошибка: reportInterval должен быть положительным числом", zap.Int("reportInterval", cfg.ReportInterval))
	}
	if cfg.PollInterval <= 0 {
		logger.Fatal("Ошибка: pollInterval должен быть положительным числом", zap.Int("pollInterval", cfg.PollInterval))
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
	)

	// Создаем коллектор метрик
	collector := agent.NewMetricsCollector()

	// Создаем клиент для отправки метрик
	sender := agent.NewMetricsSender(serverURL)

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

			if err := sender.SendAllMetrics(gauges, counters); err != nil {
				logger.Error("Ошибка при отправке метрик", zap.Error(err))
			} else {
				logger.Info("Метрики успешно отправлены")
			}
		}
	}()

	// Блокируем main
	select {}
}
