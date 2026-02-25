// Package config содержит структуры конфигурации и функции для их загрузки из окружения и флагов командной строки.
package config

import (
	"flag"
	"fmt"

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

// LoadAgentConfig загружает конфигурацию агента из флагов и переменных окружения
// Приоритет: переменные окружения > флаги командной строки > значения по умолчанию
func LoadAgentConfig() (AgentConfig, error) {
	// Значения по умолчанию
	cfg := AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		RateLimit:      1,
	}

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

	// Парсим переменные окружения (приоритет выше флагов)
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
