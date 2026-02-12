// Package example содержит примеры структур для демонстрации генерации методов Reset().
package example

// Pool представляет пул строковых элементов с счётчиком и флагом активности.
//
// generate:reset
type Pool struct {
	Items   []string
	Counter int
	Active  bool
}

// Buffer представляет буфер с данными, метаданными и размером.
//
// generate:reset
type Buffer struct {
	Data     []byte
	Metadata map[string]interface{}
	Size     int
}
