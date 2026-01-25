package audit

// Event представляет событие аудита запроса к метрикам
type Event struct {
	Timestamp int64    `json:"ts"`         // Unix timestamp события
	Metrics   []string `json:"metrics"`    // Наименование полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}
