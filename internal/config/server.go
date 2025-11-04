package config

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

// NewServerConfig возвращает конфигурацию сервера с значениями по умолчанию
func NewServerConfig() ServerConfig {
	return ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "metrics-db.json",
		Restore:         true,
	}
}
