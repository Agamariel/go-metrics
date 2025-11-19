package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Agamariel/go-metrics/internal/logger"
)

// DB интерфейс для работы с базой данных
type DB interface {
	Ping(ctx context.Context, log logger.Logger) error
}

// DBHandler — HTTP-обработчик для проверки соединения с базой данных
type DBHandler struct {
	db     DB
	logger logger.Logger
}

// NewDBHandler создает новый хендлер для проверки БД
func NewDBHandler(database DB, log logger.Logger) *DBHandler {
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

	// Проверяем соединение с БД, передавая логгер явно
	if err := h.db.Ping(ctx, h.logger); err != nil {
		http.Error(w, "Ошибка проверки соединения с БД", http.StatusInternalServerError)
		return
	}

	// Соединение успешно
	w.WriteHeader(http.StatusOK)
}
