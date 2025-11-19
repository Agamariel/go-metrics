package repository

import (
	"database/sql"
	"fmt"

	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/Agamariel/go-metrics/internal/models"
	"go.uber.org/zap"
)

// реализация интерфейса Storage для PostgreSQL
type PostgresStorage struct {
	db     *sql.DB
	logger logger.Logger
}

func NewPostgresStorage(db *sql.DB, log logger.Logger) *PostgresStorage {
	if log == nil {
		log = logger.Nop()
	}
	return &PostgresStorage{
		db:     db,
		logger: log,
	}
}

// UpdateMetric реализует интерфейс Storage
func (p *PostgresStorage) UpdateMetric(metric models.Metrics) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("значение gauge не может быть nil")
		}
		_, err := p.db.Exec(
			`INSERT INTO metrics (id, type, value) 
			 VALUES ($1, $2, $3) 
			 ON CONFLICT (id, type) DO UPDATE SET value = $3`,
			metric.ID, metric.MType, *metric.Value,
		)
		if err != nil {
			p.logger.Error("Ошибка при обновлении gauge метрики",
				zap.String("id", metric.ID),
				zap.Error(err),
			)
			return fmt.Errorf("ошибка обновления gauge: %w", err)
		}
	case models.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("значение counter не может быть nil")
		}
		_, err := p.db.Exec(
			`INSERT INTO metrics (id, type, delta) 
			 VALUES ($1, $2, $3) 
			 ON CONFLICT (id, type) DO UPDATE SET delta = metrics.delta + $3`,
			metric.ID, metric.MType, *metric.Delta,
		)
		if err != nil {
			p.logger.Error("Ошибка при обновлении counter метрики",
				zap.String("id", metric.ID),
				zap.Error(err),
			)
			return fmt.Errorf("ошибка обновления counter: %w", err)
		}
	default:
		return fmt.Errorf("неизвестный тип метрики: %s", metric.MType)
	}
	return nil
}

// UpdateMetrics реализует интерфейс Storage для пакетного обновления
func (p *PostgresStorage) UpdateMetrics(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем запросы для каждого типа метрики
	stmtGauge, err := tx.Prepare(
		`INSERT INTO metrics (id, type, value) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (id, type) DO UPDATE SET value = $3`,
	)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса для gauge: %w", err)
	}
	defer stmtGauge.Close()

	stmtCounter, err := tx.Prepare(
		`INSERT INTO metrics (id, type, delta) 
		 VALUES ($1, $2, $3) 
		 ON CONFLICT (id, type) DO UPDATE SET delta = metrics.delta + $3`,
	)
	if err != nil {
		return fmt.Errorf("ошибка подготовки запроса для counter: %w", err)
	}
	defer stmtCounter.Close()

	// Выполняем обновления
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("значение gauge не может быть nil для метрики %s", metric.ID)
			}
			_, err := stmtGauge.Exec(metric.ID, metric.MType, *metric.Value)
			if err != nil {
				p.logger.Error("Ошибка при обновлении gauge метрики в батче",
					zap.String("id", metric.ID),
					zap.Error(err),
				)
				return fmt.Errorf("ошибка обновления gauge %s: %w", metric.ID, err)
			}
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("значение counter не может быть nil для метрики %s", metric.ID)
			}
			_, err := stmtCounter.Exec(metric.ID, metric.MType, *metric.Delta)
			if err != nil {
				p.logger.Error("Ошибка при обновлении counter метрики в батче",
					zap.String("id", metric.ID),
					zap.Error(err),
				)
				return fmt.Errorf("ошибка обновления counter %s: %w", metric.ID, err)
			}
		default:
			return fmt.Errorf("неизвестный тип метрики: %s для метрики %s", metric.MType, metric.ID)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetMetric реализует интерфейс Storage
func (p *PostgresStorage) GetMetric(id string, mType string) (models.Metrics, bool) {
	var result models.Metrics
	result.ID = id
	result.MType = mType

	var value sql.NullFloat64
	var delta sql.NullInt64

	err := p.db.QueryRow(
		`SELECT value, delta FROM metrics WHERE id = $1 AND type = $2`,
		id, mType,
	).Scan(&value, &delta)

	if err == sql.ErrNoRows {
		return models.Metrics{}, false
	}
	if err != nil {
		p.logger.Error("Ошибка при получении метрики",
			zap.String("id", id),
			zap.String("type", mType),
			zap.Error(err),
		)
		return models.Metrics{}, false
	}

	switch mType {
	case models.Gauge:
		if value.Valid {
			v := value.Float64
			result.Value = &v
			return result, true
		}
	case models.Counter:
		if delta.Valid {
			d := delta.Int64
			result.Delta = &d
			return result, true
		}
	}

	return models.Metrics{}, false
}

// GetAllMetrics реализует интерфейс Storage
func (p *PostgresStorage) GetAllMetrics() []models.Metrics {
	rows, err := p.db.Query(`SELECT id, type, value, delta FROM metrics`)
	if err != nil {
		p.logger.Error("Ошибка при получении всех метрик", zap.Error(err))
		return nil
	}
	defer rows.Close()

	var all []models.Metrics
	for rows.Next() {
		var m models.Metrics
		var value sql.NullFloat64
		var delta sql.NullInt64

		if err := rows.Scan(&m.ID, &m.MType, &value, &delta); err != nil {
			p.logger.Error("Ошибка при сканировании метрики", zap.Error(err))
			continue
		}

		if m.MType == models.Gauge && value.Valid {
			v := value.Float64
			m.Value = &v
		} else if m.MType == models.Counter && delta.Valid {
			d := delta.Int64
			m.Delta = &d
		}

		all = append(all, m)
	}

	if err := rows.Err(); err != nil {
		p.logger.Error("Ошибка при итерации метрик", zap.Error(err))
	}

	return all
}

// Close закрывает хранилище
func (p *PostgresStorage) Close() error {
	// Подключение к БД управляется через db.DB, не закрываем его здесь
	return nil
}

// Убедимся, что PostgresStorage удовлетворяет интерфейсу Storage
var _ Storage = (*PostgresStorage)(nil)
