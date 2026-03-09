package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockApp реализует appRunner для тестов.
type mockApp struct {
	runErr      error
	shutdownErr error
	runDelay    time.Duration // задержка перед возвратом из Run (имитирует работающий сервер)
	shutdownCh  chan struct{} // закрывается при вызове Shutdown
}

func newMockApp() *mockApp {
	return &mockApp{
		shutdownCh: make(chan struct{}),
		runDelay:   24 * time.Hour, // блокируется до вызова Shutdown
	}
}

func (m *mockApp) Run() error {
	select {
	case <-m.shutdownCh:
		return m.runErr
	case <-time.After(m.runDelay):
		return m.runErr
	}
}

func (m *mockApp) Shutdown(ctx context.Context) error {
	close(m.shutdownCh)
	return m.shutdownErr
}

// sendSignal посылает сигнал в канал и закрывает его.
func sendSignal(stop chan<- os.Signal) {
	stop <- os.Interrupt
}

//  тесты runWithStop

func TestRunWithStop_StopsOnSignal(t *testing.T) {
	app := newMockApp()
	stop := make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- runWithStop(app, stop)
	}()

	time.Sleep(10 * time.Millisecond)
	sendSignal(stop)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("runWithStop не завершился после сигнала")
	}
}

func TestRunWithStop_ShutdownError(t *testing.T) {
	shutdownErr := errors.New("shutdown failure")
	app := newMockApp()
	app.shutdownErr = shutdownErr

	stop := make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- runWithStop(app, stop)
	}()

	time.Sleep(10 * time.Millisecond)
	sendSignal(stop)

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка при остановке")
		assert.Contains(t, err.Error(), "shutdown failure")
	case <-time.After(3 * time.Second):
		t.Fatal("runWithStop не завершился")
	}
}

func TestRunWithStop_RunError(t *testing.T) {
	startErr := errors.New("port already in use")
	app := &mockApp{
		runErr:     startErr,
		runDelay:   0,
		shutdownCh: make(chan struct{}),
	}

	stop := make(chan os.Signal, 1)
	err := runWithStop(app, stop)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ошибка запуска")
	assert.Contains(t, err.Error(), "port already in use")
}

func TestRunWithStop_RunSuccessNoError(t *testing.T) {
	app := &mockApp{
		runErr:     nil,
		runDelay:   0,
		shutdownCh: make(chan struct{}),
	}

	stop := make(chan os.Signal, 1)
	err := runWithStop(app, stop)
	assert.NoError(t, err)
}

func TestRunWithStop_ShutdownCalledAfterSignal(t *testing.T) {
	app := newMockApp()
	stop := make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- runWithStop(app, stop)
	}()

	time.Sleep(10 * time.Millisecond)
	sendSignal(stop)

	select {
	case <-done:
		// Если Shutdown был вызван, shutdownCh закрыт — Run тоже должен завершиться
		select {
		case <-app.shutdownCh:
			// ok: Shutdown был вызван
		default:
			t.Error("Shutdown не был вызван после получения сигнала")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("runWithStop не завершился")
	}
}

//  тест runApp: проверяем регистрацию сигналов через runApp

func TestRunApp_RunErrorPropagated(t *testing.T) {
	startErr := errors.New("listen tcp: address already in use")
	app := &mockApp{
		runErr:     startErr,
		runDelay:   0,
		shutdownCh: make(chan struct{}),
	}

	err := runApp(app)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ошибка запуска")
}

func TestRunApp_CleanExitWhenRunReturnsNil(t *testing.T) {
	app := &mockApp{
		runErr:     nil,
		runDelay:   0,
		shutdownCh: make(chan struct{}),
	}

	err := runApp(app)
	assert.NoError(t, err)
}
