package sha256Hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CalculateSHA256 возвращает HMAC-SHA256(data, key) в hex-кодировке.
func CalculateSHA256(data []byte, key string) string {
	if key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySHA256 проверяет hex-хеш константным временем.
func VerifySHA256(data []byte, key, expectedHash string) bool {
	if key == "" {
		return true
	}
	computed := hmac.New(sha256.New, []byte(key))
	computed.Write(data)
	expected, err := hex.DecodeString(expectedHash)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, computed.Sum(nil))
}
