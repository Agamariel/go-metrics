package repository

import (
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
func (m *MemStorage) UpdateMetric(metric models.Metrics) error {
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

func (m *MemStorage) GetMetric(id string, mType string) (models.Metrics, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result models.Metrics
	result.ID = id
	result.MType = mType

	switch mType {
	case models.Gauge:
		val, ok := m.gauges[id]
		if !ok {
			return models.Metrics{}, false
		}
		result.Value = &val
		return result, true
	case models.Counter:
		val, ok := m.counters[id]
		if !ok {
			return models.Metrics{}, false
		}
		result.Delta = &val
		return result, true
	default:
		return models.Metrics{}, false
	}
}

func (m *MemStorage) GetAllMetrics() []models.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []models.Metrics
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
	return all
}

// Close закрывает хранилище (для MemStorage ничего не делает)
func (m *MemStorage) Close() error {
	return nil
}

// Убедимся, что MemStorage удовлетворяет интерфейсу Storage
var _ Storage = (*MemStorage)(nil)
