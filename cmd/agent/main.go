package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Agamariel/go-metrics/internal/agent"
	"github.com/Agamariel/go-metrics/internal/config"
	"github.com/Agamariel/go-metrics/pkg/buildinfo"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run содержит основную логику агента и возвращает ошибку вместо вызова os.Exit
func run() error {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	// Инициализируем zap логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("ошибка при инициализации логгера: %w", err)
	}
	defer logger.Sync()

	// Создаем контекст для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Загружаем конфигурацию
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		return fmt.Errorf("ошибка при загрузке конфигурации: %w", err)
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
		zap.Int("rate_limit", cfg.RateLimit),
		zap.Bool("hash_enabled", cfg.Key != ""),
		zap.Bool("crypto_enabled", cfg.CryptoKey != ""),
	)

	// Создаем коллектор метрик
	collector := agent.NewMetricsCollector()

	// Создаем клиент для отправки метрик
	sender, err := agent.NewMetricsSender(serverURL, cfg.Key, cfg.CryptoKey)
	if err != nil {
		return fmt.Errorf("ошибка при создании отправщика метрик: %w", err)
	}

	// Объединяем сбор метрик в одну горутину
	// будем использовать один тикер на оба сборщика
	go func() {
		ticker := time.NewTicker(pollDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collector.CollectMetrics()
				if err := collector.CollectPSUtilMetrics(); err != nil {
					logger.Error("Ошибка при сборе метрик через gopsutil", zap.Error(err))
				}
			}
		}
	}()

	// Создаем канал для задач отправки метрик (worker pool)
	jobs := make(chan agent.MetricsJob, cfg.RateLimit)

	// WaitGroup для ожидания завершения воркеров
	var wg sync.WaitGroup

	// Создаем и запускаем воркеров для отправки метрик
	for w := 1; w <= cfg.RateLimit; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				// Создаем контекст с таймаутом для отправки метрик
				sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := sender.SendAllMetrics(sendCtx, job.Gauges, job.Counters)
				cancel()

				if err != nil {
					logger.Error("Ошибка при отправке метрик",
						zap.Int("worker_id", workerID),
						zap.Error(err))
				} else {
					logger.Info("Метрики успешно отправлены",
						zap.Int("worker_id", workerID))
				}
			}
		}(w)
	}

	// Запускаем горутину для добавления задач отправки в очередь
	go func() {
		ticker := time.NewTicker(reportDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				gauges := collector.GetGauges()
				counters := collector.GetCounters()

				// Отправляем задачу в канал jobs
				select {
				case jobs <- agent.NewMetricsJob(gauges, counters):
					logger.Info("Задача отправки метрик добавлена в очередь")
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Ожидаем сигнала завершения
	<-ctx.Done()
	logger.Info("Получен сигнал завершения, начинаем graceful shutdown")

	// Финальная отправка метрик, собранных после последнего тика reportInterval
	gauges := collector.GetGauges()
	counters := collector.GetCounters()
	if len(gauges) > 0 || len(counters) > 0 {
		jobs <- agent.NewMetricsJob(gauges, counters)
		logger.Info("Финальная отправка метрик добавлена в очередь")
	}

	// Закрываем канал jobs, чтобы воркеры завершили работу
	close(jobs)

	// Ожидаем завершения всех воркеров
	wg.Wait()

	logger.Info("Агент успешно завершен")
	return nil
}
