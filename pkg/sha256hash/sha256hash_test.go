package sha256hash

import (
	"testing"
)

func TestCalculateSHA256(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      string
		expected string
	}{
		{
			name:     "empty key",
			data:     []byte("test data"),
			key:      "",
			expected: "",
		},
		{
			name:     "with key",
			data:     []byte("test data"),
			key:      "secret",
			expected: "c66d73e3c4354ac8fa8c95dd1f3f79931d723bbc430030329a4de1fcb0993dc3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSHA256(tt.data, tt.key)
			if result != tt.expected {
				t.Errorf("CalculateSHA256() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVerifySHA256(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		key          string
		expectedHash string
		expected     bool
	}{
		{
			name:         "empty key always valid",
			data:         []byte("test data"),
			key:          "",
			expectedHash: "any hash",
			expected:     true,
		},
		{
			name:         "valid hash",
			data:         []byte("test data"),
			key:          "secret",
			expectedHash: CalculateSHA256([]byte("test data"), "secret"),
			expected:     true,
		},
		{
			name:         "invalid hash",
			data:         []byte("test data"),
			key:          "secret",
			expectedHash: "invalid",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifySHA256(tt.data, tt.key, tt.expectedHash)
			if result != tt.expected {
				t.Errorf("VerifySHA256() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
