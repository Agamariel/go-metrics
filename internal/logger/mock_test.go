package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewMock(t *testing.T) {
	// Act
	mock := NewMock()

	// Assert
	assert.NotNil(t, mock)
	assert.Implements(t, (*Logger)(nil), mock)
}

func TestMockLogger_Methods(t *testing.T) {
	// Arrange
	mock := NewMock()

	// Настраиваем ожидания
	mock.On("Info", "test info", []zap.Field(nil))
	mock.On("Error", "test error", []zap.Field(nil))
	mock.On("Warn", "test warn", []zap.Field(nil))
	mock.On("Debug", "test debug", []zap.Field(nil))
	mock.On("Fatal", "test fatal", []zap.Field(nil))

	// Act
	mock.Info("test info")
	mock.Error("test error")
	mock.Warn("test warn")
	mock.Debug("test debug")
	mock.Fatal("test fatal")

	// Assert
	mock.AssertExpectations(t)
}

func TestMockLogger_InfoWithFields(t *testing.T) {
	// Arrange
	mock := NewMock()
	fields := []zap.Field{zap.String("key", "value")}

	mock.On("Info", "message", fields)

	// Act
	mock.Info("message", fields...)

	// Assert
	mock.AssertCalled(t, "Info", "message", fields)
}

func TestMockLogger_ErrorWithFields(t *testing.T) {
	// Arrange
	mock := NewMock()
	fields := []zap.Field{zap.Error(assert.AnError)}

	mock.On("Error", "error occurred", fields)

	// Act
	mock.Error("error occurred", fields...)

	// Assert
	mock.AssertCalled(t, "Error", "error occurred", fields)
}

func TestMockLogger_MultipleInvocations(t *testing.T) {
	// Arrange
	mock := NewMock()

	mock.On("Info", "first call", []zap.Field(nil)).Once()
	mock.On("Info", "second call", []zap.Field(nil)).Once()

	// Act
	mock.Info("first call")
	mock.Info("second call")

	// Assert
	mock.AssertNumberOfCalls(t, "Info", 2)
}
