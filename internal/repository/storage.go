// Package repository предоставляет интерфейс и реализации хранилища метрик.
//
// Пакет определяет интерфейс Storage, который абстрагирует детали хранения данных,
// и предоставляет несколько реализаций:
//   - MemStorage — хранение в памяти (для разработки и тестирования)
//   - FileStorage — хранение в файле с периодическим сохранением
//   - PostgresStorage — хранение в PostgreSQL (для продакшена)
//
// # Принцип инверсии зависимостей
//
// Интерфейс Storage определяется на уровне бизнес-логики (service),
// что позволяет легко менять реализацию хранилища без изменения сервисного слоя.
//
// # Пример использования
//
//	// Использование in-memory хранилища
//	storage := repository.NewMemStorage()
//	defer storage.Close()
//
//	// Сохранение метрики
//	metric := models.Metrics{ID: "cpu", MType: models.Gauge, Value: &value}
//	err := storage.UpdateMetric(ctx, metric)
//
//	// Получение метрики
//	m, err := storage.GetMetric(ctx, "cpu", models.Gauge)
package repository

import (
	"context"
	"errors"

	"github.com/Agamariel/go-metrics/internal/models"
)

// ErrNotFound возвращается, когда запрошенная метрика не найдена в хранилище.
var ErrNotFound = errors.New("metric not found")

// Storage определяет интерфейс для работы с хранилищем метрик.
//
// Все методы принимают context для поддержки отмены и таймаутов.
// Реализации должны быть потокобезопасными.
type Storage interface {
	// UpdateMetric сохраняет или обновляет одну метрику.
	// Для counter-метрик значение delta прибавляется к текущему.
	// Для gauge-метрик значение замещается.
	UpdateMetric(ctx context.Context, m models.Metrics) error

	// UpdateMetrics пакетно обновляет несколько метрик.
	// Рекомендуется для эффективной отправки большого количества метрик.
	// Реализации должны обеспечивать атомарность операции.
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error

	// GetMetric возвращает метрику по идентификатору и типу.
	// Возвращает ErrNotFound, если метрика не существует.
	GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error)

	// GetAllMetrics возвращает все сохранённые метрики.
	// При отсутствии метрик возвращает пустой слайс.
	GetAllMetrics(ctx context.Context) ([]models.Metrics, error)

	// Close освобождает ресурсы хранилища.
	// Для PostgresStorage закрывает соединения с БД.
	// Для FileStorage сохраняет данные на диск.
	Close() error
}
