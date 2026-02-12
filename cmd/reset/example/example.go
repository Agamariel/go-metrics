package example

// generate:reset
type Pool struct {
	Items   []string
	Counter int
	Active  bool
}

// generate:reset
type Buffer struct {
	Data     []byte
	Metadata map[string]interface{}
	Size     int
}
