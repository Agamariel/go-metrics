// Package models содержит структуры данных для работы с метриками.
//
// Пакет определяет модель Metrics, которая используется для передачи
// метрик между агентом и сервером в формате JSON.
package models

// Типы поддерживаемых метрик.
const (
	// Counter — тип метрики-счётчика. Значение счётчика накапливается
	// (каждое новое значение delta прибавляется к текущему).
	Counter = "counter"

	// Gauge — тип метрики-измерения. Значение замещается при каждом обновлении.
	Gauge = "gauge"
)

// Metrics представляет метрику для обмена между агентом и сервером.
//
// Структура использует указатели для Delta и Value, чтобы различать
// нулевое значение и отсутствие значения при сериализации в JSON.
//
// Пример использования:
//
//	// Создание gauge-метрики
//	value := 42.5
//	m := models.Metrics{
//	    ID:    "temperature",
//	    MType: models.Gauge,
//	    Value: &value,
//	}
//
//	// Создание counter-метрики
//	delta := int64(10)
//	m := models.Metrics{
//	    ID:    "requests",
//	    MType: models.Counter,
//	    Delta: &delta,
//	}
type Metrics struct {
	// ID — уникальный идентификатор (имя) метрики.
	ID string `json:"id"`

	// MType — тип метрики: "gauge" или "counter".
	MType string `json:"type"`

	// Delta — значение для counter-метрик. Прибавляется к текущему значению.
	// Используется только для метрик типа Counter.
	Delta *int64 `json:"delta,omitempty"`

	// Value — значение для gauge-метрик. Замещает текущее значение.
	// Используется только для метрик типа Gauge.
	Value *float64 `json:"value,omitempty"`

	// Hash — HMAC-SHA256 хеш для проверки целостности данных.
	// Вычисляется на основе ID, MType и значения метрики.
	Hash string `json:"hash,omitempty"`
}
