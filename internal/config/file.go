package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// resolveConfigPath возвращает путь к файлу конфигурации.
// Переменная окружения CONFIG имеет приоритет над значением флага.
func resolveConfigPath(flagValue string) string {
	if envVal := os.Getenv("CONFIG"); envVal != "" {
		return envVal
	}
	return flagValue
}

// loadJSONFile читает JSON-файл по заданному пути и десериализует его в dest.
func loadJSONFile(path string, dest interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла конфигурации %q: %w", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("ошибка парсинга файла конфигурации %q: %w", path, err)
	}
	return nil
}
