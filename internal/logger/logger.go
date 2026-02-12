// Package logger предоставляет интерфейс для структурированного логирования.
package logger

import "go.uber.org/zap"

// Logger определяет интерфейс для логирования в приложении
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
}

// Nop возвращает no-op логгер, который ничего не делает
func Nop() Logger {
	return &nopLogger{}
}

type nopLogger struct{}

func (n *nopLogger) Info(msg string, fields ...zap.Field)  {}
func (n *nopLogger) Error(msg string, fields ...zap.Field) {}
func (n *nopLogger) Warn(msg string, fields ...zap.Field)  {}
func (n *nopLogger) Debug(msg string, fields ...zap.Field) {}
func (n *nopLogger) Fatal(msg string, fields ...zap.Field) {}

// ZapAdapter преобразует *zap.Logger в интерфейс Logger
type ZapAdapter struct {
	*zap.Logger
}

// NewZapAdapter создает новый адаптер для zap.Logger
func NewZapAdapter(l *zap.Logger) Logger {
	if l == nil {
		return Nop()
	}
	return &ZapAdapter{Logger: l}
}
