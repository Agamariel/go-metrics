package repository

import "github.com/Agamariel/go-metrics/internal/models"

// Storage описывает интерфейс доступа к метрикам.
// Определяется на уровне бизнес-логики, чтобы быть независимым от деталей хранения.
type Storage interface {
	UpdateMetric(m models.Metrics) error
	GetMetric(id string, mType string) (models.Metrics, bool)
	GetAllMetrics() []models.Metrics
}
