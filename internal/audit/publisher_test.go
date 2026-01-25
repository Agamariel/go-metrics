package audit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockLogger для тестирования
type MockLogger struct {
	mu       sync.Mutex
	messages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		messages: make([]string, 0),
	}
}

func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

// MockObserver для тестирования
type MockObserver struct {
	mu     sync.Mutex
	events []Event
	err    error
	delay  time.Duration
}

func NewMockObserver() *MockObserver {
	return &MockObserver{
		events: make([]Event, 0),
	}
}

func (m *MockObserver) Notify(ctx context.Context, event Event) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}

	m.events = append(m.events, event)
	return nil
}

func (m *MockObserver) GetEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.events...)
}

func TestPublisher_Attach(t *testing.T) {
	logger := NewMockLogger()
	publisher := NewPublisher(logger)

	obs1 := NewMockObserver()
	obs2 := NewMockObserver()

	publisher.Attach(obs1)
	publisher.Attach(obs2)

	if len(publisher.observers) != 2 {
		t.Errorf("Expected 2 observers, got %d", len(publisher.observers))
	}
}

func TestPublisher_NotifyAll(t *testing.T) {
	logger := NewMockLogger()
	publisher := NewPublisher(logger)

	obs1 := NewMockObserver()
	obs2 := NewMockObserver()

	publisher.Attach(obs1)
	publisher.Attach(obs2)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1", "metric2"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	publisher.NotifyAll(ctx, event)

	// Даем время горутинам завершиться
	publisher.Close()

	events1 := obs1.GetEvents()
	events2 := obs2.GetEvents()

	if len(events1) != 1 {
		t.Errorf("Observer 1: expected 1 event, got %d", len(events1))
	}

	if len(events2) != 1 {
		t.Errorf("Observer 2: expected 1 event, got %d", len(events2))
	}

	if len(events1) > 0 && events1[0].IPAddress != event.IPAddress {
		t.Errorf("Observer 1: expected IP %s, got %s", event.IPAddress, events1[0].IPAddress)
	}
}

func TestPublisher_NotifyAll_WithError(t *testing.T) {
	logger := NewMockLogger()
	publisher := NewPublisher(logger)

	obs := NewMockObserver()
	obs.err = errors.New("test error")

	publisher.Attach(obs)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	publisher.NotifyAll(ctx, event)

	// Даем время горутинам завершиться
	publisher.Close()

	// Проверяем, что ошибка была залогирована
	logger.mu.Lock()
	hasError := false
	for _, msg := range logger.messages {
		if msg == "Ошибка при отправке события аудита" {
			hasError = true
			break
		}
	}
	logger.mu.Unlock()

	if !hasError {
		t.Error("Expected error to be logged")
	}
}

func TestPublisher_NotifyAll_NoObservers(t *testing.T) {
	logger := NewMockLogger()
	publisher := NewPublisher(logger)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	// Не должно паниковать при отсутствии наблюдателей
	publisher.NotifyAll(ctx, event)

	publisher.Close()
}

func TestPublisher_Close(t *testing.T) {
	logger := NewMockLogger()
	publisher := NewPublisher(logger)

	obs := NewMockObserver()
	obs.delay = 100 * time.Millisecond // Добавляем задержку

	publisher.Attach(obs)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	publisher.NotifyAll(ctx, event)

	// Close должен дождаться завершения всех операций
	start := time.Now()
	publisher.Close()
	duration := time.Since(start)

	if duration < 100*time.Millisecond {
		t.Error("Close should wait for all operations to complete")
	}

	events := obs.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}
