package main

import (
	"log"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
)

const (
	serverURL      = "http://localhost:8080"
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func main() {
	// Создаем коллектор метрик
	collector := agent.NewMetricsCollector()

	// Создаем отправитель метрик
	sender := agent.NewMetricsSender(serverURL)

	// Запускаем горутину для сбора метрик
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for range ticker.C {
			collector.CollectMetrics()
			log.Println("Metrics collected")
		}
	}()

	// Запускаем горутину для отправки метрик
	go func() {
		ticker := time.NewTicker(reportInterval)
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
