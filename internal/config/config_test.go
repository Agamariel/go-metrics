package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyAgentFileConfig_AllFields(t *testing.T) {
	cfg := AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		RateLimit:      1,
	}

	fc := agentFileConfig{
		Address:        "localhost:9090",
		ReportInterval: "5s",
		PollInterval:   "3s",
		Key:            "secret-key",
		RateLimit:      4,
		CryptoKey:      "/path/to/key.pem",
	}

	noFlags := map[string]bool{}
	err := applyAgentFileConfig(&cfg, fc, noFlags)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Address)
	assert.Equal(t, 5, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, "secret-key", cfg.Key)
	assert.Equal(t, 4, cfg.RateLimit)
	assert.Equal(t, "/path/to/key.pem", cfg.CryptoKey)
}

func TestApplyAgentFileConfig_FlagsOverride(t *testing.T) {
	cfg := AgentConfig{
		Address:        "flag-addr:1111",
		ReportInterval: 99,
		PollInterval:   88,
		Key:            "flag-key",
		RateLimit:      7,
		CryptoKey:      "flag-crypto",
	}

	fc := agentFileConfig{
		Address:        "file-addr:2222",
		ReportInterval: "20s",
		PollInterval:   "15s",
		Key:            "file-key",
		RateLimit:      3,
		CryptoKey:      "file-crypto",
	}

	allFlags := map[string]bool{
		"a":          true,
		"r":          true,
		"p":          true,
		"k":          true,
		"l":          true,
		"crypto-key": true,
	}
	err := applyAgentFileConfig(&cfg, fc, allFlags)
	require.NoError(t, err)

	assert.Equal(t, "flag-addr:1111", cfg.Address)
	assert.Equal(t, 99, cfg.ReportInterval)
	assert.Equal(t, 88, cfg.PollInterval)
	assert.Equal(t, "flag-key", cfg.Key)
	assert.Equal(t, 7, cfg.RateLimit)
	assert.Equal(t, "flag-crypto", cfg.CryptoKey)
}

func TestApplyAgentFileConfig_EmptyValuesSkipped(t *testing.T) {
	cfg := AgentConfig{
		Address:        "keep",
		ReportInterval: 10,
		PollInterval:   2,
		Key:            "keep-key",
		RateLimit:      1,
		CryptoKey:      "keep-crypto",
	}

	fc := agentFileConfig{}

	err := applyAgentFileConfig(&cfg, fc, map[string]bool{})
	require.NoError(t, err)

	assert.Equal(t, "keep", cfg.Address)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, "keep-key", cfg.Key)
	assert.Equal(t, 1, cfg.RateLimit)
	assert.Equal(t, "keep-crypto", cfg.CryptoKey)
}

func TestApplyAgentFileConfig_InvalidDuration(t *testing.T) {
	cfg := AgentConfig{}

	tests := []struct {
		name string
		fc   agentFileConfig
	}{
		{
			name: "invalid report_interval",
			fc:   agentFileConfig{ReportInterval: "invalid"},
		},
		{
			name: "invalid poll_interval",
			fc:   agentFileConfig{PollInterval: "not-a-duration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyAgentFileConfig(&cfg, tt.fc, map[string]bool{})
			assert.Error(t, err)
		})
	}
}

func TestApplyServerFileConfig_AllFields(t *testing.T) {
	cfg := ServerConfig{
		Address:         "localhost:8080",
		StoreInterval:   300,
		FileStoragePath: "localhost.json",
		Restore:         true,
		ShutdownTimeout: 5,
	}

	restoreVal := false
	fc := serverFileConfig{
		Address:         "localhost:9090",
		Restore:         &restoreVal,
		StoreInterval:   "60s",
		StoreFile:       "/data/metrics.json",
		DatabaseDSN:     "postgres://user:pass@localhost/db",
		Key:             "hmac-key",
		ShutdownTimeout: "30s",
		AuditFile:       "/var/log/audit.log",
		AuditURL:        "http://audit.example.com",
		CryptoKey:       "/etc/keys/private.pem",
	}

	noFlags := map[string]bool{}
	err := applyServerFileConfig(&cfg, fc, noFlags)
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Address)
	assert.Equal(t, false, cfg.Restore)
	assert.Equal(t, 60, cfg.StoreInterval)
	assert.Equal(t, "/data/metrics.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "hmac-key", cfg.Key)
	assert.Equal(t, 30, cfg.ShutdownTimeout)
	assert.Equal(t, "/var/log/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit.example.com", cfg.AuditURL)
	assert.Equal(t, "/etc/keys/private.pem", cfg.CryptoKey)
}

func TestApplyServerFileConfig_FlagsOverride(t *testing.T) {
	cfg := ServerConfig{
		Address:         "flag-addr:1111",
		StoreInterval:   99,
		FileStoragePath: "flag-file.json",
		Restore:         true,
		ShutdownTimeout: 15,
		DatabaseDSN:     "flag-dsn",
		Key:             "flag-key",
		AuditFile:       "flag-audit.log",
		AuditURL:        "http://flag-audit",
		CryptoKey:       "flag-crypto",
	}

	restoreVal := false
	fc := serverFileConfig{
		Address:         "file-addr:2222",
		Restore:         &restoreVal,
		StoreInterval:   "120s",
		StoreFile:       "file-store.json",
		DatabaseDSN:     "file-dsn",
		Key:             "file-key",
		ShutdownTimeout: "60s",
		AuditFile:       "file-audit.log",
		AuditURL:        "http://file-audit",
		CryptoKey:       "file-crypto",
	}

	allFlags := map[string]bool{
		"a":          true,
		"r":          true,
		"i":          true,
		"f":          true,
		"d":          true,
		"k":          true,
		"t":          true,
		"audit-file": true,
		"audit-url":  true,
		"crypto-key": true,
	}
	err := applyServerFileConfig(&cfg, fc, allFlags)
	require.NoError(t, err)

	assert.Equal(t, "flag-addr:1111", cfg.Address)
	assert.Equal(t, true, cfg.Restore)
	assert.Equal(t, 99, cfg.StoreInterval)
	assert.Equal(t, "flag-file.json", cfg.FileStoragePath)
	assert.Equal(t, "flag-dsn", cfg.DatabaseDSN)
	assert.Equal(t, "flag-key", cfg.Key)
	assert.Equal(t, 15, cfg.ShutdownTimeout)
	assert.Equal(t, "flag-audit.log", cfg.AuditFile)
	assert.Equal(t, "http://flag-audit", cfg.AuditURL)
	assert.Equal(t, "flag-crypto", cfg.CryptoKey)
}

func TestApplyServerFileConfig_InvalidDuration(t *testing.T) {
	cfg := ServerConfig{}

	tests := []struct {
		name string
		fc   serverFileConfig
	}{
		{
			name: "invalid store_interval",
			fc:   serverFileConfig{StoreInterval: "bad"},
		},
		{
			name: "invalid shutdown_timeout",
			fc:   serverFileConfig{ShutdownTimeout: "not-a-duration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyServerFileConfig(&cfg, tt.fc, map[string]bool{})
			assert.Error(t, err)
		})
	}
}

func TestApplyServerFileConfig_PartialFields(t *testing.T) {
	cfg := ServerConfig{
		Address:         "keep-addr",
		StoreInterval:   300,
		FileStoragePath: "keep.json",
		Restore:         true,
		ShutdownTimeout: 5,
	}

	fc := serverFileConfig{
		Key:       "only-key",
		AuditFile: "only-audit.log",
	}

	err := applyServerFileConfig(&cfg, fc, map[string]bool{})
	require.NoError(t, err)

	assert.Equal(t, "keep-addr", cfg.Address)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "keep.json", cfg.FileStoragePath)
	assert.Equal(t, true, cfg.Restore)
	assert.Equal(t, 5, cfg.ShutdownTimeout)
	assert.Equal(t, "only-key", cfg.Key)
	assert.Equal(t, "only-audit.log", cfg.AuditFile)
	assert.Equal(t, "", cfg.AuditURL)
	assert.Equal(t, "", cfg.CryptoKey)
}

func TestLoadJSONFile_AgentAllFields(t *testing.T) {
	content := map[string]interface{}{
		"address":         "json-host:3000",
		"report_interval": "15s",
		"poll_interval":   "5s",
		"key":             "json-key",
		"rate_limit":      8,
		"crypto_key":      "/json/key.pem",
	}

	tmpFile := writeJSONTempFile(t, content)

	var fc agentFileConfig
	err := loadJSONFile(tmpFile, &fc)
	require.NoError(t, err)

	assert.Equal(t, "json-host:3000", fc.Address)
	assert.Equal(t, "15s", fc.ReportInterval)
	assert.Equal(t, "5s", fc.PollInterval)
	assert.Equal(t, "json-key", fc.Key)
	assert.Equal(t, 8, fc.RateLimit)
	assert.Equal(t, "/json/key.pem", fc.CryptoKey)
}

func TestLoadJSONFile_ServerAllFields(t *testing.T) {
	content := map[string]interface{}{
		"address":          "json-host:4000",
		"restore":          false,
		"store_interval":   "120s",
		"store_file":       "/json/store.json",
		"database_dsn":     "postgres://json@localhost/db",
		"key":              "json-hmac",
		"shutdown_timeout": "10s",
		"audit_file":       "/json/audit.log",
		"audit_url":        "http://json-audit.example.com",
		"crypto_key":       "/json/private.pem",
	}

	tmpFile := writeJSONTempFile(t, content)

	var fc serverFileConfig
	err := loadJSONFile(tmpFile, &fc)
	require.NoError(t, err)

	assert.Equal(t, "json-host:4000", fc.Address)
	require.NotNil(t, fc.Restore)
	assert.Equal(t, false, *fc.Restore)
	assert.Equal(t, "120s", fc.StoreInterval)
	assert.Equal(t, "/json/store.json", fc.StoreFile)
	assert.Equal(t, "postgres://json@localhost/db", fc.DatabaseDSN)
	assert.Equal(t, "json-hmac", fc.Key)
	assert.Equal(t, "10s", fc.ShutdownTimeout)
	assert.Equal(t, "/json/audit.log", fc.AuditFile)
	assert.Equal(t, "http://json-audit.example.com", fc.AuditURL)
	assert.Equal(t, "/json/private.pem", fc.CryptoKey)
}

func TestValidateAgentConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AgentConfig
		wantErr bool
	}{
		{
			name: "valid",
			cfg: AgentConfig{
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      1,
			},
			wantErr: false,
		},
		{
			name: "zero report interval",
			cfg: AgentConfig{
				ReportInterval: 0,
				PollInterval:   2,
				RateLimit:      1,
			},
			wantErr: true,
		},
		{
			name: "negative poll interval",
			cfg: AgentConfig{
				ReportInterval: 10,
				PollInterval:   -1,
				RateLimit:      1,
			},
			wantErr: true,
		},
		{
			name: "zero rate limit",
			cfg: AgentConfig{
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentConfig(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// resolveConfigPath tests

func TestResolveConfigPath_EnvOverridesFlag(t *testing.T) {
	t.Setenv("CONFIG", "/from/env.json")
	result := resolveConfigPath("/from/flag.json")
	assert.Equal(t, "/from/env.json", result)
}

func TestResolveConfigPath_FlagWhenNoEnv(t *testing.T) {
	t.Setenv("CONFIG", "")
	result := resolveConfigPath("/from/flag.json")
	assert.Equal(t, "/from/flag.json", result)
}

func TestResolveConfigPath_EmptyWhenBothEmpty(t *testing.T) {
	t.Setenv("CONFIG", "")
	result := resolveConfigPath("")
	assert.Equal(t, "", result)
}

// loadJSONFile error paths

func TestLoadJSONFile_FileNotFound(t *testing.T) {
	var fc agentFileConfig
	err := loadJSONFile("/nonexistent/path/config.json", &fc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ошибка чтения файла конфигурации")
}

func TestLoadJSONFile_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json}"), 0644))

	var fc agentFileConfig
	err := loadJSONFile(path, &fc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ошибка парсинга файла конфигурации")
}

// LoadAgentConfig via file (через сброс flag.CommandLine)

func TestLoadAgentConfig_FromFileViaEnv(t *testing.T) {
	content := map[string]interface{}{
		"address":         "file-agent:5555",
		"report_interval": "20s",
		"poll_interval":   "4s",
		"key":             "file-agent-key",
		"rate_limit":      3,
		"crypto_key":      "/file/agent.pem",
	}
	tmpFile := writeJSONTempFile(t, content)

	t.Setenv("CONFIG", tmpFile)
	// Сбрасываем глобальный flag.CommandLine, чтобы повторный вызов flag.Parse не паниковал
	resetFlagCommandLine(t)

	cfg, err := LoadAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "file-agent:5555", cfg.Address)
	assert.Equal(t, 20, cfg.ReportInterval)
	assert.Equal(t, 4, cfg.PollInterval)
	assert.Equal(t, "file-agent-key", cfg.Key)
	assert.Equal(t, 3, cfg.RateLimit)
	assert.Equal(t, "/file/agent.pem", cfg.CryptoKey)
}

func TestLoadAgentConfig_EnvVarsOverrideFile(t *testing.T) {
	content := map[string]interface{}{
		"address":         "file-host:9999",
		"report_interval": "30s",
		"poll_interval":   "5s",
		"key":             "file-key",
		"rate_limit":      2,
	}
	tmpFile := writeJSONTempFile(t, content)

	t.Setenv("CONFIG", tmpFile)
	t.Setenv("ADDRESS", "env-host:1234")
	t.Setenv("KEY", "env-key")
	t.Setenv("RATE_LIMIT", "5")
	resetFlagCommandLine(t)

	cfg, err := LoadAgentConfig()
	require.NoError(t, err)

	// Env vars побеждают файл
	assert.Equal(t, "env-host:1234", cfg.Address)
	assert.Equal(t, "env-key", cfg.Key)
	assert.Equal(t, 5, cfg.RateLimit)
	// Из файла (env var не задан)
	assert.Equal(t, 30, cfg.ReportInterval)
}

func TestLoadAgentConfig_Defaults(t *testing.T) {
	t.Setenv("CONFIG", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("REPORT_INTERVAL", "")
	t.Setenv("POLL_INTERVAL", "")
	t.Setenv("KEY", "")
	t.Setenv("RATE_LIMIT", "")
	t.Setenv("CRYPTO_KEY", "")
	resetFlagCommandLine(t)

	cfg, err := LoadAgentConfig()
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 1, cfg.RateLimit)
}

// LoadServerConfig via file

func TestLoadServerConfig_FromFileViaEnv(t *testing.T) {
	restoreFalse := false
	content := map[string]interface{}{
		"address":          "file-server:7777",
		"restore":          restoreFalse,
		"store_interval":   "120s",
		"store_file":       "/file/metrics.json",
		"database_dsn":     "postgres://file@localhost/db",
		"key":              "file-server-key",
		"shutdown_timeout": "15s",
		"audit_file":       "/file/audit.log",
		"audit_url":        "http://file-audit.example.com",
		"crypto_key":       "/file/private.pem",
	}
	tmpFile := writeJSONTempFile(t, content)

	t.Setenv("CONFIG", tmpFile)
	t.Setenv("STORE_INTERVAL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("FILE_STORAGE_PATH", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("KEY", "")
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("AUDIT_URL", "")
	t.Setenv("CRYPTO_KEY", "")
	t.Setenv("RESTORE", "")
	resetFlagCommandLine(t)

	cfg, err := LoadServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "file-server:7777", cfg.Address)
	assert.Equal(t, false, cfg.Restore)
	assert.Equal(t, 120, cfg.StoreInterval)
	assert.Equal(t, "/file/metrics.json", cfg.FileStoragePath)
	assert.Equal(t, "postgres://file@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "file-server-key", cfg.Key)
	assert.Equal(t, 15, cfg.ShutdownTimeout)
	assert.Equal(t, "/file/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://file-audit.example.com", cfg.AuditURL)
	assert.Equal(t, "/file/private.pem", cfg.CryptoKey)
}

func TestLoadServerConfig_Defaults(t *testing.T) {
	t.Setenv("CONFIG", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("STORE_INTERVAL", "")
	t.Setenv("FILE_STORAGE_PATH", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("KEY", "")
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("AUDIT_URL", "")
	t.Setenv("CRYPTO_KEY", "")
	t.Setenv("RESTORE", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	resetFlagCommandLine(t)

	cfg, err := LoadServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.Address)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "metrics-db.json", cfg.FileStoragePath)
	assert.Equal(t, true, cfg.Restore)
	assert.Equal(t, 5, cfg.ShutdownTimeout)
}

func TestLoadServerConfig_EnvVarsOverrideFile(t *testing.T) {
	content := map[string]interface{}{
		"key":              "file-key",
		"audit_file":       "/file/audit.log",
		"audit_url":        "http://file-audit",
		"shutdown_timeout": "10s",
		"store_interval":   "60s",
	}
	tmpFile := writeJSONTempFile(t, content)

	t.Setenv("CONFIG", tmpFile)
	t.Setenv("KEY", "env-key")
	t.Setenv("AUDIT_URL", "http://env-audit")
	t.Setenv("STORE_INTERVAL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("FILE_STORAGE_PATH", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("CRYPTO_KEY", "")
	t.Setenv("RESTORE", "")
	resetFlagCommandLine(t)

	cfg, err := LoadServerConfig()
	require.NoError(t, err)

	assert.Equal(t, "env-key", cfg.Key)
	assert.Equal(t, "http://env-audit", cfg.AuditURL)
	// Из файла
	assert.Equal(t, "/file/audit.log", cfg.AuditFile)
	assert.Equal(t, 10, cfg.ShutdownTimeout)
	assert.Equal(t, 60, cfg.StoreInterval)
}

// resetFlagCommandLine сбрасывает глобальный flag.CommandLine и os.Args между тестами.
// Это необходимо, так как LoadAgentConfig/LoadServerConfig вызывают flag.Parse(),
// который читает os.Args — в тестовом бинаре там находятся флаги -test.*, неизвестные
// нашему flag.CommandLine.
func resetFlagCommandLine(t *testing.T) {
	t.Helper()

	origArgs := os.Args
	// Оставляем только имя бинаря — убираем все тестовые флаги
	os.Args = os.Args[:1]

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	})
}

func writeJSONTempFile(t *testing.T, data interface{}) string {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, b, 0644))
	return path
}
