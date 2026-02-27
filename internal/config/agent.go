// Package config содержит структуры конфигурации и функции для их загрузки из окружения и флагов командной строки.
package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

// AgentConfig содержит конфигурацию агента
type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

// agentFileConfig описывает структуру JSON-файла конфигурации агента
type agentFileConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	Key            string `json:"key"`
	RateLimit      int    `json:"rate_limit"`
	CryptoKey      string `json:"crypto_key"`
}

// applyAgentFileConfig применяет значения из JSON-файла к конфигурации агента.
// Поля, для которых флаг был явно задан в командной строке, не перезаписываются.
func applyAgentFileConfig(cfg *AgentConfig, fc agentFileConfig, setFlags map[string]bool) error {
	if !setFlags["a"] && fc.Address != "" {
		cfg.Address = fc.Address
	}
	if !setFlags["r"] && fc.ReportInterval != "" {
		d, err := time.ParseDuration(fc.ReportInterval)
		if err != nil {
			return fmt.Errorf("некорректное значение report_interval %q: %w", fc.ReportInterval, err)
		}
		cfg.ReportInterval = int(d.Seconds())
	}
	if !setFlags["p"] && fc.PollInterval != "" {
		d, err := time.ParseDuration(fc.PollInterval)
		if err != nil {
			return fmt.Errorf("некорректное значение poll_interval %q: %w", fc.PollInterval, err)
		}
		cfg.PollInterval = int(d.Seconds())
	}
	if !setFlags["k"] && fc.Key != "" {
		cfg.Key = fc.Key
	}
	if !setFlags["l"] && fc.RateLimit > 0 {
		cfg.RateLimit = fc.RateLimit
	}
	if !setFlags["crypto-key"] && fc.CryptoKey != "" {
		cfg.CryptoKey = fc.CryptoKey
	}
	return nil
}

// LoadAgentConfig загружает конфигурацию агента из флагов, файла конфигурации и переменных окружения
// Приоритет: переменные окружения > флаги командной строки > файл конфигурации > значения по умолчанию
func LoadAgentConfig() (AgentConfig, error) {
	// Значения по умолчанию
	cfg := AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		RateLimit:      1,
	}

	var configPath string
	flag.StringVar(&configPath, "c", "", "путь к файлу конфигурации JSON")
	flag.StringVar(&configPath, "config", "", "путь к файлу конфигурации JSON")

	// Определяем флаги командной строки
	flag.StringVar(&cfg.Address, "a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "частота отправки метрик на сервер (в секундах)")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "частота опроса метрик из пакета runtime (в секундах)")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для подписи данных")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "количество одновременно исходящих запросов на сервер")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу с публичным ключом для шифрования данных")

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
		var fc agentFileConfig
		if err := loadJSONFile(configPath, &fc); err != nil {
			return cfg, err
		}
		if err := applyAgentFileConfig(&cfg, fc, setFlags); err != nil {
			return cfg, err
		}
	}

	// Парсим переменные окружения (приоритет выше флагов и файла)
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("ошибка при парсинге переменных окружения: %w", err)
	}

	// Валидация
	if err := validateAgentConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// validateAgentConfig проверяет корректность конфигурации агента
func validateAgentConfig(cfg AgentConfig) error {
	validations := []struct {
		value int
		name  string
	}{
		{cfg.ReportInterval, "reportInterval"},
		{cfg.PollInterval, "pollInterval"},
		{cfg.RateLimit, "rateLimit"},
	}

	for _, v := range validations {
		if v.value <= 0 {
			return fmt.Errorf("%s должен быть положительным числом, получено: %d", v.name, v.value)
		}
	}

	return nil
}
