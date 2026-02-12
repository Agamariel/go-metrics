package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/Agamariel/go-metrics/internal/models"
)

// MemStorage реализует интерфейс Storage с хранением данных в памяти.
//
// Особенности:
//   - Потокобезопасен (использует sync.RWMutex)
//   - Данные теряются при перезапуске приложения
//   - Подходит для разработки, тестирования и сценариев без персистентности
//
// Для сохранения данных между перезапусками используйте FileStorage
// или PostgresStorage.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage создаёт новое хранилище метрик в памяти.
//
// Пример использования:
//
//	storage := repository.NewMemStorage()
//	defer storage.Close()
//
//	value := 42.5
//	metric := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &value}
//	storage.UpdateMetric(ctx, metric)
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateMetric сохраняет или обновляет метрику в памяти.
//
// Для gauge-метрик значение замещается.
// Для counter-метрик значение delta прибавляется к текущему.
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

// UpdateMetrics пакетно обновляет несколько метрик за одну операцию.
//
// Все обновления выполняются атомарно под одной блокировкой,
// что эффективнее множественных вызовов UpdateMetric.
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

// GetMetric возвращает метрику по идентификатору и типу.
//
// Возвращает ErrNotFound, если метрика с указанным id и типом не существует.
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

// GetAllMetrics возвращает все сохранённые метрики (gauge и counter).
//
// Возвращает пустой слайс, если метрики отсутствуют.
// Порядок метрик не гарантирован.
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

// Close освобождает ресурсы хранилища.
//
// Для MemStorage метод ничего не делает, так как нет внешних ресурсов.
// Предоставлен для совместимости с интерфейсом Storage.
func (m *MemStorage) Close() error {
	return nil
}

// Проверка времени компиляции: MemStorage реализует интерфейс Storage.
var _ Storage = (*MemStorage)(nil)
