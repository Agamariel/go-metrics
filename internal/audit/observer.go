package audit

import "context"

// Observer представляет интерфейс наблюдателя в паттерне Observer.
// Каждый наблюдатель получает уведомления о событиях аудита.
type Observer interface {
	// Notify отправляет событие аудита наблюдателю
	Notify(ctx context.Context, event Event) error
}
