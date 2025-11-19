package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

// metricsService определяет интерфейс для работы с метриками.
type metricsService interface {
	UpdateMetricByPath(path string) error
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, delta int64) error
	UpdateMetrics(metrics []models.Metrics) error
	GetMetric(name, mType string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
}

// MetricsHandler — HTTP-обработчик метрик.
type MetricsHandler struct {
	service  metricsService
	template *template.Template
}

// NewMetricsHandler — конструктор.
// Принимает любой тип, реализующий интерфейс metricsService.
// service.MetricsService автоматически удовлетворяет этому интерфейсу.
func NewMetricsHandler(s metricsService) *MetricsHandler {
	// Загружаем шаблон один раз при создании хэндлера
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	templatePath := filepath.Join(dir, "metrics.gohtml")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		// В случае ошибки паникуем, так как без шаблона работа невозможна
		panic(fmt.Sprintf("failed to load template: %v", err))
	}

	return &MetricsHandler{
		service:  s,
		template: tmpl,
	}
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

// GetMetricHandler обрабатывает GET /value/{type}/{name}
// Возвращает значение метрики в текстовом виде
func (h *MetricsHandler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	metric, err := h.service.GetMetric(metricName, metricType)
	if err != nil {
		switch err {
		case service.ErrInvalidName:
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		case service.ErrInvalidType:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Формируем текстовое представление значения
	var value string
	switch metric.MType {
	case models.Gauge:
		if metric.Value != nil {
			value = fmt.Sprintf("%g", *metric.Value)
		}
	case models.Counter:
		if metric.Delta != nil {
			value = fmt.Sprintf("%d", *metric.Delta)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

// MetricView - структура для отображения метрики в шаблоне
type MetricView struct {
	Type  string
	Name  string
	Value string
}

// ListMetricsHandler обрабатывает GET /
// Возвращает HTML-страницу со списком всех метрик
func (h *MetricsHandler) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := h.service.GetAllMetrics()

	// Преобразуем метрики в структуры для отображения
	var views []MetricView
	for _, m := range metrics {
		view := MetricView{
			Type: m.MType,
			Name: m.ID,
		}

		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				view.Value = fmt.Sprintf("%.6f", *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				view.Value = fmt.Sprintf("%d", *m.Delta)
			}
		}

		views = append(views, view)
	}

	// Используем предварительно загруженный шаблон
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := h.template.Execute(w, views); err != nil {
		http.Error(w, "template execution error", http.StatusInternalServerError)
	}
}

// UpdateMetricJSONHandler обрабатывает POST /update
// Принимает метрику в формате JSON и сохраняет её
func (h *MetricsHandler) UpdateMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var metric models.Metrics

	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if metric.ID == "" || metric.MType == "" {
		http.Error(w, "Missing required fields: id or type", http.StatusBadRequest)
		return
	}

	// Проверяем тип метрики и наличие соответствующего значения
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			http.Error(w, "Missing value for gauge metric", http.StatusBadRequest)
			return
		}
		// Сохраняем gauge метрику
		if err := h.service.UpdateGauge(metric.ID, *metric.Value); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	case models.Counter:
		if metric.Delta == nil {
			http.Error(w, "Missing delta for counter metric", http.StatusBadRequest)
			return
		}
		// Сохраняем counter метрику
		if err := h.service.UpdateCounter(metric.ID, *metric.Delta); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	// Возвращаем обновлённую метрику
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metric)
}

// GetMetricJSONHandler обрабатывает POST /value
// Возвращает значение метрики в формате JSON
func (h *MetricsHandler) GetMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var request models.Metrics

	// Декодируем JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if request.ID == "" || request.MType == "" {
		http.Error(w, "Missing required fields: id or type", http.StatusBadRequest)
		return
	}

	// Получаем метрику из сервиса
	metric, err := h.service.GetMetric(request.ID, request.MType)
	if err != nil {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	// Возвращаем метрику в формате JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metric)
}

// UpdateMetricsBatchHandler обрабатывает POST /updates/
// Принимает множество метрик в формате JSON ([]Metrics) и сохраняет их
func (h *MetricsHandler) UpdateMetricsBatchHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type должен быть application/json", http.StatusBadRequest)
		return
	}

	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	// Проверяем, что массив не пустой
	if len(metrics) == 0 {
		http.Error(w, "Массив метрик пуст", http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей для каждой метрики
	for i, metric := range metrics {
		if metric.ID == "" || metric.MType == "" {
			http.Error(w, fmt.Sprintf("Пропущены обязательные поля: id или type в индексе %d", i), http.StatusBadRequest)
			return
		}

		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				http.Error(w, fmt.Sprintf("Пропущены значения для gauge метрики в индексе %d", i), http.StatusBadRequest)
				return
			}
		case models.Counter:
			if metric.Delta == nil {
				http.Error(w, fmt.Sprintf("Пропущены значения для counter метрики в индексе %d", i), http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, fmt.Sprintf("Пропущен тип метрики в индексе %d: %s", i, metric.MType), http.StatusBadRequest)
			return
		}
	}

	if err := h.service.UpdateMetrics(metrics); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при обновлении метрик: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
