package repository

import (
	"sync"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/service"
)

// MemStorage — реализация интерфейса service.Storage в памяти.
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

// UpdateMetric реализует интерфейс service.Storage.
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

// Убедимся, что MemStorage удовлетворяет интерфейсу service.Storage
var _ service.Storage = (*MemStorage)(nil)
