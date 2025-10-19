package agent

import (
	"fmt"
	"net/http"
	"time"
)

// MetricsSender отправляет метрики на сервер
type MetricsSender struct {
	serverURL string
	client    *http.Client
}

// NewMetricsSender создает новый отправитель метрик
func NewMetricsSender(serverURL string) *MetricsSender {
	return &MetricsSender{
		serverURL: serverURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendMetric отправляет одну метрику на сервер
func (s *MetricsSender) SendMetric(metricType, metricName, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", s.serverURL, metricType, metricName, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

// SendAllMetrics отправляет все метрики на сервер
func (s *MetricsSender) SendAllMetrics(gauges map[string]float64, counters map[string]int64) error {
	// Отправляем gauge метрики
	for name, value := range gauges {
		valueStr := fmt.Sprintf("%f", value)
		if err := s.SendMetric("gauge", name, valueStr); err != nil {
			return fmt.Errorf("failed to send gauge metric %s: %w", name, err)
		}
	}

	// Отправляем counter метрики
	for name, value := range counters {
		valueStr := fmt.Sprintf("%d", value)
		if err := s.SendMetric("counter", name, valueStr); err != nil {
			return fmt.Errorf("failed to send counter metric %s: %w", name, err)
		}
	}

	return nil
}
