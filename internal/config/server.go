package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	ShutdownTimeout int    `env:"SHUTDOWN_TIMEOUT"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
}

// LoadServerConfig загружает конфигурацию сервера из флагов и переменных окружения
// Приоритет: переменные окружения > флаги командной строки > значения по умолчанию
func LoadServerConfig() (ServerConfig, error) {
	// Значения по умолчанию
	cfg := ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "metrics-db.json",
		Restore:         true,
		ShutdownTimeout: 5,
	}

	// Определяем флаги командной строки
	flag.StringVar(&cfg.Address, "a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "интервал сохранения метрик в секундах (0 = синхронное сохранение)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь к файлу для сохранения метрик")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "загружать ли ранее сохранённые метрики при старте")
	flag.IntVar(&cfg.ShutdownTimeout, "t", cfg.ShutdownTimeout, "таймаут graceful shutdown в секундах")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для подписи данных")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		return cfg, fmt.Errorf("неизвестные аргументы: %v", flag.Args())
	}

	// Парсим переменные окружения (приоритет выше флагов)
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("ошибка при парсинге переменных окружения: %w", err)
	}

	// Валидация
	if cfg.ShutdownTimeout <= 0 {
		return cfg, fmt.Errorf("таймаут shutdown должен быть положительным числом, получено: %d", cfg.ShutdownTimeout)
	}
	if cfg.StoreInterval <= 0 {
		return cfg, fmt.Errorf("интервал сохранения метрик должен быть положительным числом, получено: %d", cfg.StoreInterval)
	}

	return cfg, nil
}
