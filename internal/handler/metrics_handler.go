package handler

import (
	"fmt"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

// MetricsHandler — HTTP-обработчик метрик.
type MetricsHandler struct {
	service *service.MetricsService
}

// NewMetricsHandler — конструктор.
func NewMetricsHandler(s *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{service: s}
}

// UpdateMetricHandler обрабатывает POST /update/{type}/{name}/{value}
// Теперь использует chi роутер для извлечения параметров из URL
func (h *MetricsHandler) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметры через chi.URLParam
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	// Формируем путь для совместимости с существующей бизнес-логикой
	path := fmt.Sprintf("%s/%s/%s", metricType, metricName, metricValue)
	
	err := h.service.UpdateMetricByPath(path)
	if err != nil {
		switch err {
		case service.ErrInvalidName:
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		case service.ErrInvalidType, service.ErrInvalidValue:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
