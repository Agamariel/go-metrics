package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/pkg/retry"
	"github.com/Agamariel/go-metrics/pkg/sha256hash"
	"github.com/go-resty/resty/v2"
)

// MetricsSender отправляет метрики на сервер
type MetricsSender struct {
	serverURL string
	client    *resty.Client
	key       string
}

// NewMetricsSender создает новый клиент для отправки метрик
func NewMetricsSender(serverURL string, key string) *MetricsSender {
	client := resty.New()
	client.SetTimeout(5 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Accept-Encoding", "gzip")

	return &MetricsSender{
		serverURL: serverURL,
		client:    client,
		key:       key,
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

	// Сериализуем метрику в JSON для вычисления хеша
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Вычисляем хеш если ключ задан
	var request *resty.Request
	if s.key != "" {
		hashValue := sha256hash.CalculateSHA256(jsonData, s.key)
		request = s.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("HashSHA256", hashValue)
	} else {
		request = s.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip")
	}

	// Сжимаем данные
	compressed, err := compressJSON(metric)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	resp, err := request.
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
func (s *MetricsSender) SendAllMetrics(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
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
	return s.SendMetricsBatch(ctx, metrics)
}

// SendMetricsBatch отправляет батч метрик на сервер через /updates/
func (s *MetricsSender) SendMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	// Сериализуем метрики в JSON для вычисления хеша
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	compressed, err := compressJSON(metrics)
	if err != nil {
		return fmt.Errorf("compress: %w", err)
	}

	url := s.serverURL + "/updates/"
	strategy := retry.Linear(1*time.Second, 3*time.Second, 5*time.Second)

	// 1 начальная + 3 повтора
	const maxAttempts = 4

	return retry.Do(ctx, maxAttempts, strategy, retry.IsHTTPRetriable,
		func() error {
			// Создаем запрос с хешем если ключ задан
			var request *resty.Request
			if s.key != "" {
				hashValue := sha256hash.CalculateSHA256(jsonData, s.key)
				request = s.client.R().
					SetContext(ctx).
					SetHeader("Content-Type", "application/json").
					SetHeader("Content-Encoding", "gzip").
					SetHeader("HashSHA256", hashValue)
			} else {
				request = s.client.R().
					SetContext(ctx).
					SetHeader("Content-Type", "application/json").
					SetHeader("Content-Encoding", "gzip")
			}

			resp, err := request.
				SetBody(compressed).
				Post(url)

			if err != nil {
				return fmt.Errorf("post: %w", err)
			}
			if sc := resp.StatusCode(); sc != http.StatusOK {
				if sc >= 500 && sc < 600 {
					return fmt.Errorf("%w: status %d body %s", retry.ErrRetriable, sc, resp.Body())
				}
				return fmt.Errorf("non-retriable status %d body %s", sc, resp.Body())
			}
			return nil
		},
	)
}
