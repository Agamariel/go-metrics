package retry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrRetriesExceeded возвращается, если все попытки исчерпаны.
var ErrRetriesExceeded = errors.New("retries exceeded")

// Strategy вычисляет задержку перед очередной попыткой.
type Strategy func(attempt int) time.Duration

// Constant возвращает стратегию с фиксированной задержкой.
func Constant(delay time.Duration) Strategy {
	return func(int) time.Duration { return delay }
}

// Exponential возвращает экспоненциальный бэкофф (1s, 2s, 4s …) с верхним пределом.
func Exponential(cap time.Duration) Strategy {
	return func(attempt int) time.Duration {
		d := time.Duration(1<<uint(attempt)) * time.Second
		if d > cap {
			return cap
		}
		return d
	}
}

// Linear возвращает линейную стратегию с заданными задержками, циклически повторяя их.
func Linear(delays ...time.Duration) Strategy {
	return func(attempt int) time.Duration {
		if len(delays) == 0 {
			return 0
		}
		return delays[attempt%len(delays)]
	}
}

// Do выполняет fn до maxAttempts раз с задержкой strategy.
// Попытки прекращаются, если ctx отменён или fn возвращает не-retriable ошибку.
func Do(ctx context.Context, maxAttempts int, strategy Strategy, isRetriable func(error) bool, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !isRetriable(err) {
			return err
		}
		lastErr = err

		if attempt < maxAttempts-1 { // последнюю задержку не делаем
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(strategy(attempt)):
			}
		}
	}
	return fmt.Errorf("%w: %v", ErrRetriesExceeded, lastErr)
}

// Используется для обертки ошибок, которые можно повторить
var ErrRetriable = errors.New("retriable")

// Retriable ошибки: временные проблемы с соединением, таймауты, 5xx ошибки сервера
func IsHTTPRetriable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrRetriable) {
		return true
	}

	// net.Error
	var ne interface {
		Timeout() bool
		Temporary() bool
	}
	if errors.As(err, &ne) && (ne.Timeout() || ne.Temporary()) {
		return true
	}

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "no such host") ||
		strings.Contains(s, "network is unreachable")
}
