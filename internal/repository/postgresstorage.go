package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/pkg/retry"
	"go.uber.org/zap"
)

const (
	// defaultRetryAttempts — количество попыток для retry операций (1 начальная + 3 повтора)
	defaultRetryAttempts = 4
)

// реализация интерфейса Storage для PostgreSQL
type PostgresStorage struct {
	db     *sql.DB
	logger logger.Logger
	retry  struct {
		strategy    retry.Strategy
		maxAttempts int
	}
}

func NewPostgresStorage(db *sql.DB, log logger.Logger) *PostgresStorage {
	if log == nil {
		log = logger.Nop()
	}
	s := &PostgresStorage{
		db:     db,
		logger: log,
	}
	s.retry.strategy = retry.Linear(1*time.Second, 3*time.Second, 5*time.Second)
	s.retry.maxAttempts = defaultRetryAttempts
	return s
}

// UpdateMetric реализует интерфейс Storage
func (p *PostgresStorage) UpdateMetric(ctx context.Context, metric models.Metrics) error {
	return retry.Do(ctx, p.retry.maxAttempts, p.retry.strategy, retry.IsPostgresRetriableError, func() error {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("значение gauge не может быть nil")
			}
			_, err := p.db.ExecContext(ctx,
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
			_, err := p.db.ExecContext(ctx,
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
	})
}

// UpdateMetrics реализует интерфейс Storage для пакетного обновления
func (p *PostgresStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	return retry.Do(ctx, p.retry.maxAttempts, p.retry.strategy, retry.IsPostgresRetriableError, func() error {
		// Начинаем транзакцию
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("ошибка начала транзакции: %w", err)
		}
		defer tx.Rollback()

		// Подготавливаем запросы для каждого типа метрики
		stmtGauge, err := tx.PrepareContext(ctx,
			`INSERT INTO metrics (id, type, value) 
			 VALUES ($1, $2, $3) 
			 ON CONFLICT (id, type) DO UPDATE SET value = $3`,
		)
		if err != nil {
			return fmt.Errorf("ошибка подготовки запроса для gauge: %w", err)
		}
		defer stmtGauge.Close()

		stmtCounter, err := tx.PrepareContext(ctx,
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
				_, err := stmtGauge.ExecContext(ctx, metric.ID, metric.MType, *metric.Value)
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
				_, err := stmtCounter.ExecContext(ctx, metric.ID, metric.MType, *metric.Delta)
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
	})
}

// GetMetric реализует интерфейс Storage
func (p *PostgresStorage) GetMetric(ctx context.Context, id string, mType string) (models.Metrics, error) {
	var result models.Metrics
	result.ID = id
	result.MType = mType

	var value sql.NullFloat64
	var delta sql.NullInt64

	err := retry.Do(ctx, p.retry.maxAttempts, p.retry.strategy, retry.IsPostgresRetriableError, func() error {
		return p.db.QueryRowContext(ctx,
			`SELECT value, delta FROM metrics WHERE id = $1 AND type = $2`,
			id, mType,
		).Scan(&value, &delta)
	})

	if err == sql.ErrNoRows {
		return models.Metrics{}, ErrNotFound
	}
	if err != nil {
		p.logger.Error("Ошибка при получении метрики",
			zap.String("id", id),
			zap.String("type", mType),
			zap.Error(err),
		)
		return models.Metrics{}, fmt.Errorf("get metric: %w", err)
	}

	switch mType {
	case models.Gauge:
		if value.Valid {
			v := value.Float64
			result.Value = &v
			return result, nil
		}
	case models.Counter:
		if delta.Valid {
			d := delta.Int64
			result.Delta = &d
			return result, nil
		}
	}

	return models.Metrics{}, ErrNotFound
}

// GetAllMetrics реализует интерфейс Storage
func (p *PostgresStorage) GetAllMetrics(ctx context.Context) []models.Metrics {
	var rows *sql.Rows
	var err error

	err = retry.Do(ctx, p.retry.maxAttempts, p.retry.strategy, retry.IsPostgresRetriableError, func() error {
		rows, err = p.db.QueryContext(ctx, `SELECT id, type, value, delta FROM metrics`)
		if err != nil {
			return err
		}
		return nil
	})

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

		if scanErr := rows.Scan(&m.ID, &m.MType, &value, &delta); scanErr != nil {
			p.logger.Error("Ошибка при сканировании метрики", zap.Error(scanErr))
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

	// Проверяем ошибки, возникшие во время итерации
	if rowsErr := rows.Err(); rowsErr != nil {
		p.logger.Error("Ошибка при итерации метрик", zap.Error(rowsErr))
		return nil
	}

	return all
}

// Close закрывает хранилище
// PostgresStorage не владеет *sql.DB — подключение управляется через db.DB в cmd/server/main.go
func (p *PostgresStorage) Close() error {
	return nil
}

// Убедимся, что PostgresStorage удовлетворяет интерфейсу Storage
var _ Storage = (*PostgresStorage)(nil)
