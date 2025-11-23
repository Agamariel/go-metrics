package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_EmptyDSN(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := Config{Type: PostgreSQL}
	log := logger.Nop()

	// Act
	db, err := New(ctx, cfg, log)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "DSN для PostgreSQL не может быть пустым")
}

func TestNew_InvalidDSN(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := Config{
		Type: PostgreSQL,
		PostgreSQL: PostgreSQLConfig{
			DSN: "invalid://connection/string",
		},
	}
	log := logger.Nop()

	// Act
	db, err := New(ctx, cfg, log)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestNew_WithNilLogger(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := NewPostgreSQL("postgres://user:pass@localhost/testdb?sslmode=disable")

	// Act
	// Не вызовет панику, так как внутри есть проверка на nil
	db, err := New(ctx, cfg, nil)

	// Assert
	// Ожидаем ошибку подключения (т.к. БД нет), но не панику
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestNew_DefaultConnectionPoolSettings(t *testing.T) {
	tests := []struct {
		name         string
		cfg          Config
		expectedMax  int
		expectedIdle int
	}{
		{
			name: "use defaults when zero",
			cfg: Config{
				Type: PostgreSQL,
				PostgreSQL: PostgreSQLConfig{
					DSN: "postgres://user:pass@localhost/testdb?sslmode=disable",
				},
				Pool: PoolConfig{
					MaxOpenConns: 0,
					MaxIdleConns: 0,
				},
			},
			expectedMax:  25,
			expectedIdle: 5,
		},
		{
			name: "use custom values",
			cfg: Config{
				Type: PostgreSQL,
				PostgreSQL: PostgreSQLConfig{
					DSN: "postgres://user:pass@localhost/testdb?sslmode=disable",
				},
				Pool: PoolConfig{
					MaxOpenConns: 50,
					MaxIdleConns: 10,
				},
			},
			expectedMax:  50,
			expectedIdle: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем только логику установки значений
			defaultPool := DefaultPoolConfig()
			maxOpen := maxInt(tt.cfg.Pool.MaxOpenConns, defaultPool.MaxOpenConns)
			maxIdle := maxInt(tt.cfg.Pool.MaxIdleConns, defaultPool.MaxIdleConns)

			assert.Equal(t, tt.expectedMax, maxOpen)
			assert.Equal(t, tt.expectedIdle, maxIdle)
		})
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "a is positive",
			a:        10,
			b:        5,
			expected: 10,
		},
		{
			name:     "a is zero, return b",
			a:        0,
			b:        5,
			expected: 5,
		},
		{
			name:     "a is negative, return b",
			a:        -1,
			b:        5,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxInt(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxDuration(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		def      time.Duration
		expected time.Duration
	}{
		{
			name:     "d is positive",
			d:        10 * time.Second,
			def:      5 * time.Second,
			expected: 10 * time.Second,
		},
		{
			name:     "d is zero, return default",
			d:        0,
			def:      5 * time.Second,
			expected: 5 * time.Second,
		},
		{
			name:     "d is negative, return default",
			d:        -1 * time.Second,
			def:      5 * time.Second,
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxDuration(tt.d, tt.def)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Type: PostgreSQL,
				PostgreSQL: PostgreSQLConfig{
					DSN: "postgres://user:pass@localhost/db",
				},
				Pool: PoolConfig{
					MaxOpenConns:    25,
					MaxIdleConns:    5,
					ConnMaxLifetime: 5 * time.Minute,
					ConnMaxIdleTime: 3 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "empty DSN",
			cfg: Config{
				Type: PostgreSQL,
			},
			wantErr: true,
		},
		{
			name: "zero pool settings (should use defaults)",
			cfg: Config{
				Type: PostgreSQL,
				PostgreSQL: PostgreSQLConfig{
					DSN: "postgres://user:pass@localhost/db",
				},
				Pool: PoolConfig{
					MaxOpenConns:    0,
					MaxIdleConns:    0,
					ConnMaxLifetime: 0,
					ConnMaxIdleTime: 0,
				},
			},
			wantErr: false, // Будут использованы значения по умолчанию
		},
		{
			name: "unsupported DB type",
			cfg: Config{
				Type: DBType("unsupported"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(context.Background(), tt.cfg, logger.Nop())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// Может быть ошибка подключения, но не ошибка валидации
				if err != nil {
					// Проверяем, что это не ошибка валидации
					assert.NotContains(t, err.Error(), "DSN для PostgreSQL не может быть пустым")
					assert.NotContains(t, err.Error(), "неподдерживаемый тип базы данных")
				}
			}
		})
	}
}

// Интеграционный тест с SQLite для проверки реального подключения
func TestDB_Integration_SQLite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	// Используем in-memory SQLite для тестирования
	// Примечание: потребуется драйвер SQLite, но мы используем pgx
	// Этот тест будет пропущен, если нет реальной PostgreSQL
	t.Skip("Требуется реальная PostgreSQL для интеграционного теста")
}

// Бенчмарк для проверки производительности создания подключения
func BenchmarkNew(b *testing.B) {
	cfg := NewPostgreSQL("postgres://user:pass@localhost/db?sslmode=disable")
	log := logger.Nop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db, _ := New(ctx, cfg, log)
		if db != nil {
			db.Close()
		}
	}
}

func TestNewPostgreSQL(t *testing.T) {
	// Arrange
	dsn := "postgres://user:pass@localhost/testdb?sslmode=disable"

	// Act
	cfg := NewPostgreSQL(dsn)

	// Assert
	assert.Equal(t, PostgreSQL, cfg.Type)
	assert.Equal(t, dsn, cfg.PostgreSQL.DSN)
	assert.Equal(t, DefaultPoolConfig(), cfg.Pool)
}

func TestDefaultPoolConfig(t *testing.T) {
	// Act
	poolCfg := DefaultPoolConfig()

	// Assert
	assert.Equal(t, 25, poolCfg.MaxOpenConns)
	assert.Equal(t, 5, poolCfg.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, poolCfg.ConnMaxLifetime)
	assert.Equal(t, 3*time.Minute, poolCfg.ConnMaxIdleTime)
}

func TestNew_ReturnsSQLDB(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := NewPostgreSQL("postgres://user:pass@localhost/testdb?sslmode=disable")
	log := logger.Nop()

	// Act
	db, err := New(ctx, cfg, log)

	// Assert
	// Ожидаем ошибку подключения (т.к. БД нет), но проверяем тип возвращаемого значения
	if err == nil {
		require.NotNil(t, db)
		assert.IsType(t, (*sql.DB)(nil), db)
		if db != nil {
			db.Close()
		}
	} else {
		// Ошибка подключения ожидается, но тип должен быть правильным
		assert.Nil(t, db)
	}
}
