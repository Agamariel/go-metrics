package handler

import (
	"net/http"
	"strings"

	"github.com/Agamariel/go-metrics/internal/service"
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
func (h *MetricsHandler) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/update/")
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
