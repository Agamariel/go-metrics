package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
)

func main() {
	// Определяем флаги командной строки
	serverAddress := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик на сервер (в секундах)")
	pollInterval := flag.Int("p", 2, "частота опроса метрик из пакета runtime (в секундах)")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		log.Fatalf("Ошибка: неизвестные аргументы: %v", flag.Args())
	}

	// Минимальные проверки значений
	if *reportInterval <= 0 {
		log.Fatalf("Ошибка: reportInterval должен быть положительным числом, получено: %d", *reportInterval)
	}
	if *pollInterval <= 0 {
		log.Fatalf("Ошибка: pollInterval должен быть положительным числом, получено: %d", *pollInterval)
	}

	// Преобразуем интервалы в time.Duration
	pollDuration := time.Duration(*pollInterval) * time.Second
	reportDuration := time.Duration(*reportInterval) * time.Second

	// Формируем полный URL сервера
	serverURL := fmt.Sprintf("http://%s", *serverAddress)

	log.Printf("Agent configuration:")
	log.Printf("  Server address: %s", serverURL)
	log.Printf("  Poll interval: %d seconds", *pollInterval)
	log.Printf("  Report interval: %d seconds", *reportInterval)

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
