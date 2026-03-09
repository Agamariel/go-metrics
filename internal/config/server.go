package config

import (
	"flag"
	"fmt"
	"time"

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
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
	CryptoKey       string `env:"CRYPTO_KEY"`
}

// serverFileConfig описывает структуру JSON-файла конфигурации сервера
type serverFileConfig struct {
	Address         string `json:"address"`
	Restore         *bool  `json:"restore"`
	StoreInterval   string `json:"store_interval"`
	StoreFile       string `json:"store_file"`
	DatabaseDSN     string `json:"database_dsn"`
	Key             string `json:"key"`
	ShutdownTimeout string `json:"shutdown_timeout"`
	AuditFile       string `json:"audit_file"`
	AuditURL        string `json:"audit_url"`
	CryptoKey       string `json:"crypto_key"`
}

// applyServerFileConfig применяет значения из JSON-файла к конфигурации сервера.
// Поля, для которых флаг был явно задан в командной строке, не перезаписываются.
func applyServerFileConfig(cfg *ServerConfig, fc serverFileConfig, setFlags map[string]bool) error {
	if !setFlags["a"] && fc.Address != "" {
		cfg.Address = fc.Address
	}
	if !setFlags["r"] && fc.Restore != nil {
		cfg.Restore = *fc.Restore
	}
	if !setFlags["i"] && fc.StoreInterval != "" {
		d, err := time.ParseDuration(fc.StoreInterval)
		if err != nil {
			return fmt.Errorf("некорректное значение store_interval %q: %w", fc.StoreInterval, err)
		}
		cfg.StoreInterval = int(d.Seconds())
	}
	if !setFlags["f"] && fc.StoreFile != "" {
		cfg.FileStoragePath = fc.StoreFile
	}
	if !setFlags["d"] && fc.DatabaseDSN != "" {
		cfg.DatabaseDSN = fc.DatabaseDSN
	}
	if !setFlags["k"] && fc.Key != "" {
		cfg.Key = fc.Key
	}
	if !setFlags["t"] && fc.ShutdownTimeout != "" {
		d, err := time.ParseDuration(fc.ShutdownTimeout)
		if err != nil {
			return fmt.Errorf("некорректное значение shutdown_timeout %q: %w", fc.ShutdownTimeout, err)
		}
		cfg.ShutdownTimeout = int(d.Seconds())
	}
	if !setFlags["audit-file"] && fc.AuditFile != "" {
		cfg.AuditFile = fc.AuditFile
	}
	if !setFlags["audit-url"] && fc.AuditURL != "" {
		cfg.AuditURL = fc.AuditURL
	}
	if !setFlags["crypto-key"] && fc.CryptoKey != "" {
		cfg.CryptoKey = fc.CryptoKey
	}
	return nil
}

// LoadServerConfig загружает конфигурацию сервера из флагов, файла конфигурации и переменных окружения
// Приоритет: переменные окружения > флаги командной строки > файл конфигурации > значения по умолчанию
func LoadServerConfig() (ServerConfig, error) {
	// Значения по умолчанию
	cfg := ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "metrics-db.json",
		Restore:         true,
		ShutdownTimeout: 5,
	}

	var configPath string
	flag.StringVar(&configPath, "c", "", "путь к файлу конфигурации JSON")
	flag.StringVar(&configPath, "config", "", "путь к файлу конфигурации JSON")

	// Определяем флаги командной строки
	flag.StringVar(&cfg.Address, "a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	flag.IntVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "интервал сохранения метрик в секундах (0 = синхронное сохранение)")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "путь к файлу для сохранения метрик")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "загружать ли ранее сохранённые метрики при старте")
	flag.IntVar(&cfg.ShutdownTimeout, "t", cfg.ShutdownTimeout, "таймаут graceful shutdown в секундах")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "строка подключения к базе данных")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для подписи данных")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу для сохранения логов аудита")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL для отправки логов аудита")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу с приватным ключом для расшифровки данных")

	flag.Parse()

	// Проверяем, что не было передано лишних аргументов
	if flag.NArg() > 0 {
		return cfg, fmt.Errorf("неизвестные аргументы: %v", flag.Args())
	}

	// Запоминаем явно заданные флаги
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	// Загружаем JSON-файл конфигурации (приоритет ниже флагов)
	configPath = resolveConfigPath(configPath)
	if configPath != "" {
		var fc serverFileConfig
		if err := loadJSONFile(configPath, &fc); err != nil {
			return cfg, err
		}
		if err := applyServerFileConfig(&cfg, fc, setFlags); err != nil {
			return cfg, err
		}
	}

	// Парсим переменные окружения (приоритет выше флагов и файла)
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
