package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// Config содержит конфигурацию подключения к базе данных
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DB обертка над *sql.DB
type DB struct {
	*sql.DB
}

// New создает новое подключение к базе данных PostgreSQL
func New(ctx context.Context, cfg Config, log logger.Logger) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("DSN не может быть пустым")
	}

	if log == nil {
		log = logger.Nop()
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	// если в cfg 0 – ставим дефолт
	maxOpenConns := maxInt(cfg.MaxOpenConns, 25)
	maxIdleConns := maxInt(cfg.MaxIdleConns, 5)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(maxDuration(cfg.ConnMaxLifetime, 5*time.Minute))
	db.SetConnMaxIdleTime(maxDuration(cfg.ConnMaxIdleTime, 3*time.Minute))

	// Проверяем соединение с таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка проверки подключения к БД: %w", err)
	}

	log.Info("Подключение к базе данных установлено",
		zap.String("driver", "pgx"),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
	)

	return &DB{
		DB: db,
	}, nil
}

func (db *DB) Ping(ctx context.Context, log logger.Logger) error {
	if err := db.PingContext(ctx); err != nil {
		if log != nil {
			log.Error("Ошибка проверки подключения к БД", zap.Error(err))
		}
		return err
	}
	return nil
}

func (db *DB) Close(log logger.Logger) error {
	if log != nil {
		log.Info("Закрытие подключения к базе данных")
	}
	return db.DB.Close()
}

// maxInt хелпер для задания значений по умолчанию
func maxInt(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

// maxDuration хелпер для задания значений по умолчанию
func maxDuration(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}
