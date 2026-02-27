package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKeyPair генерирует RSA ключевую пару и сохраняет в temp-файлы.
func generateTestKeyPair(t *testing.T) (pubKeyPath, privKeyPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	dir := t.TempDir()

	// Сохраняем приватный ключ (PKCS8)
	privKeyPath = filepath.Join(dir, "private.pem")
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	require.NoError(t, os.WriteFile(privKeyPath, privPEM, 0600))

	// Сохраняем публичный ключ (PKIX)
	pubKeyPath = filepath.Join(dir, "public.pem")
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	require.NoError(t, os.WriteFile(pubKeyPath, pubPEM, 0644))

	return pubKeyPath, privKeyPath
}

func TestLoadPublicKey_Success(t *testing.T) {
	pubPath, _ := generateTestKeyPair(t)

	key, err := LoadPublicKey(pubPath)
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestLoadPublicKey_FileNotFound(t *testing.T) {
	_, err := LoadPublicKey("/nonexistent/path/key.pem")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "чтение файла публичного ключа")
}

func TestLoadPublicKey_InvalidPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0644))

	_, err := LoadPublicKey(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не удалось декодировать PEM блок")
}

func TestLoadPublicKey_WrongKeyType(t *testing.T) {
	// Кладём приватный ключ туда, где ожидается публичный
	_, privPath := generateTestKeyPair(t)
	privData, err := os.ReadFile(privPath)
	require.NoError(t, err)

	// Декодируем PEM и перекодируем с типом PUBLIC KEY но содержимым PKCS8
	block, _ := pem.Decode(privData)
	require.NotNil(t, block)

	// Подставляем корректный PEM-заголовок, но содержимое — приватный ключ как PKIX
	// x509.ParsePKIXPublicKey ожидает публичный ключ — это вызовет ошибку парсинга
	wrongPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: block.Bytes})
	path := filepath.Join(t.TempDir(), "wrong.pem")
	require.NoError(t, os.WriteFile(path, wrongPEM, 0644))

	_, err = LoadPublicKey(path)
	assert.Error(t, err)
}

func TestLoadPrivateKey_PKCS8_Success(t *testing.T) {
	_, privPath := generateTestKeyPair(t)

	key, err := LoadPrivateKey(privPath)
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestLoadPrivateKey_PKCS1_Success(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Сохраняем в формате PKCS1
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	path := filepath.Join(t.TempDir(), "pkcs1.pem")
	require.NoError(t, os.WriteFile(path, privPEM, 0600))

	key, err := LoadPrivateKey(path)
	require.NoError(t, err)
	assert.NotNil(t, key)
}

func TestLoadPrivateKey_FileNotFound(t *testing.T) {
	_, err := LoadPrivateKey("/nonexistent/path/key.pem")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "чтение файла приватного ключа")
}

func TestLoadPrivateKey_InvalidPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a pem"), 0644))

	_, err := LoadPrivateKey(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не удалось декодировать PEM блок")
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plaintext := []byte("hello, metrics!")

	encrypted, err := Encrypt(&privateKey.PublicKey, plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := Decrypt(privateKey, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_LargePayload(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plaintext := make([]byte, 10_000)
	_, err = rand.Read(plaintext)
	require.NoError(t, err)

	encrypted, err := Encrypt(&privateKey.PublicKey, plaintext)
	require.NoError(t, err)

	decrypted, err := Decrypt(privateKey, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecrypt_TooShortData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	_, err = Decrypt(privateKey, []byte("short"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "данные слишком короткие")
}
