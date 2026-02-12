// Package service содержит бизнес-логику для работы с метриками.
//
// Пакет предоставляет MetricsService, который связывает HTTP-обработчики
// с хранилищем данных, выполняя валидацию и преобразование входных данных.
//
// # Основные возможности
//
//   - Обновление gauge-метрик (замена значения)
//   - Обновление counter-метрик (накопление значения)
//   - Пакетное обновление метрик
//   - Получение метрик по имени и типу
//   - Получение списка всех метрик
//
// # Пример использования
//
//	storage := repository.NewMemStorage()
//	svc := service.NewMetricsService(storage)
//
//	// Обновление gauge-метрики
//	err := svc.UpdateGauge(ctx, "temperature", 36.6)
//
//	// Обновление counter-метрики
//	err := svc.UpdateCounter(ctx, "requests", 1)
//
//	// Получение метрики
//	metric, err := svc.GetMetric(ctx, "temperature", models.Gauge)
package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/repository"
)

// Ошибки валидации метрик.
var (
	// ErrInvalidType возвращается при указании неподдерживаемого типа метрики.
	// Поддерживаемые типы: "gauge" и "counter".
	ErrInvalidType = errors.New("invalid metric type")

	// ErrInvalidName возвращается при пустом или некорректном имени метрики.
	ErrInvalidName = errors.New("invalid metric name")

	// ErrInvalidValue возвращается при невозможности распарсить значение метрики.
	// Для gauge ожидается float64, для counter — int64.
	ErrInvalidValue = errors.New("invalid metric value")
)

// MetricsService предоставляет методы для управления метриками.
//
// Сервис выполняет валидацию входных данных и делегирует
// операции хранения в репозиторий через интерфейс Storage.
type MetricsService struct {
	storage repository.Storage
}

// NewMetricsService создаёт новый экземпляр сервиса метрик.
//
// Параметр storage должен реализовывать интерфейс repository.Storage.
// Можно использовать MemStorage для хранения в памяти или PostgresStorage
// для персистентного хранения.
func NewMetricsService(storage repository.Storage) *MetricsService {
	return &MetricsService{storage: storage}
}

// UpdateMetricByPath обновляет метрику по данным из URL-пути.
//
// Ожидаемый формат пути: "{type}/{name}/{value}".
//
// Возвращаемые ошибки:
//   - ErrInvalidType: неверный тип метрики
//   - ErrInvalidName: пустое имя метрики
//   - ErrInvalidValue: невозможно распарсить значение
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

// UpdateMetric обновляет одну метрику в хранилище.
//
// Делегирует операцию напрямую в хранилище без дополнительной валидации.
func (s *MetricsService) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	return s.storage.UpdateMetric(ctx, metric)
}

// UpdateMetrics пакетно обновляет несколько метрик.
//
// Все метрики обновляются в рамках одной транзакции (если хранилище поддерживает).
// При пустом слайсе метод возвращает nil без обращения к хранилищу.
func (s *MetricsService) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return s.storage.UpdateMetrics(ctx, metrics)
}

// UpdateGauge обновляет gauge-метрику с указанным именем и значением.
//
// Gauge-метрики замещают предыдущее значение при обновлении.
//
// Возвращает ErrInvalidName, если имя пустое или содержит только пробелы.
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

// UpdateCounter обновляет counter-метрику с указанным именем и delta-значением.
//
// Counter-метрики накапливают значения: delta прибавляется к текущему значению.
//
// Возвращает ErrInvalidName, если имя пустое или содержит только пробелы.
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

// GetMetric возвращает метрику по имени и типу.
//
// Параметры:
//   - name: имя метрики (не может быть пустым)
//   - mType: тип метрики ("gauge" или "counter")
//
// Возвращаемые ошибки:
//   - ErrInvalidName: пустое имя или метрика не найдена
//   - ErrInvalidType: неподдерживаемый тип метрики
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

// GetAllMetrics возвращает все сохранённые метрики.
//
// Возвращает слайс всех gauge и counter метрик из хранилища.
// При отсутствии метрик возвращает пустой слайс.
func (s *MetricsService) GetAllMetrics(ctx context.Context) ([]models.Metrics, error) {
	return s.storage.GetAllMetrics(ctx)
}
