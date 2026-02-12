// Package db содержит конфигурацию подключения к базе данных.
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

type DBType string

const (
	PostgreSQL DBType = "postgresql"
	SQLite     DBType = "sqlite"
)

// PoolConfig параметры пула подключений
type PoolConfig struct {
	MaxOpenConns    int           // Максимальное количество открытых подключений
	MaxIdleConns    int           // Максимальное количество неактивных подключений в пуле
	ConnMaxLifetime time.Duration // Максимальное время жизни подключения
	ConnMaxIdleTime time.Duration // Максимальное время простоя подключения
}

// PostgreSQLConfig настройки для PostgreSQL
type PostgreSQLConfig struct {
	DSN string
}

// SQLiteConfig настройки для SQLite
type SQLiteConfig struct {
	Path string
}

// Config содержит конфигурацию подключения к базе данных
type Config struct {
	Type       DBType
	Pool       PoolConfig
	PostgreSQL PostgreSQLConfig
	SQLite     SQLiteConfig
}

// DefaultPoolConfig - конфигурация пула с значениями по умолчанию
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 3 * time.Minute,
	}
}

func NewPostgreSQL(dsn string) Config {
	return Config{
		Type: PostgreSQL,
		PostgreSQL: PostgreSQLConfig{
			DSN: dsn,
		},
		Pool: DefaultPoolConfig(),
	}
}

// New создает новое подключение к базе данных и возвращает *sql.DB
func New(ctx context.Context, cfg Config, log logger.Logger) (*sql.DB, error) {
	if log == nil {
		log = logger.Nop()
	}

	// Определяем тип БД и DSN
	var driver string
	var dsn string
	var dbType DBType

	// Если тип не указан, но есть DSN в PostgreSQL - используем PostgreSQL (обратная совместимость)
	if cfg.Type == "" {
		if cfg.PostgreSQL.DSN != "" {
			dbType = PostgreSQL
		} else {
			return nil, fmt.Errorf("тип базы данных не указан и DSN не найден")
		}
	} else {
		dbType = cfg.Type
	}

	// Получаем DSN и драйвер в зависимости от типа БД
	switch dbType {
	case PostgreSQL:
		if cfg.PostgreSQL.DSN == "" {
			return nil, fmt.Errorf("DSN для PostgreSQL не может быть пустым")
		}
		driver = "pgx"
		dsn = cfg.PostgreSQL.DSN
	case SQLite:
		if cfg.SQLite.Path == "" {
			return nil, fmt.Errorf("путь к файлу SQLite не может быть пустым")
		}
		driver = "sqlite3"
		dsn = cfg.SQLite.Path
	default:
		return nil, fmt.Errorf("неподдерживаемый тип базы данных: %s", dbType)
	}

	// Открываем подключение
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	// Настраиваем пул подключений с использованием значений по умолчанию, если не указаны
	poolCfg := cfg.Pool
	if poolCfg.MaxOpenConns == 0 && poolCfg.MaxIdleConns == 0 &&
		poolCfg.ConnMaxLifetime == 0 && poolCfg.ConnMaxIdleTime == 0 {
		poolCfg = DefaultPoolConfig()
	}

	maxOpenConns := maxInt(poolCfg.MaxOpenConns, DefaultPoolConfig().MaxOpenConns)
	maxIdleConns := maxInt(poolCfg.MaxIdleConns, DefaultPoolConfig().MaxIdleConns)
	connMaxLifetime := maxDuration(poolCfg.ConnMaxLifetime, DefaultPoolConfig().ConnMaxLifetime)
	connMaxIdleTime := maxDuration(poolCfg.ConnMaxIdleTime, DefaultPoolConfig().ConnMaxIdleTime)

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	// Проверяем соединение с таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка проверки подключения к БД: %w", err)
	}

	log.Info("Подключение к базе данных установлено",
		zap.String("driver", driver),
		zap.String("type", string(dbType)),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
	)

	return db, nil
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
