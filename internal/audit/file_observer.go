package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Agamariel/go-metrics/pkg/retry"
)

// FileObserver записывает события аудита в файл.
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileObserver создает новый FileObserver.
func NewFileObserver(filePath string) (*FileObserver, error) {
	if filePath == "" {
		return nil, fmt.Errorf("filePath не может быть пустым")
	}

	// Проверяем, что можем создать/открыть файл
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w", filePath, err)
	}
	file.Close()

	return &FileObserver{
		filePath: filePath,
	}, nil
}

// Notify записывает событие аудита в файл.
func (f *FileObserver) Notify(ctx context.Context, event Event) error {
	// Сериализуем событие в JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации события: %w", err)
	}

	data = append(data, '\n')

	// Используем retry механизм для записи
	writeFunc := func() error {
		f.mu.Lock()
		defer f.mu.Unlock()

		file, err := os.OpenFile(f.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("не удалось открыть файл: %w", err)
		}
		defer file.Close()

		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("ошибка записи в файл: %w", err)
		}

		return nil
	}

	// Стратегия retry: 3 попытки с задержками 1s, 2s, 3s
	err = retry.Do(
		ctx,
		3,
		retry.Linear(1*time.Second, 2*time.Second, 3*time.Second),
		func(err error) bool {
			return err != nil
		},
		writeFunc,
	)

	if err != nil {
		return fmt.Errorf("не удалось записать событие аудита в файл после retry: %w", err)
	}

	return nil
}
