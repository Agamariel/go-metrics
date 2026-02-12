// Package retry предоставляет механизм повторных попыток с настраиваемыми стратегиями задержки.
//
// Пакет полезен для обработки временных сбоев при работе с внешними сервисами:
// базами данных, HTTP API, очередями сообщений и т.д.
//
// # Стратегии задержки
//
// Пакет предоставляет три встроенные стратегии:
//   - Constant: фиксированная задержка между попытками
//   - Exponential: экспоненциальный бэкофф (1s, 2s, 4s, ...)
//   - Linear: заданная последовательность задержек
//
// # Пример использования
//
//	err := retry.Do(ctx, 3, retry.Exponential(30*time.Second), retry.IsHTTPRetriable, func() error {
//	    return httpClient.Do(req)
//	})
//	if errors.Is(err, retry.ErrRetriesExceeded) {
//	    log.Printf("Все попытки исчерпаны")
//	}
package retry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrRetriesExceeded возвращается, когда все попытки исчерпаны
// и операция так и не была выполнена успешно.
//
// Ошибка оборачивает последнюю полученную ошибку, которую можно извлечь
// с помощью errors.Unwrap.
var ErrRetriesExceeded = errors.New("retries exceeded")

// Strategy определяет функцию расчёта задержки перед следующей попыткой.
//
// Параметр attempt — номер текущей попытки (начиная с 0).
// Возвращает продолжительность задержки перед следующей попыткой.
type Strategy func(attempt int) time.Duration

// Constant создаёт стратегию с фиксированной задержкой между попытками.
//
// Пример: Constant(5*time.Second) — задержка 5 секунд между каждой попыткой.
func Constant(delay time.Duration) Strategy {
	return func(int) time.Duration { return delay }
}

// Exponential создаёт стратегию экспоненциального бэкоффа.
//
// Задержки растут экспоненциально: 1s, 2s, 4s, 8s, ...
// Параметр cap ограничивает максимальную задержку.
//
// Пример: Exponential(30*time.Second) — бэкофф до 30 секунд максимум.
func Exponential(cap time.Duration) Strategy {
	return func(attempt int) time.Duration {
		d := time.Duration(1<<uint(attempt)) * time.Second
		if d > cap {
			return cap
		}
		return d
	}
}

// Linear создаёт стратегию с заданной последовательностью задержек.
//
// При исчерпании последовательности она повторяется циклически.
//
// Пример: Linear(1*time.Second, 3*time.Second, 5*time.Second)
// даст задержки: 1s, 3s, 5s, 1s, 3s, 5s, ...
func Linear(delays ...time.Duration) Strategy {
	return func(attempt int) time.Duration {
		if len(delays) == 0 {
			return 0
		}
		return delays[attempt%len(delays)]
	}
}

// Do выполняет функцию fn с повторными попытками при ошибках.
//
// Параметры:
//   - ctx: контекст для отмены операции
//   - maxAttempts: максимальное количество попыток
//   - strategy: стратегия расчёта задержки между попытками
//   - isRetriable: функция определения, можно ли повторить попытку для данной ошибки
//   - fn: функция для выполнения
//
// Возвращает nil при успешном выполнении, ErrRetriesExceeded если все попытки
// исчерпаны, или исходную ошибку если она не подлежит повторной попытке.
//
// Функция прекращает попытки если:
//   - fn вернула nil (успех)
//   - isRetriable(err) вернула false
//   - контекст был отменён
//   - достигнуто maxAttempts
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

// ErrRetriable — маркерная ошибка для обозначения повторяемых ошибок.
//
// Используйте fmt.Errorf("операция не удалась: %w", retry.ErrRetriable)
// для создания ошибок, которые будут определяться как повторяемые.
var ErrRetriable = errors.New("retriable")

// IsHTTPRetriable определяет, является ли ошибка временной и подлежит повторной попытке.
//
// Возвращает true для следующих типов ошибок:
//   - Ошибки, содержащие ErrRetriable
//   - Сетевые таймауты и временные ошибки (net.Error)
//   - Ошибки соединения: connection refused, connection reset
//   - Ошибки DNS: no such host
//   - Ошибки сети: network is unreachable
//
// Пример использования:
//
//	err := retry.Do(ctx, 3, retry.Constant(time.Second), retry.IsHTTPRetriable, func() error {
//	    resp, err := http.Get(url)
//	    if err != nil {
//	        return err
//	    }
//	    if resp.StatusCode >= 500 {
//	        return fmt.Errorf("server error: %d: %w", resp.StatusCode, retry.ErrRetriable)
//	    }
//	    return nil
//	})
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
