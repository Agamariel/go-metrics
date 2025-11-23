package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"net/http"

	"github.com/Agamariel/go-metrics/pkg/sha256hash"
)

type hashWriter struct {
	http.ResponseWriter
	mac hash.Hash
}

func (w *hashWriter) Write(b []byte) (int, error) {
	w.mac.Write(b)
	return w.ResponseWriter.Write(b)
}

// HashMiddleware проверяет HashSHA256 запроса и добавляет его в ответ.
func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if err := verifyRequestHash(r, key); err != nil {
				http.Error(w, "invalid hash", http.StatusBadRequest)
				return
			}

			mac := hmac.New(sha256.New, []byte(key))
			hw := &hashWriter{
				ResponseWriter: w,
				mac:            mac,
			}

			next.ServeHTTP(hw, r)

			w.Header().Set("HashSHA256", hex.EncodeToString(mac.Sum(nil)))
		})
	}
}

func verifyRequestHash(r *http.Request, key string) error {
	h := r.Header.Get("HashSHA256")
	if h == "" {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if !sha256hash.VerifySHA256(body, key, h) {
		return http.ErrAbortHandler
	}
	return nil
}
