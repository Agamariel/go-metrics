// Добавляем сервис, чтобы связать бизнес-логику и репозиторий.
package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
)

var (
	ErrInvalidType  = errors.New("invalid metric type")
	ErrInvalidName  = errors.New("invalid metric name")
	ErrInvalidValue = errors.New("invalid metric value")
)

// MetricsService — сервис для управления метриками.
type MetricsService struct {
	storage repository.Storage
}

// NewMetricsService — конструктор.
func NewMetricsService(storage repository.Storage) *MetricsService {
	return &MetricsService{storage: storage}
}

// UpdateMetricByPath — обновляет метрику по данным из URL.
func (s *MetricsService) UpdateMetricByPath(ctx context.Context, path string) error {
	parts := strings.Split(path, "/")

	switch len(parts) {
	case 1:
		return ErrInvalidType
	case 2:
		return ErrInvalidName
	}

	mType, name, valueStr := parts[0], parts[1], parts[2]

	// Проверяем, что имя метрики не пустое
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	switch mType {
	case models.Gauge:
		val, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return ErrInvalidValue
		}
		m := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &val,
		}
		return s.storage.UpdateMetric(ctx, m)

	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return ErrInvalidValue
		}
		m := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &delta,
		}
		return s.storage.UpdateMetric(ctx, m)

	default:
		return ErrInvalidType
	}
}

// UpdateMetric — обновляет метрику
func (s *MetricsService) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	return s.storage.UpdateMetric(ctx, metric)
}

// UpdateMetrics — обновляет множество метрик
func (s *MetricsService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return s.storage.UpdateMetrics(ctx, metrics)
}

// UpdateGauge — обновляет gauge метрику
func (s *MetricsService) UpdateGauge(ctx context.Context, name string, value float64) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	m := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
	return s.storage.UpdateMetric(ctx, m)
}

// UpdateCounter — обновляет counter метрику
func (s *MetricsService) UpdateCounter(ctx context.Context, name string, delta int64) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	m := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}
	return s.storage.UpdateMetric(ctx, m)
}

// GetMetric — получает метрику по имени и типу.
func (s *MetricsService) GetMetric(ctx context.Context, name, mType string) (models.Metrics, error) {
	// Проверяем, что имя не пустое
	if strings.TrimSpace(name) == "" {
		return models.Metrics{}, ErrInvalidName
	}

	// Проверяем тип метрики
	if mType != models.Gauge && mType != models.Counter {
		return models.Metrics{}, ErrInvalidType
	}

	metric, err := s.storage.GetMetric(ctx, name, mType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return models.Metrics{}, ErrInvalidName
		}
		return models.Metrics{}, err
	}

	return metric, nil
}

// GetAllMetrics — получает все метрики.
func (s *MetricsService) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	return s.storage.GetAllMetrics(ctx)
}
