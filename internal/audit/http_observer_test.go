package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewHTTPObserver(t *testing.T) {
	url := "http://example.com/audit"
	observer := NewHTTPObserver(url)

	if observer.url != url {
		t.Errorf("Expected URL %s, got %s", url, observer.url)
	}

	if observer.client == nil {
		t.Error("Client should not be nil")
	}

	if observer.client.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", observer.client.Timeout)
	}
}

func TestHTTPObserver_Notify_Success(t *testing.T) {
	var mu sync.Mutex
	receivedEvents := []Event{}

	// Создаем тестовый HTTP сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var event Event
		if err := json.Unmarshal(body, &event); err != nil {
			t.Errorf("Failed to unmarshal event: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"Alloc", "TotalAlloc"},
		IPAddress: "192.168.0.42",
	}

	ctx := context.Background()
	if err := observer.Notify(ctx, event); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(receivedEvents))
	}

	if receivedEvents[0].IPAddress != event.IPAddress {
		t.Errorf("Expected IP %s, got %s", event.IPAddress, receivedEvents[0].IPAddress)
	}

	if len(receivedEvents[0].Metrics) != len(event.Metrics) {
		t.Errorf("Expected %d metrics, got %d", len(event.Metrics), len(receivedEvents[0].Metrics))
	}
}

func TestHTTPObserver_Notify_ServerError(t *testing.T) {
	attempts := 0
	var mu sync.Mutex

	// Сервер всегда возвращает 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	err := observer.Notify(ctx, event)

	if err == nil {
		t.Error("Expected error for 500 response")
	}

	mu.Lock()
	defer mu.Unlock()

	// Проверяем, что были попытки retry (должно быть 3 попытки)
	if attempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", attempts)
	}
}

func TestHTTPObserver_Notify_ClientError(t *testing.T) {
	attempts := 0
	var mu sync.Mutex

	// Сервер возвращает 400
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		mu.Unlock()

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	ctx := context.Background()
	err := observer.Notify(ctx, event)

	if err == nil {
		t.Error("Expected error for 400 response")
	}

	mu.Lock()
	defer mu.Unlock()

	// 4xx ошибки не должны вызывать retry, только 1 попытка
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry for 4xx), got %d", attempts)
	}
}

func TestHTTPObserver_Notify_MultipleEvents(t *testing.T) {
	var mu sync.Mutex
	receivedEvents := []Event{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var event Event
		json.Unmarshal(body, &event)

		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)
	ctx := context.Background()

	// Отправляем несколько событий
	for i := 0; i < 3; i++ {
		event := Event{
			Timestamp: time.Now().Unix() + int64(i),
			Metrics:   []string{"metric" + string(rune('1'+i))},
			IPAddress: "192.168.0." + string(rune('1'+i)),
		}

		if err := observer.Notify(ctx, event); err != nil {
			t.Fatalf("Notify %d failed: %v", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(receivedEvents))
	}
}

func TestHTTPObserver_Notify_ContextCancellation(t *testing.T) {
	// Сервер с задержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	observer := NewHTTPObserver(server.URL)

	event := Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "192.168.0.1",
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := observer.Notify(ctx, event)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}
