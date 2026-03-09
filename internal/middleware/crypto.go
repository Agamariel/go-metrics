package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/Agamariel/go-metrics/pkg/crypto"
)

// CryptoMiddleware расшифровывает тело входящего запроса приватным RSA ключом.
// Должна стоять первой в цепочке middleware — до GzipMiddleware.
// Если ключ не задан (nil), middleware пропускает запрос без изменений.
func CryptoMiddleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			encrypted, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "ошибка чтения тела запроса", http.StatusBadRequest)
				return
			}

			plaintext, err := crypto.Decrypt(privateKey, encrypted)
			if err != nil {
				http.Error(w, "ошибка расшифровки запроса", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))

			next.ServeHTTP(w, r)
		})
	}
}
