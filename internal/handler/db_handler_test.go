package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agamariel/go-metrics/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDB — mock для интерфейса DB
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Ping(ctx context.Context, log logger.Logger) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func TestNewDbHandler(t *testing.T) {
	tests := []struct {
		name   string
		db     DB
		logger logger.Logger
	}{
		{
			name:   "with logger",
			db:     &MockDB{},
			logger: logger.NewMock(),
		},
		{
			name:   "without logger (should use Nop)",
			db:     &MockDB{},
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewDbHandler(tt.db, tt.logger)
			assert.NotNil(t, handler)
			assert.NotNil(t, handler.logger)
			assert.Equal(t, tt.db, handler.db)
		})
	}
}

func TestDbHandler_PingDB_Success(t *testing.T) {
	// Arrange
	mockDB := new(MockDB)
	mockLogger := logger.NewMock()

	// Ожидаем вызов Ping, который вернет nil (успех)
	mockDB.On("Ping", mock.Anything, mockLogger).Return(nil)

	handler := NewDbHandler(mockDB, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
	mockDB.AssertExpectations(t)
}

func TestDbHandler_PingDB_DatabaseError(t *testing.T) {
	// Arrange
	mockDB := new(MockDB)
	mockLogger := logger.NewMock()

	expectedErr := errors.New("connection refused")

	// Ожидаем вызов Ping, который вернет ошибку
	mockDB.On("Ping", mock.Anything, mockLogger).Return(expectedErr)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	handler := NewDbHandler(mockDB, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Ошибка проверки соединения с БД")
	mockDB.AssertExpectations(t)
}

func TestDbHandler_PingDB_NilDatabase(t *testing.T) {
	// Arrange
	mockLogger := logger.NewMock()
	handler := NewDbHandler(nil, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "База данных не сконфигурирована")
}

func TestDbHandler_PingDB_WithContext(t *testing.T) {
	// Arrange
	mockDB := new(MockDB)
	mockLogger := logger.NewMock()

	// Проверяем, что контекст передается в Ping
	mockDB.On("Ping", mock.MatchedBy(func(ctx context.Context) bool {
		// Проверяем, что контекст имеет deadline (из-за WithTimeout)
		_, ok := ctx.Deadline()
		return ok
	}), mockLogger).Return(nil)

	handler := NewDbHandler(mockDB, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)
	mockDB.AssertExpectations(t)
}

func TestDbHandler_PingDB_ContextCancellation(t *testing.T) {
	// Arrange
	mockDB := new(MockDB)
	mockLogger := logger.NewMock()

	expectedErr := context.DeadlineExceeded

	mockDB.On("Ping", mock.Anything, mockLogger).Return(expectedErr)
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	handler := NewDbHandler(mockDB, mockLogger)

	// Создаем запрос с уже отмененным контекстом
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	req := httptest.NewRequest(http.MethodGet, "/ping", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.PingDB(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockDB.AssertExpectations(t)
}

// Бенчмарк для проверки производительности
func BenchmarkDbHandler_PingDB(b *testing.B) {
	mockDB := new(MockDB)
	mockLogger := logger.Nop()

	mockDB.On("Ping", mock.Anything, mockLogger).Return(nil)

	handler := NewDbHandler(mockDB, mockLogger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.PingDB(w, req)
	}
}
