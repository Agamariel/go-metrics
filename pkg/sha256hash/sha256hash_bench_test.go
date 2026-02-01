package sha256hash

import (
	"fmt"
	"testing"
)

// BenchmarkCalculateSHA256 измеряет скорость вычисления HMAC-SHA256
func BenchmarkCalculateSHA256(b *testing.B) {
	key := "secret-key-12345"

	dataSizes := []int{100, 1024, 10240, 102400} // 100B, 1KB, 10KB, 100KB
	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i % 256)
			}
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				_ = CalculateSHA256(data, key)
			}
		})
	}
}

// BenchmarkVerifySHA256 измеряет скорость проверки HMAC-SHA256
func BenchmarkVerifySHA256(b *testing.B) {
	key := "secret-key-12345"

	dataSizes := []int{100, 1024, 10240, 102400}
	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("Size_%dB", size), func(b *testing.B) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i % 256)
			}
			expectedHash := CalculateSHA256(data, key)
			b.ResetTimer()
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				_ = VerifySHA256(data, key, expectedHash)
			}
		})
	}
}

// BenchmarkCalculateSHA256_NoKey измеряет производительность при отсутствии ключа
func BenchmarkCalculateSHA256_NoKey(b *testing.B) {
	data := []byte("some test data for benchmarking")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateSHA256(data, "")
	}
}
