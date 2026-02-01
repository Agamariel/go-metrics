package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/Agamariel/go-metrics/internal/models"
)

// MemStorage — реализация интерфейса Storage в памяти.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage — конструктор.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateMetric реализует интерфейс Storage.
func (m *MemStorage) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch metric.MType {
	case models.Gauge:
		if metric.Value != nil {
			m.gauges[metric.ID] = *metric.Value
		}
	case models.Counter:
		if metric.Delta != nil {
			m.counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}

// UpdateMetrics реализует интерфейс Storage для пакетного обновления.
func (m *MemStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case models.Counter:
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	return nil
}

func (m *MemStorage) GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result models.Metrics
	result.ID = id
	result.MType = mType

	switch mType {
	case models.Gauge:
		val, ok := m.gauges[id]
		if !ok {
			return models.Metrics{}, ErrNotFound
		}
		result.Value = &val
		return result, nil
	case models.Counter:
		val, ok := m.counters[id]
		if !ok {
			return models.Metrics{}, ErrNotFound
		}
		result.Delta = &val
		return result, nil
	default:
		return models.Metrics{}, fmt.Errorf("invalid metric type: %s", mType)
	}
}

func (m *MemStorage) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Предварительно выделяем память для среза, чтобы избежать реаллокаций
	totalLen := len(m.gauges) + len(m.counters)
	all := make([]models.Metrics, 0, totalLen)
	
	for id, val := range m.gauges {
		v := val
		all = append(all, models.Metrics{
			ID:    id,
			MType: models.Gauge,
			Value: &v,
		})
	}
	for id, val := range m.counters {
		v := val
		all = append(all, models.Metrics{
			ID:    id,
			MType: models.Counter,
			Delta: &v,
		})
	}
	return all, nil
}

// Close закрывает хранилище (для MemStorage ничего не делает)
func (m *MemStorage) Close() error {
	return nil
}

// Убедимся, что MemStorage удовлетворяет интерфейсу Storage
var _ Storage = (*MemStorage)(nil)
