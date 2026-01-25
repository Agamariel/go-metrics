package audit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
	wg        sync.WaitGroup
	logger    Logger
}

func NewPublisher(logger Logger) *Publisher {
	return &Publisher{
		observers: make([]Observer, 0),
		logger:    logger,
	}
}

// Attach добавляет наблюдателя в список подписчиков
func (p *Publisher) Attach(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// NotifyAll отправляет событие всем наблюдателям асинхронно.
func (p *Publisher) NotifyAll(ctx context.Context, event Event) {
	p.mu.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()

	// Если нет наблюдателей, ничего не делаем
	if len(observers) == 0 {
		return
	}

	// Создаем контекст с таймаутом для всех операций аудита
	auditCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, obs := range observers {
		p.wg.Add(1)
		go func(observer Observer) {
			defer p.wg.Done()

			if err := observer.Notify(auditCtx, event); err != nil {
				p.logger.Error("Ошибка при отправке события аудита", zap.Error(err))
			}
		}(obs)
	}
}

// Close ожидает завершения всех активных операций аудита.
func (p *Publisher) Close() {
	p.wg.Wait()
	p.logger.Info("Publisher аудита закрыт, все операции завершены")
}
