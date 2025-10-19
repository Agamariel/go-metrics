// Добавляем сервис, чтобы связать бизнес-логику и репозиторий.
package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/Agamariel/go-metrics/internal/models"
)

var (
	ErrInvalidType  = errors.New("invalid metric type")
	ErrInvalidName  = errors.New("invalid metric name")
	ErrInvalidValue = errors.New("invalid metric value")
)

// MetricsService — сервис для управления метриками.
type MetricsService struct {
	storage Storage
}

// NewMetricsService — конструктор.
func NewMetricsService(storage Storage) *MetricsService {
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
