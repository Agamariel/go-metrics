package repository

import (
	"context"
	"errors"

	"github.com/Agamariel/go-metrics/internal/models"
)

// ErrNotFound возвращается, когда метрика не найдена
var ErrNotFound = errors.New("metric not found")

// Storage описывает интерфейс доступа к метрикам.
// Определяется на уровне бизнес-логики, чтобы быть независимым от деталей хранения.
type Storage interface {
	UpdateMetric(ctx context.Context, m models.Metrics) error
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error)
	GetAllMetrics(ctx context.Context) []models.Metrics
	Close() error
}
