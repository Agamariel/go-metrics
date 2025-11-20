package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
	"go.uber.org/zap"
)

// DBHandler — HTTP-обработчик для проверки соединения с базой данных
type DBHandler struct {
	db     *sql.DB
	logger logger.Logger
}

// NewDBHandler создает новый хендлер для проверки БД
func NewDBHandler(database *sql.DB, log logger.Logger) *DBHandler {
	if log == nil {
		log = logger.Nop()
	}
	return &DBHandler{
		db:     database,
		logger: log,
	}
}

// PingDB обрабатывает GET /ping
// Возвращает 200 OK при успехе, 500 Internal Server Error при ошибке
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
