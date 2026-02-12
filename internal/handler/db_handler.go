package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"go.uber.org/zap"
)

// DBHandler предоставляет HTTP-обработчик для проверки соединения с базой данных.
//
// Используется для health check эндпоинта, позволяющего
// проверить доступность базы данных.
type DBHandler struct {
	db     *sql.DB
	logger logger.Logger
}

// NewDBHandler создаёт новый обработчик для проверки базы данных.
//
// Параметры:
//   - database: соединение с базой данных (может быть nil)
//   - log: логгер для записи ошибок (если nil, используется Nop-логгер)
//
// Пример использования:
//
//	db, _ := sql.Open("postgres", dsn)
//	handler := NewDBHandler(db, logger)
//	r.Get("/ping", handler.PingDB)
func NewDBHandler(database *sql.DB, log logger.Logger) *DBHandler {
	if log == nil {
		log = logger.Nop()
	}
	return &DBHandler{
		db:     database,
		logger: log,
	}
}

// PingDB обрабатывает GET /ping.
//
// Проверяет соединение с базой данных с таймаутом 3 секунды.
// Используется для health check и мониторинга доступности БД.
//
// Коды ответа:
//   - 200 OK: соединение с БД успешно
//   - 500 Internal Server Error: БД не сконфигурирована или недоступна
func (h *DBHandler) PingDB(w http.ResponseWriter, r *http.Request) {
	// Если БД не инициализирована, возвращаем ошибку
	if h.db == nil {
		http.Error(w, "База данных не сконфигурирована", http.StatusInternalServerError)
		return
	}

	// Создаем контекст с таймаутом для проверки
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Проверяем соединение с БД
	if err := h.db.PingContext(ctx); err != nil {
		if h.logger != nil {
			h.logger.Error("Ошибка проверки соединения с БД", zap.Error(err))
		}
		http.Error(w, "Ошибка проверки соединения с БД", http.StatusInternalServerError)
		return
	}

	// Соединение успешно
	w.WriteHeader(http.StatusOK)
}
