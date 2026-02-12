// Package audit предоставляет систему аудита для отслеживания операций с метриками.
//
// Пакет реализует паттерн Observer (Наблюдатель), позволяя подписываться
// на события обновления метрик и обрабатывать их асинхронно.
//
// # Компоненты
//
//   - Event: структура события аудита
//   - Observer: интерфейс наблюдателя
//   - Publisher: издатель событий, управляющий подписчиками
//   - FileObserver: наблюдатель, записывающий события в файл
//   - HTTPObserver: наблюдатель, отправляющий события по HTTP
//
// # Пример использования
//
//	publisher := audit.NewPublisher(logger)
//	defer publisher.Close()
//
//	// Добавление наблюдателей
//	publisher.Attach(audit.NewFileObserver("audit.log", logger))
//	publisher.Attach(audit.NewHTTPObserver("http://audit-service/events", logger))
//
//	// Отправка события (выполняется асинхронно)
//	event := audit.Event{
//	    Timestamp: time.Now().Unix(),
//	    Metrics:   []string{"cpu", "memory"},
//	    IPAddress: "192.168.1.1",
//	}
//	publisher.NotifyAll(ctx, event)
package audit

// Event представляет событие аудита для операций с метриками.
//
// Событие содержит информацию о времени запроса, запрошенных метриках
// и IP-адресе клиента. Используется для логирования и мониторинга
// доступа к метрикам.
type Event struct {
	// Timestamp — Unix timestamp момента создания события.
	Timestamp int64 `json:"ts"`

	// Metrics — список имён метрик, затронутых операцией.
	Metrics []string `json:"metrics"`

	// IPAddress — IP-адрес клиента, выполнившего запрос.
	IPAddress string `json:"ip_address"`
}
