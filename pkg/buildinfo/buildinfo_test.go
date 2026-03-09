package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrNA_EmptyString(t *testing.T) {
	assert.Equal(t, "N/A", orNA(""))
}

func TestOrNA_NonEmptyString(t *testing.T) {
	assert.Equal(t, "v1.2.3", orNA("v1.2.3"))
}

func TestPrint_WithValues(t *testing.T) {
	// Print пишет в stdout — проверяем только что не паникует
	assert.NotPanics(t, func() {
		Print("v1.0.0", "2026-02-27", "abc123")
	})
}

func TestPrint_EmptyValues(t *testing.T) {
	assert.NotPanics(t, func() {
		Print("", "", "")
	})
}
