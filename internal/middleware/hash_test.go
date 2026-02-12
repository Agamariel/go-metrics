package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agamariel/go-metrics/pkg/sha256hash"
)

func TestHashMiddleware_NoKey(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := HashMiddleware("")
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	// Без ключа middleware не должен проверять или добавлять хеш
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Header().Get("HashSHA256") != "" {
		t.Error("Should not add HashSHA256 header when key is empty")
	}
}

func TestHashMiddleware_WithValidHash(t *testing.T) {
	key := "secret-key"
	requestBody := []byte("request data")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем тело для проверки
		body, _ := io.ReadAll(r.Body)
		if string(body) != string(requestBody) {
			t.Errorf("Request body mismatch: expected %q, got %q", requestBody, body)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response data"))
	})

	middleware := HashMiddleware(key)
	wrappedHandler := middleware(handler)

	// Вычисляем правильный хеш для запроса
	expectedHash := sha256hash.CalculateSHA256(requestBody, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(requestBody))
	req.Header.Set("HashSHA256", expectedHash)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Проверяем статус
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Проверяем, что добавлен хеш ответа
	responseHash := rr.Header().Get("HashSHA256")
	if responseHash == "" {
		t.Error("Response should have HashSHA256 header")
	}

	// Проверяем, что хеш валиден
	responseBody := rr.Body.Bytes()
	if !sha256hash.VerifySHA256(responseBody, key, responseHash) {
		t.Error("Response hash is invalid")
	}
}

func TestHashMiddleware_WithInvalidHash(t *testing.T) {
	key := "secret-key"
	requestBody := []byte("request data")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := HashMiddleware(key)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(requestBody))
	req.Header.Set("HashSHA256", "invalid-hash-value")

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Ожидаем ошибку
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "invalid hash") {
		t.Errorf("Expected error message about invalid hash, got %q", rr.Body.String())
	}
}

func TestHashMiddleware_NoHashInRequest(t *testing.T) {
	key := "secret-key"
	requestBody := []byte("request data")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	middleware := HashMiddleware(key)
	wrappedHandler := middleware(handler)

	// Запрос без хеша должен проходить
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(requestBody))
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Хеш ответа должен быть добавлен
	if rr.Header().Get("HashSHA256") == "" {
		t.Error("Response should have HashSHA256 header")
	}
}

func TestHashMiddleware_EmptyBody(t *testing.T) {
	key := "secret-key"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	middleware := HashMiddleware(key)
	wrappedHandler := middleware(handler)

	// Запрос с пустым телом
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Хеш должен быть добавлен
	if rr.Header().Get("HashSHA256") == "" {
		t.Error("Response should have HashSHA256 header even with empty request body")
	}
}

func TestHashWriter_Write(t *testing.T) {
	rr := httptest.NewRecorder()
	
	// Создаём HMAC вручную, как это делается в middleware
	mac := hmac.New(sha256.New, []byte("test-key"))
	
	hw := &hashWriter{
		ResponseWriter: rr,
		mac:            mac,
	}

	data := []byte("test data")
	n, err := hw.Write(data)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Проверяем, что данные записались в ResponseWriter
	if rr.Body.String() != "test data" {
		t.Errorf("Expected body %q, got %q", "test data", rr.Body.String())
	}

	// Проверяем, что данные записались в mac
	if hw.mac.Sum(nil) == nil {
		t.Error("MAC should have data")
	}
}

func TestVerifyRequestHash_ValidHash(t *testing.T) {
	key := "test-key"
	body := []byte("test body")
	hash := sha256hash.CalculateSHA256(body, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", hash)

	err := verifyRequestHash(req, key)
	if err != nil {
		t.Errorf("verifyRequestHash failed with valid hash: %v", err)
	}

	// Проверяем, что тело запроса можно прочитать снова
	readBody, _ := io.ReadAll(req.Body)
	if string(readBody) != string(body) {
		t.Error("Request body should be readable after verification")
	}
}

func TestVerifyRequestHash_InvalidHash(t *testing.T) {
	key := "test-key"
	body := []byte("test body")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "invalid-hash")

	err := verifyRequestHash(req, key)
	if err == nil {
		t.Error("Expected error with invalid hash")
	}
}

func TestVerifyRequestHash_NoHash(t *testing.T) {
	key := "test-key"
	body := []byte("test body")

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	// Без заголовка HashSHA256

	err := verifyRequestHash(req, key)
	if err != nil {
		t.Errorf("Should not fail when hash header is missing: %v", err)
	}
}

func TestHashMiddleware_MultipleWrites(t *testing.T) {
	key := "secret-key"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("part1-"))
		w.Write([]byte("part2-"))
		w.Write([]byte("part3"))
	})

	middleware := HashMiddleware(key)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Проверяем, что все части записались
	if rr.Body.String() != "part1-part2-part3" {
		t.Errorf("Expected %q, got %q", "part1-part2-part3", rr.Body.String())
	}

	// Проверяем хеш
	responseHash := rr.Header().Get("HashSHA256")
	if responseHash == "" {
		t.Error("Response should have HashSHA256 header")
	}

	// Хеш должен быть вычислен для всего ответа
	expectedHash := sha256hash.CalculateSHA256([]byte("part1-part2-part3"), key)
	if responseHash != expectedHash {
		t.Errorf("Expected hash %q, got %q", expectedHash, responseHash)
	}
}
