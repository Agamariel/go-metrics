package middleware

import (
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"go.uber.org/zap"
)

// responseWriter оборачивает http.ResponseWriter для перехвата статуса и размера ответа
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader перехватывает код статуса
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write перехватывает размер ответа
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger создаёт middleware для логирования запросов
func Logger(log logger.Logger) func(next http.Handler) http.Handler {
	if log == nil {
		log = logger.Nop()
	}

	return func(next http.Handler) http.Handler { // Возвращаем middleware функцию для обертки следующего обработчика
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // возвращаем функционально расширенный хендлер
			start := time.Now()

			// Оборачиваем ResponseWriter для добавления статуса и размера
			rw := &responseWriter{
				ResponseWriter: w,
				status:         200,
				size:           0,
			}

			// Выполняем следующий обработчик
			next.ServeHTTP(rw, r)

			// Вычисляем время выполнения
			duration := time.Since(start)

			// Логируем информацию о запросе и ответе
			log.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.Int("status", rw.status),
				zap.Int("size", rw.size),
				zap.Duration("duration", duration),
			)
		})
	}
}
