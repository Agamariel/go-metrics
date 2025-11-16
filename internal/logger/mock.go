package logger

import (
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockLogger — mock реализация интерфейса Logger для тестирования
// Используйте этот mock во всех тестах, где требуется имитация логгера
type MockLogger struct {
	mock.Mock
}

// Info записывает информационное сообщение
func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// Error записывает сообщение об ошибке
func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// Warn записывает предупреждение
func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// Debug записывает отладочное сообщение
func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// Fatal записывает фатальное сообщение
func (m *MockLogger) Fatal(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// NewMock создает новый экземпляр MockLogger
func NewMock() *MockLogger {
	return &MockLogger{}
}
