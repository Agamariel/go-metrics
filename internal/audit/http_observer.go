package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/pkg/retry"
)

// HTTPObserver отправляет события аудита на удаленный сервер по HTTP.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создает новый HTTPObserver.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие аудита на удаленный сервер.
func (h *HTTPObserver) Notify(ctx context.Context, event Event) error {
	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации события: %w", err)
	}

	// Функция отправки HTTP запроса
	sendFunc := func() error {
		// Создаем новый запрос для каждой попытки
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("ошибка создания HTTP запроса: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := h.client.Do(req)
		if err != nil {
			return fmt.Errorf("ошибка отправки HTTP запроса: %w", err)
		}
		defer resp.Body.Close()

		io.Copy(io.Discard, resp.Body)

		if resp.StatusCode >= 500 {
			return fmt.Errorf("%w: сервер вернул ошибку: %d %s", retry.ErrRetriable, resp.StatusCode, resp.Status)
		} else if resp.StatusCode >= 400 {
			return fmt.Errorf("клиентская ошибка: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	}

	// Стратегия retry: 3 попытки с задержками 1s, 2s, 3s
	err = retry.Do(
		ctx,
		3,
		retry.Linear(1*time.Second, 2*time.Second, 3*time.Second),
		func(err error) bool {
			if err == nil {
				return false
			}
			return retry.IsHTTPRetriable(err)
		},
		sendFunc,
	)

	if err != nil {
		return fmt.Errorf("не удалось отправить событие аудита по HTTP после retry: %w", err)
	}

	return nil
}
