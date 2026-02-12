package example

import "testing"

func TestPool_Reset(t *testing.T) {
	p := &Pool{
		Items:   []string{"a", "b", "c"},
		Counter: 42,
		Active:  true,
	}

	p.Reset()

	if len(p.Items) != 0 {
		t.Errorf("Expected Items length 0, got %d", len(p.Items))
	}
	if cap(p.Items) == 0 {
		t.Error("Expected Items capacity to be preserved")
	}
	if p.Counter != 0 {
		t.Errorf("Expected Counter 0, got %d", p.Counter)
	}
	if p.Active != false {
		t.Error("Expected Active false")
	}
}

func TestBuffer_Reset(t *testing.T) {
	b := &Buffer{
		Data:     []byte("hello"),
		Metadata: map[string]interface{}{"key": "value"},
		Size:     100,
	}

	b.Reset()

	if len(b.Data) != 0 {
		t.Errorf("Expected Data length 0, got %d", len(b.Data))
	}
	if len(b.Metadata) != 0 {
		t.Errorf("Expected Metadata length 0, got %d", len(b.Metadata))
	}
	if b.Size != 0 {
		t.Errorf("Expected Size 0, got %d", b.Size)
	}
}
