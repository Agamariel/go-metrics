package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Agamariel/go-metrics/pkg/retry"
)

// Example демонстрирует базовое использование пакета retry.
func Example() {
	ctx := context.Background()
	attempts := 0

	// Функция, которая успешно выполняется с третьей попытки
	fn := func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("временная ошибка: %w", retry.ErrRetriable)
		}
		return nil
	}

	err := retry.Do(ctx, 5, retry.Constant(10*time.Millisecond), retry.IsHTTPRetriable, fn)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
	} else {
		fmt.Printf("Успех после %d попыток\n", attempts)
	}

	// Output:
	// Успех после 3 попыток
}

// ExampleDo демонстрирует использование функции Do с разными стратегиями.
func ExampleDo() {
	ctx := context.Background()
	counter := 0

	// Функция, всегда завершающаяся ошибкой
	failingFn := func() error {
		counter++
		return fmt.Errorf("ошибка подключения: %w", retry.ErrRetriable)
	}

	err := retry.Do(ctx, 3, retry.Constant(10*time.Millisecond), retry.IsHTTPRetriable, failingFn)
	if errors.Is(err, retry.ErrRetriesExceeded) {
		fmt.Printf("Все %d попытки исчерпаны\n", counter)
	}

	// Output:
	// Все 3 попытки исчерпаны
}

// ExampleDo_nonRetriable демонстрирует поведение при не-retriable ошибке.
func ExampleDo_nonRetriable() {
	ctx := context.Background()
	attempts := 0

	// Функция, возвращающая не-retriable ошибку
	fn := func() error {
		attempts++
		return errors.New("критическая ошибка") // не содержит ErrRetriable
	}

	err := retry.Do(ctx, 5, retry.Constant(10*time.Millisecond), retry.IsHTTPRetriable, fn)
	fmt.Printf("Попыток: %d, Ошибка: %v\n", attempts, err)

	// Output:
	// Попыток: 1, Ошибка: критическая ошибка
}

// ExampleConstant демонстрирует стратегию с постоянной задержкой.
func ExampleConstant() {
	strategy := retry.Constant(5 * time.Second)

	// Задержка одинакова для всех попыток
	fmt.Printf("Попытка 0: %v\n", strategy(0))
	fmt.Printf("Попытка 1: %v\n", strategy(1))
	fmt.Printf("Попытка 2: %v\n", strategy(2))

	// Output:
	// Попытка 0: 5s
	// Попытка 1: 5s
	// Попытка 2: 5s
}

// ExampleExponential демонстрирует стратегию экспоненциального бэкоффа.
func ExampleExponential() {
	strategy := retry.Exponential(30 * time.Second)

	// Задержка удваивается с каждой попыткой
	fmt.Printf("Попытка 0: %v\n", strategy(0))
	fmt.Printf("Попытка 1: %v\n", strategy(1))
	fmt.Printf("Попытка 2: %v\n", strategy(2))
	fmt.Printf("Попытка 3: %v\n", strategy(3))
	fmt.Printf("Попытка 4: %v\n", strategy(4))
	fmt.Printf("Попытка 5: %v\n", strategy(5)) // ограничено cap

	// Output:
	// Попытка 0: 1s
	// Попытка 1: 2s
	// Попытка 2: 4s
	// Попытка 3: 8s
	// Попытка 4: 16s
	// Попытка 5: 30s
}

// ExampleLinear демонстрирует стратегию с заданной последовательностью задержек.
func ExampleLinear() {
	strategy := retry.Linear(1*time.Second, 3*time.Second, 5*time.Second)

	// Задержки циклически повторяются
	fmt.Printf("Попытка 0: %v\n", strategy(0))
	fmt.Printf("Попытка 1: %v\n", strategy(1))
	fmt.Printf("Попытка 2: %v\n", strategy(2))
	fmt.Printf("Попытка 3: %v\n", strategy(3)) // цикл повторяется
	fmt.Printf("Попытка 4: %v\n", strategy(4))

	// Output:
	// Попытка 0: 1s
	// Попытка 1: 3s
	// Попытка 2: 5s
	// Попытка 3: 1s
	// Попытка 4: 3s
}

// ExampleIsHTTPRetriable демонстрирует определение retriable-ошибок.
func ExampleIsHTTPRetriable() {
	// Retriable ошибки
	err1 := fmt.Errorf("connection refused")
	err2 := fmt.Errorf("server error: %w", retry.ErrRetriable)

	// Не-retriable ошибка
	err3 := errors.New("invalid request")

	fmt.Printf("connection refused: %v\n", retry.IsHTTPRetriable(err1))
	fmt.Printf("server error: %v\n", retry.IsHTTPRetriable(err2))
	fmt.Printf("invalid request: %v\n", retry.IsHTTPRetriable(err3))

	// Output:
	// connection refused: true
	// server error: true
	// invalid request: false
}

// ExampleDo_withContext демонстрирует отмену retry через context.
func ExampleDo_withContext() {
	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	fn := func() error {
		attempts++
		return fmt.Errorf("ошибка: %w", retry.ErrRetriable)
	}

	// Стратегия с большой задержкой, которая будет прервана таймаутом
	err := retry.Do(ctx, 100, retry.Constant(100*time.Millisecond), retry.IsHTTPRetriable, fn)

	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Printf("Прервано по таймауту после %d попыток\n", attempts)
	}

	// Output:
	// Прервано по таймауту после 1 попыток
}
