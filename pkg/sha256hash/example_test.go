package sha256hash_test

import (
	"encoding/json"
	"fmt"

	"github.com/Agamariel/go-metrics/pkg/sha256hash"
)

// Example демонстрирует базовое использование пакета sha256hash.
func Example() {
	key := "my-secret-key"
	data := []byte("Hello, World!")

	// Вычисляем хеш
	hash := sha256hash.CalculateSHA256(data, key)
	fmt.Printf("Hash length: %d\n", len(hash))

	// Проверяем хеш
	valid := sha256hash.VerifySHA256(data, key, hash)
	fmt.Printf("Valid: %v\n", valid)

	// Output:
	// Hash length: 64
	// Valid: true
}

// ExampleCalculateSHA256 демонстрирует вычисление HMAC-SHA256 хеша.
func ExampleCalculateSHA256() {
	key := "secret"
	data := []byte(`{"id":"cpu","type":"gauge","value":42.5}`)

	hash := sha256hash.CalculateSHA256(data, key)
	fmt.Printf("Длина хеша: %d символов\n", len(hash))
	fmt.Printf("Хеш не пустой: %v\n", hash != "")

	// Output:
	// Длина хеша: 64 символов
	// Хеш не пустой: true
}

// ExampleCalculateSHA256_emptyKey демонстрирует поведение при пустом ключе.
func ExampleCalculateSHA256_emptyKey() {
	data := []byte("some data")

	// При пустом ключе хеширование отключено
	hash := sha256hash.CalculateSHA256(data, "")
	fmt.Printf("Хеш при пустом ключе: '%s'\n", hash)

	// Output:
	// Хеш при пустом ключе: ''
}

// ExampleVerifySHA256 демонстрирует проверку HMAC-SHA256 хеша.
func ExampleVerifySHA256() {
	key := "secret"
	data := []byte("important data")

	// Вычисляем оригинальный хеш
	originalHash := sha256hash.CalculateSHA256(data, key)

	// Проверяем с правильными данными
	valid := sha256hash.VerifySHA256(data, key, originalHash)
	fmt.Printf("Корректные данные: %v\n", valid)

	// Проверяем с изменёнными данными
	modifiedData := []byte("modified data")
	valid = sha256hash.VerifySHA256(modifiedData, key, originalHash)
	fmt.Printf("Изменённые данные: %v\n", valid)

	// Output:
	// Корректные данные: true
	// Изменённые данные: false
}

// ExampleVerifySHA256_emptyKey демонстрирует проверку при пустом ключе.
func ExampleVerifySHA256_emptyKey() {
	data := []byte("any data")

	// При пустом ключе проверка всегда успешна (отключена)
	valid := sha256hash.VerifySHA256(data, "", "any_hash")
	fmt.Printf("Пустой ключ: %v\n", valid)

	// Output:
	// Пустой ключ: true
}

// ExampleVerifySHA256_invalidHash демонстрирует обработку невалидного хеша.
func ExampleVerifySHA256_invalidHash() {
	key := "secret"
	data := []byte("data")

	// Невалидная hex-строка
	valid := sha256hash.VerifySHA256(data, key, "not-a-valid-hex")
	fmt.Printf("Невалидный hex: %v\n", valid)

	// Неправильный хеш
	valid = sha256hash.VerifySHA256(data, key, "aabbccdd")
	fmt.Printf("Неправильный хеш: %v\n", valid)

	// Output:
	// Невалидный hex: false
	// Неправильный хеш: false
}

// Example_metricSigning демонстрирует подпись метрик для передачи между агентом и сервером.
func Example_metricSigning() {
	type Metric struct {
		ID    string   `json:"id"`
		Type  string   `json:"type"`
		Value *float64 `json:"value,omitempty"`
		Hash  string   `json:"hash,omitempty"`
	}

	key := "shared-secret"

	// Агент: создаём и подписываем метрику
	value := 42.5
	metric := Metric{
		ID:    "cpu_usage",
		Type:  "gauge",
		Value: &value,
	}

	// Сериализуем без хеша для вычисления подписи
	data, _ := json.Marshal(metric)
	metric.Hash = sha256hash.CalculateSHA256(data, key)

	// Отправляем метрику (имитация)
	payload, _ := json.Marshal(metric)
	fmt.Printf("Payload contains hash: %v\n", len(metric.Hash) > 0)

	// Сервер: получаем и проверяем метрику
	var received Metric
	json.Unmarshal(payload, &received)
	receivedHash := received.Hash
	received.Hash = ""

	// Проверяем подпись
	dataToVerify, _ := json.Marshal(received)
	valid := sha256hash.VerifySHA256(dataToVerify, key, receivedHash)
	fmt.Printf("Signature valid: %v\n", valid)

	// Output:
	// Payload contains hash: true
	// Signature valid: true
}
