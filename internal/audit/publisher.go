package audit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Logger определяет интерфейс логгера для Publisher.
//
// Совместим с *zap.Logger и другими логгерами,
// реализующими эти методы.
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

// Publisher управляет подписчиками и рассылает им события аудита.
//
// Publisher потокобезопасен и может использоваться из нескольких горутин.
// События отправляются наблюдателям асинхронно в отдельных горутинах.
//
// Важно вызвать Close() перед завершением программы для корректного
// завершения всех операций аудита.
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
	wg        sync.WaitGroup
	logger    Logger
}

// NewPublisher создаёт новый издатель событий аудита.
//
// Параметр logger используется для логирования ошибок при отправке событий.
func NewPublisher(logger Logger) *Publisher {
	return &Publisher{
		observers: make([]Observer, 0),
		logger:    logger,
	}
}

// Attach добавляет наблюдателя в список подписчиков.
//
// Метод потокобезопасен. Добавленный наблюдатель будет получать
// все последующие события через NotifyAll.
func (p *Publisher) Attach(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// NotifyAll асинхронно отправляет событие всем подписанным наблюдателям.
//
// Каждый наблюдатель получает событие в отдельной горутине.
// Операции аудита выполняются с таймаутом 10 секунд.
// Ошибки логируются, но не прерывают отправку другим наблюдателям.
//
// Метод возвращается немедленно, не дожидаясь обработки события.
// Для гарантированной доставки перед завершением программы вызовите Close().
func (p *Publisher) NotifyAll(ctx context.Context, event Event) {
	p.mu.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()

	// Если нет наблюдателей, ничего не делаем
	if len(observers) == 0 {
		return
	}

	// Создаем контекст с таймаутом для всех операций аудита.
	// cancel нельзя вызывать через defer здесь: NotifyAll завершается сразу
	// после запуска горутин, и преждевременный cancel отменил бы их работу.
	// cancel вызывается отдельной горутиной после завершения всех наблюдателей.
	auditCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	var obsWg sync.WaitGroup
	for _, obs := range observers {
		obsWg.Add(1)
		p.wg.Add(1)
		go func(observer Observer) {
			defer p.wg.Done()
			defer obsWg.Done()

			if err := observer.Notify(auditCtx, event); err != nil {
				p.logger.Error("Ошибка при отправке события аудита", zap.Error(err))
			}
		}(obs)
	}

	// Освобождаем контекст после завершения всех наблюдателей
	go func() {
		obsWg.Wait()
		cancel()
	}()
}

// Close ожидает завершения всех активных операций аудита.
//
// Метод блокирует выполнение до тех пор, пока все запущенные
// горутины NotifyAll не завершатся. Вызывайте перед завершением
// программы для гарантии доставки всех событий.
func (p *Publisher) Close() {
	p.wg.Wait()
	p.logger.Info("Publisher аудита закрыт, все операции завершены")
}
