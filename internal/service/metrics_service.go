// Добавляем сервис, чтобы связать бизнес-логику и репозиторий.
package service

import (
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

// MetricsServiceInterface определяет интерфейс для работы с метриками
type MetricsServiceInterface interface {
	UpdateMetricByPath(path string) error
	UpdateMetric(metric models.Metrics) error
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, delta int64) error
	GetMetric(name, mType string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
}

// MetricsService — сервис для управления метриками.
type MetricsService struct {
	storage repository.Storage
}

// NewMetricsService — конструктор.
func NewMetricsService(storage repository.Storage) *MetricsService {
	return &MetricsService{storage: storage}
}

// UpdateMetricByPath — обновляет метрику по данным из URL.
func (s *MetricsService) UpdateMetricByPath(path string) error {
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
		return s.storage.UpdateMetric(m)

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
		return s.storage.UpdateMetric(m)

	default:
		return ErrInvalidType
	}
}

// UpdateMetric — обновляет метрику
func (s *MetricsService) UpdateMetric(metric models.Metrics) error {
	return s.storage.UpdateMetric(metric)
}

// UpdateGauge — обновляет gauge метрику
func (s *MetricsService) UpdateGauge(name string, value float64) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	m := models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &value,
	}
	return s.storage.UpdateMetric(m)
}

// UpdateCounter — обновляет counter метрику
func (s *MetricsService) UpdateCounter(name string, delta int64) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}

	m := models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}
	return s.storage.UpdateMetric(m)
}

// GetMetric — получает метрику по имени и типу.
func (s *MetricsService) GetMetric(name, mType string) (models.Metrics, error) {
	// Проверяем, что имя не пустое
	if strings.TrimSpace(name) == "" {
		return models.Metrics{}, ErrInvalidName
	}

	// Проверяем тип метрики
	if mType != models.Gauge && mType != models.Counter {
		return models.Metrics{}, ErrInvalidType
	}

	metric, found := s.storage.GetMetric(name, mType)
	if !found {
		return models.Metrics{}, ErrInvalidName
	}

	return metric, nil
}

// GetAllMetrics — получает все метрики.
func (s *MetricsService) GetAllMetrics() []models.Metrics {
	return s.storage.GetAllMetrics()
}
