// Package crypto предоставляет функции для гибридного шифрования RSA+AES-GCM.
//
// Используется для защиты данных при передаче метрик от агента к серверу:
// агент шифрует данные публичным ключом сервера, сервер расшифровывает приватным.
//
// # Алгоритм
//
// Encrypt выполняет гибридное шифрование:
//  1. Генерирует случайный 32-байтный AES ключ и 12-байтный nonce
//  2. Шифрует данные AES-256-GCM
//  3. Шифрует AES ключ с помощью RSA-OAEP (SHA-256)
//
// Формат зашифрованных данных: [encryptedAESKey | nonce | ciphertext+tag]
// где len(encryptedAESKey) == размер RSA ключа в байтах.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadPublicKey загружает RSA публичный ключ из PEM-файла.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение файла публичного ключа: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("не удалось декодировать PEM блок из файла %s", path)
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("парсинг публичного ключа: %w", err)
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("ключ в файле %s не является RSA публичным ключом", path)
	}

	return rsaKey, nil
}

// LoadPrivateKey загружает RSA приватный ключ из PEM-файла.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение файла приватного ключа: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("не удалось декодировать PEM блок из файла %s", path)
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Попытка парсинга в устаревшем формате PKCS1
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("парсинг приватного ключа (PKCS8: %v, PKCS1: %v)", err, err2)
		}
		return rsaKey, nil
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("ключ в файле %s не является RSA приватным ключом", path)
	}

	return rsaKey, nil
}

// Encrypt шифрует данные с использованием гибридной схемы RSA-OAEP + AES-256-GCM.
//
// Формат результата: [encryptedAESKey(keySize байт) | nonce(12 байт) | ciphertext+tag]
func Encrypt(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	// Генерируем случайный AES ключ (32 байта = AES-256)
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("генерация AES ключа: %w", err)
	}

	// Шифруем AES ключ публичным RSA ключом (OAEP + SHA-256)
	encryptedAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("шифрование AES ключа: %w", err)
	}

	// Создаём AES-GCM шифр
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("создание AES шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("создание GCM: %w", err)
	}

	// Генерируем случайный nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("генерация nonce: %w", err)
	}

	// Шифруем данные AES-GCM
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Собираем результат: [encryptedAESKey | nonce | ciphertext+tag]
	result := make([]byte, 0, len(encryptedAESKey)+len(nonce)+len(ciphertext))
	result = append(result, encryptedAESKey...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt расшифровывает данные, зашифрованные функцией Encrypt.
func Decrypt(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	keySize := privateKey.Size()

	if len(data) < keySize+12 {
		return nil, fmt.Errorf("данные слишком короткие для расшифровки: %d байт", len(data))
	}

	// Извлекаем зашифрованный AES ключ
	encryptedAESKey := data[:keySize]

	// Расшифровываем AES ключ приватным RSA ключом
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedAESKey, nil)
	if err != nil {
		return nil, fmt.Errorf("расшифровка AES ключа: %w", err)
	}

	// Создаём AES-GCM шифр
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("создание AES шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("создание GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < keySize+nonceSize {
		return nil, fmt.Errorf("данные слишком короткие для извлечения nonce")
	}

	// Извлекаем nonce и ciphertext
	nonce := data[keySize : keySize+nonceSize]
	ciphertext := data[keySize+nonceSize:]

	// Расшифровываем данные
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("расшифровка данных AES-GCM: %w", err)
	}

	return plaintext, nil
}
