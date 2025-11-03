package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// responseBuffer буферизует ответ
type responseBuffer struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func (rb *responseBuffer) WriteHeader(statusCode int) {
	rb.statusCode = statusCode
}

func (rb *responseBuffer) Write(b []byte) (int, error) {
	return rb.buffer.Write(b)
}

// compressibleTypes содержит список MIME типов, которые должны сжиматься
var compressibleTypes = []string{
	"application/javascript",
	"application/json",
	"application/xml",
	"application/yaml",
	"text/css",
	"text/html",
	"text/plain",
	"text/xml",
	"text/csv",
}

// shouldCompress проверяет, нужно ли сжимать данный Content-Type
func shouldCompress(contentType string) bool {
	for _, compressibleType := range compressibleTypes {
		if strings.Contains(contentType, compressibleType) {
			return true
		}
	}
	return false
}

// GzipMiddleware обрабатывает сжатие и распаковку gzip
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка входящего запроса если Content-Encoding: gzip
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			// Заменяем тело запроса на распакованное для дальнейшей обработки
			r.Body = io.NopCloser(gz)
		}

		// Проверяем, поддерживает ли клиент gzip для ответа
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			// Клиент не поддерживает gzip - пропускаем обработку
			next.ServeHTTP(w, r)
			return
		}

		// Используем буферизованный writer для сбора ответа
		buf := &responseBuffer{
			ResponseWriter: w,
			buffer:         &bytes.Buffer{},
		}

		// Выполняем обработчик
		next.ServeHTTP(buf, r)

		// Проверяем Content-Type - сжимаем только поддерживаемые типы
		contentType := buf.Header().Get("Content-Type")
		compress := shouldCompress(contentType)

		if !compress || buf.buffer.Len() == 0 {
			// Не сжимаем, просто отправляем
			if buf.statusCode != 0 {
				w.WriteHeader(buf.statusCode)
			}
			w.Write(buf.buffer.Bytes())
			return
		}

		// Сжимаем ответ
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		if buf.statusCode != 0 {
			w.WriteHeader(buf.statusCode)
		}

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gz.Write(buf.buffer.Bytes())
	})
}
