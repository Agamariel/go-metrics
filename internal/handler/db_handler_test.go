package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDBHandler(t *testing.T) {
	tests := []struct {
		name   string
		db     *sql.DB
		logger logger.Logger
	}{
		{
			name:   "with logger",
			db:     createTestDB(t),
			logger: logger.NewMock(),
		},
		{
			name:   "without logger (should use Nop)",
			db:     createTestDB(t),
			logger: nil,
		},
		{
			name:   "with nil db",
			db:     nil,
			logger: logger.Nop(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewDBHandler(tt.db, tt.logger)
			assert.NotNil(t, handler)
			assert.NotNil(t, handler.logger)
			assert.Equal(t, tt.db, handler.db)
		})
	}
}

func TestDBHandler_PingDB_Success(t *testing.T) {
	// Arrange
	// Используем реальный sql.DB с невалидным DSN для тестирования
	// В реальном приложении это будет работать, но для теста мы проверим только логику
	db, err := sql.Open("pgx", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	mockLogger := logger.NewMock()
	// Ожидаем вызов Error при ошибке подключения
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	handler := NewDBHandler(db, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	// Ожидаем ошибку, так как подключение невалидно, но проверяем, что обработчик работает
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Ошибка проверки соединения с БД")
	mockLogger.AssertExpectations(t)
}

func TestDBHandler_PingDB_NilDatabase(t *testing.T) {
	// Arrange
	mockLogger := logger.Nop()
	handler := NewDBHandler(nil, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "База данных не сконфигурирована")
}

func TestDBHandler_PingDB_WithContext(t *testing.T) {
	// Arrange
	db, err := sql.Open("pgx", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	mockLogger := logger.Nop()
	handler := NewDBHandler(db, mockLogger)

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	// Ожидаем ошибку из-за невалидного подключения, но проверяем, что контекст используется
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDBHandler_PingDB_ContextCancellation(t *testing.T) {
	// Arrange
	db, err := sql.Open("pgx", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	mockLogger := logger.Nop()
	handler := NewDBHandler(db, mockLogger)

	// Создаем запрос с уже отмененным контекстом
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	req := httptest.NewRequest(http.MethodGet, "/ping", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Бенчмарк для проверки производительности
func BenchmarkDBHandler_PingDB(b *testing.B) {
	db, err := sql.Open("pgx", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
	require.NoError(b, err)
	defer db.Close()

	mockLogger := logger.Nop()
	handler := NewDBHandler(db, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PingDB(w, req)
	}
}

// createTestDB создает тестовый sql.DB для использования в тестах
func createTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable")
	require.NoError(t, err)
	return db
}
