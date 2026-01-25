package app

import (
	"github.com/Agamariel/go-metrics/internal/audit"
	"go.uber.org/zap"
)

// initAudit инициализирует систему аудита (опционально).
func (a *App) initAudit() error {
	if a.config.AuditFile == "" && a.config.AuditURL == "" {
		return nil
	}

	a.publisher = audit.NewPublisher(a.logger)

	if a.config.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(a.config.AuditFile)
		if err != nil {
			return err
		}
		a.publisher.Attach(fileObserver)
		a.logger.Info("Включен файловый аудит", zap.String("file", a.config.AuditFile))
	}

	if a.config.AuditURL != "" {
		httpObserver := audit.NewHTTPObserver(a.config.AuditURL)
		a.publisher.Attach(httpObserver)
		a.logger.Info("Включен HTTP аудит", zap.String("url", a.config.AuditURL))
	}

	return nil
}
