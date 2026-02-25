package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware_Compression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test response data"}`))
	})

	middleware := GzipMiddleware(handler)

	// Запрос с поддержкой gzip
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Проверяем заголовок Content-Encoding
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip header")
	}

	// Проверяем, что ответ сжат
	reader, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed body: %v", err)
	}

	expected := `{"message": "test response data"}`
	if string(body) != expected {
		t.Errorf("Expected %q, got %q", expected, string(body))
	}
}

func TestGzipMiddleware_NoCompression_NoAcceptEncoding(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "test"}`))
	})

	middleware := GzipMiddleware(handler)

	// Запрос без Accept-Encoding
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Не должно быть сжатия
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Should not have gzip encoding when client doesn't support it")
	}

	// Проверяем, что тело не сжато
	if rr.Body.String() != `{"message": "test"}` {
		t.Errorf("Expected uncompressed body, got %q", rr.Body.String())
	}
}

func TestGzipMiddleware_NoCompression_UnsupportedContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary image data"))
	})

	middleware := GzipMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Не должно быть сжатия для изображений
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Should not compress images")
	}

	if rr.Body.String() != "binary image data" {
		t.Errorf("Expected uncompressed body, got %q", rr.Body.String())
	}
}

func TestGzipMiddleware_Decompression(t *testing.T) {
	var receivedBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	middleware := GzipMiddleware(handler)

	// Создаём сжатый запрос
	requestBody := "compressed request data"
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(requestBody))
	gzWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Проверяем, что тело было распаковано
	if receivedBody != requestBody {
		t.Errorf("Expected %q, got %q", requestBody, receivedBody)
	}
}

func TestGzipMiddleware_InvalidGzipData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := GzipMiddleware(handler)

	// Отправляем невалидные gzip данные
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid gzip data"))
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Ожидаем ошибку
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Invalid gzip data") {
		t.Errorf("Expected error message about invalid gzip, got %q", rr.Body.String())
	}
}

func TestGzipMiddleware_EmptyResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Пустой ответ
	})

	middleware := GzipMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	// Пустой ответ не должен сжиматься
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Empty response should not be compressed")
	}

	if rr.Body.Len() != 0 {
		t.Errorf("Expected empty body, got %d bytes", rr.Body.Len())
	}
}

func TestGzipMiddleware_MultipleContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		shouldGzip  bool
	}{
		{"JSON", "application/json", true},
		{"HTML", "text/html; charset=utf-8", true},
		{"Plain text", "text/plain", true},
		{"CSS", "text/css", true},
		{"JavaScript", "application/javascript", true},
		{"XML", "application/xml", true},
		{"Image", "image/jpeg", false},
		{"Binary", "application/octet-stream", false},
		{"PDF", "application/pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test response data"))
			})

			middleware := GzipMiddleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			hasGzip := rr.Header().Get("Content-Encoding") == "gzip"
			if hasGzip != tt.shouldGzip {
				t.Errorf("Content-Type %q: expected gzip=%v, got gzip=%v",
					tt.contentType, tt.shouldGzip, hasGzip)
			}
		})
	}
}

func TestShouldCompress(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/html", true},
		{"text/plain", true},
		{"text/css", true},
		{"application/javascript", true},
		{"application/xml", true},
		{"text/xml", true},
		{"image/png", false},
		{"image/jpeg", false},
		{"video/mp4", false},
		{"application/octet-stream", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := shouldCompress(tt.contentType)
			if result != tt.expected {
				t.Errorf("shouldCompress(%q) = %v, expected %v",
					tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestResponseBuffer(t *testing.T) {
	buf := &responseBuffer{
		ResponseWriter: httptest.NewRecorder(),
		buffer:         &bytes.Buffer{},
	}

	// Тест WriteHeader
	buf.WriteHeader(http.StatusCreated)
	if buf.statusCode != http.StatusCreated {
		t.Errorf("Expected statusCode %d, got %d", http.StatusCreated, buf.statusCode)
	}

	// Тест Write
	data := []byte("test data")
	n, err := buf.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	if buf.buffer.String() != "test data" {
		t.Errorf("Expected buffer to contain %q, got %q", "test data", buf.buffer.String())
	}
}
