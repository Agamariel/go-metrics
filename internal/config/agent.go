package config

// AgentConfig содержит конфигурацию агента
type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

// NewAgentConfig возвращает конфигурацию агента с значениями по умолчанию
func NewAgentConfig() AgentConfig {
	return AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}
}
