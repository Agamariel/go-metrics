package agent

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/go-resty/resty/v2"
)

// MetricsSender отправляет метрики на сервер
type MetricsSender struct {
	serverURL string
	client    *resty.Client
}

// NewMetricsSender создает новый клиент для отправки метрик
func NewMetricsSender(serverURL string) *MetricsSender {
	client := resty.New()
	client.SetTimeout(5 * time.Second)
	client.SetHeader("Content-Type", "application/json")

	return &MetricsSender{
		serverURL: serverURL,
		client:    client,
	}
}

// SendMetric отправляет одну метрику на сервер
func (s *MetricsSender) SendMetric(metricType, metricName, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", s.serverURL, metricType, metricName, value)

	resp, err := s.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode())
	}

	return nil
}

// SendMetricJSON отправляет одну метрику на сервер в формате JSON
func (s *MetricsSender) SendMetricJSON(metric models.Metrics) error {
	url := fmt.Sprintf("%s/update/", s.serverURL)

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d, body: %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// SendAllMetrics отправляет все метрики на сервер используя JSON API
func (s *MetricsSender) SendAllMetrics(gauges map[string]float64, counters map[string]int64) error {
	// Отправляем gauge метрики
	for name, value := range gauges {
		v := value // копируем значение для создания указателя
		metric := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		}
		if err := s.SendMetricJSON(metric); err != nil {
			return fmt.Errorf("failed to send gauge metric %s: %w", name, err)
		}
	}

	// Отправляем counter метрики
	for name, value := range counters {
		v := value // копируем значение для создания указателя
		metric := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &v,
		}
		if err := s.SendMetricJSON(metric); err != nil {
			return fmt.Errorf("failed to send counter metric %s: %w", name, err)
		}
	}

	return nil
}
