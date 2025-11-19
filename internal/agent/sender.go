package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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
	client.SetHeader("Accept-Encoding", "gzip")

	return &MetricsSender{
		serverURL: serverURL,
		client:    client,
	}
}

// compressJSON сжимает данные в формате gzip
func compressJSON(data interface{}) ([]byte, error) {
	// Сериализуем в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Сжимаем
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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

// SendMetricJSON отправляет одну метрику на сервер в формате JSON с gzip сжатием
func (s *MetricsSender) SendMetricJSON(metric models.Metrics) error {
	url := fmt.Sprintf("%s/update/", s.serverURL)

	// Сжимаем данные
	compressed, err := compressJSON(metric)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressed).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d, body: %s", resp.StatusCode(), resp.Body())
	}

	return nil
}

// SendAllMetrics отправляет все метрики на сервер используя JSON API батчами
func (s *MetricsSender) SendAllMetrics(gauges map[string]float64, counters map[string]int64) error {
	// Формируем список всех метрик
	var metrics []models.Metrics

	// Добавляем gauge метрики
	for name, value := range gauges {
		v := value // копируем значение для создания указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}

	// Добавляем counter метрики
	for name, value := range counters {
		v := value // копируем значение для создания указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &v,
		})
	}

	// Не отправляем пустые батчи
	if len(metrics) == 0 {
		return nil
	}

	// Отправляем все метрики одним батчем
	return s.SendMetricsBatch(metrics)
}

// SendMetricsBatch отправляет батч метрик на сервер через /updates/
func (s *MetricsSender) SendMetricsBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/updates/", s.serverURL)

	// Сжимаем данные
	compressed, err := compressJSON(metrics)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressed).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status: %d, body: %s", resp.StatusCode(), resp.Body())
	}

	return nil
}
