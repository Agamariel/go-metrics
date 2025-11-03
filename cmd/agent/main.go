package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
	"github.com/caarlos0/env/v6"
)

// Config содержит конфигурацию агента
type Config struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {
	// Значения по умолчанию
	cfg := Config{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}

	// Определяем флаги командной строки
	serverAddress := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", cfg.ReportInterval, "частота отправки метрик на сервер (в секундах)")
	pollInterval := flag.Int("p", cfg.PollInterval, "частота опроса метрик из пакета runtime (в секундах)")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		log.Fatalf("Ошибка: неизвестные аргументы: %v", flag.Args())
	}

	// Применяем значения из флагов
	cfg.Address = *serverAddress
	cfg.ReportInterval = *reportInterval
	cfg.PollInterval = *pollInterval

	// Парсим переменные окружения (приоритет выше флагов)
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Ошибка при парсинге переменных окружения: %v", err)
	}

	// Минимальные проверки значений
	if cfg.ReportInterval <= 0 {
		log.Fatalf("Ошибка: reportInterval должен быть положительным числом, получено: %d", cfg.ReportInterval)
	}
	if cfg.PollInterval <= 0 {
		log.Fatalf("Ошибка: pollInterval должен быть положительным числом, получено: %d", cfg.PollInterval)
	}

	// Преобразуем интервалы в time.Duration
	pollDuration := time.Duration(cfg.PollInterval) * time.Second
	reportDuration := time.Duration(cfg.ReportInterval) * time.Second

	// Формируем полный URL сервера
	serverURL := fmt.Sprintf("http://%s", cfg.Address)

	log.Printf("Agent configuration:")
	log.Printf("  Server address: %s", serverURL)
	log.Printf("  Poll interval: %d seconds", cfg.PollInterval)
	log.Printf("  Report interval: %d seconds", cfg.ReportInterval)

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
			log.Println("Metrics collected")
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
				log.Printf("Failed to send metrics: %v", err)
			} else {
				log.Println("Metrics sent successfully")
			}
		}
	}()

	// Блокируем main
	select {}
}
