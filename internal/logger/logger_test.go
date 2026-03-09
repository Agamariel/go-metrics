package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewZapAdapter_WithLogger(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	l := NewZapAdapter(zapLogger)
	require.NotNil(t, l)
	_, ok := l.(*ZapAdapter)
	assert.True(t, ok, "должен вернуть *ZapAdapter")
}

func TestNewZapAdapter_NilLogger(t *testing.T) {
	l := NewZapAdapter(nil)
	require.NotNil(t, l)
	_, ok := l.(*nopLogger)
	assert.True(t, ok, "nil logger должен вернуть nopLogger")
}

func TestNopLogger_Methods(t *testing.T) {
	l := Nop()
	require.NotNil(t, l)

	// Все методы должны быть no-op (не паниковать)
	assert.NotPanics(t, func() { l.Info("msg") })
	assert.NotPanics(t, func() { l.Error("msg") })
	assert.NotPanics(t, func() { l.Warn("msg") })
	assert.NotPanics(t, func() { l.Debug("msg") })
	assert.NotPanics(t, func() { l.Fatal("msg") })
}

func TestZapAdapter_Methods(t *testing.T) {
	zapLogger := zaptest.NewLogger(t)
	l := NewZapAdapter(zapLogger)

	// Все методы не должны паниковать
	assert.NotPanics(t, func() { l.Info("info message", zap.String("key", "val")) })
	assert.NotPanics(t, func() { l.Error("error message", zap.Error(nil)) })
	assert.NotPanics(t, func() { l.Warn("warn message") })
	assert.NotPanics(t, func() { l.Debug("debug message") })
	// Fatal вызывает os.Exit — не тестируем напрямую
}
