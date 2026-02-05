// Package sha256hash предоставляет функции для вычисления и проверки HMAC-SHA256 хешей.
//
// Пакет используется для обеспечения целостности данных при передаче метрик
// между агентом и сервером. При наличии общего секретного ключа все метрики
// подписываются HMAC-SHA256 хешем.
//
// # Безопасность
//
// Функция VerifySHA256 использует константное время сравнения (hmac.Equal),
// что защищает от timing-атак.
//
// # Пример использования
//
//	key := "my-secret-key"
//	data := []byte(`{"id":"cpu","type":"gauge","value":42.5}`)
//
//	// Вычисление хеша на стороне отправителя
//	hash := sha256hash.CalculateSHA256(data, key)
//
//	// Проверка хеша на стороне получателя
//	if !sha256hash.VerifySHA256(data, key, hash) {
//	    log.Fatal("Данные повреждены или подделаны")
//	}
package sha256hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CalculateSHA256 вычисляет HMAC-SHA256 хеш данных с указанным ключом.
//
// Возвращает хеш в виде hex-строки.
// Если ключ пустой, возвращает пустую строку (хеширование отключено).
//
// Параметры:
//   - data: данные для хеширования
//   - key: секретный ключ для HMAC
func CalculateSHA256(data []byte, key string) string {
	if key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySHA256 проверяет соответствие HMAC-SHA256 хеша данным.
//
// Использует константное время сравнения для защиты от timing-атак.
//
// Параметры:
//   - data: данные для проверки
//   - key: секретный ключ для HMAC
//   - expectedHash: ожидаемый хеш в hex-формате
//
// Возвращает true если:
//   - ключ пустой (проверка отключена)
//   - вычисленный хеш совпадает с ожидаемым
//
// Возвращает false если:
//   - expectedHash не является валидной hex-строкой
//   - хеши не совпадают
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
