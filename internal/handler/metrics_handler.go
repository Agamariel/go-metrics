// Package handler предоставляет HTTP-обработчики для сервера метрик.
//
// Пакет содержит обработчики для следующих эндпоинтов:
//   - POST /update/{type}/{name}/{value} — обновление метрики через URL-параметры
//   - POST /update — обновление метрики через JSON
//   - POST /updates/ — пакетное обновление метрик
//   - GET /value/{type}/{name} — получение значения метрики в текстовом виде
//   - POST /value — получение значения метрики в формате JSON
//   - GET / — HTML-страница со списком всех метрик
//   - GET /ping — проверка соединения с базой данных
//
// # Пример использования
//
//	storage := repository.NewMemStorage()
//	svc := service.NewMetricsService(storage)
//	handler := handler.NewMetricsHandler(svc, nil)
//
//	r := chi.NewRouter()
//	r.Post("/update/{type}/{name}/{value}", handler.UpdateMetricHandler)
//	r.Get("/value/{type}/{name}", handler.GetMetricHandler)
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Agamariel/go-metrics/internal/audit"
	"github.com/Agamariel/go-metrics/internal/models"
	"github.com/Agamariel/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

// metricsService определяет интерфейс для работы с метриками.
// Позволяет использовать любую реализацию сервиса метрик для тестирования.
type metricsService interface {
	// UpdateMetricByPath обновляет метрику по данным из URL-пути.
	UpdateMetricByPath(ctx context.Context, path string) error
	// UpdateGauge обновляет gauge-метрику.
	UpdateGauge(ctx context.Context, name string, value float64) error
	// UpdateCounter обновляет counter-метрику.
	UpdateCounter(ctx context.Context, name string, delta int64) error
	// UpdateMetrics пакетно обновляет несколько метрик.
	UpdateMetrics(ctx context.Context, metrics []models.Metrics) error
	// GetMetric возвращает метрику по имени и типу.
	GetMetric(ctx context.Context, name, mType string) (models.Metrics, error)
	// GetAllMetrics возвращает все сохранённые метрики.
	GetAllMetrics(ctx context.Context) ([]models.Metrics, error)
}

// MetricsHandler предоставляет HTTP-обработчики для работы с метриками.
//
// Обработчик использует chi-роутер для извлечения URL-параметров
// и поддерживает как текстовый, так и JSON-формат обмена данными.
//
// При создании загружает HTML-шаблон для отображения списка метрик.
// Опционально поддерживает отправку событий аудита через Publisher.
type MetricsHandler struct {
	service   metricsService
	template  *template.Template
	publisher *audit.Publisher
}

// NewMetricsHandler создаёт новый обработчик метрик.
//
// Параметры:
//   - s: сервис метрик, реализующий интерфейс metricsService
//   - publisher: издатель событий аудита (может быть nil, если аудит отключен)
//
// Функция загружает HTML-шаблон metrics.gohtml из директории пакета.
// При ошибке загрузки шаблона вызывает panic.
func NewMetricsHandler(s metricsService, publisher *audit.Publisher) *MetricsHandler {
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
		service:   s,
		template:  tmpl,
		publisher: publisher,
	}
}

// UpdateMetricHandler обрабатывает POST /update/{type}/{name}/{value}.
//
// Эндпоинт принимает параметры метрики через URL:
//   - type: тип метрики ("gauge" или "counter")
//   - name: имя метрики
//   - value: значение (float64 для gauge, int64 для counter)
//
// Коды ответа:
//   - 200 OK: метрика успешно обновлена
//   - 400 Bad Request: неверный тип или значение метрики
//   - 404 Not Found: пустое имя метрики
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *MetricsHandler) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметры через chi.URLParam
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	// Формируем путь для совместимости с существующей бизнес-логикой
	path := fmt.Sprintf("%s/%s/%s", metricType, metricName, metricValue)

	err := h.service.UpdateMetricByPath(r.Context(), path)
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

	// Отправляем событие аудита
	if h.publisher != nil {
		event := audit.Event{
			Timestamp: time.Now().Unix(),
			Metrics:   []string{metricName},
			IPAddress: r.RemoteAddr,
		}
		h.publisher.NotifyAll(r.Context(), event)
	}

	w.WriteHeader(http.StatusOK)
}

// GetMetricHandler обрабатывает GET /value/{type}/{name}.
//
// Возвращает значение метрики в текстовом виде (Content-Type: text/plain).
// Формат вывода:
//   - gauge: число с плавающей точкой (например, "123.456")
//   - counter: целое число (например, "42")
//
// Коды ответа:
//   - 200 OK: метрика найдена, значение в теле ответа
//   - 400 Bad Request: неверный тип метрики
//   - 404 Not Found: метрика не найдена
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *MetricsHandler) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	metric, err := h.service.GetMetric(r.Context(), metricName, metricType)
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

// MetricView представляет метрику для отображения в HTML-шаблоне.
//
// Используется в ListMetricsHandler для рендеринга списка метрик.
type MetricView struct {
	// Type — тип метрики ("gauge" или "counter").
	Type string
	// Name — имя метрики.
	Name string
	// Value — строковое представление значения метрики.
	Value string
}

// ListMetricsHandler обрабатывает GET /.
//
// Возвращает HTML-страницу со списком всех сохранённых метрик
// (Content-Type: text/html; charset=utf-8).
//
// Коды ответа:
//   - 200 OK: страница успешно сформирована
//   - 500 Internal Server Error: ошибка получения метрик или рендеринга шаблона
func (h *MetricsHandler) ListMetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	// Преобразуем метрики в структуры для отображения
	// Предварительно выделяем память для среза
	views := make([]MetricView, 0, len(metrics))
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

// UpdateMetricJSONHandler обрабатывает POST /update.
//
// Принимает метрику в формате JSON и сохраняет её.
// Требует заголовок Content-Type: application/json.
//
// Формат запроса:
//
//	{
//	    "id": "metric_name",
//	    "type": "gauge" | "counter",
//	    "value": 123.45,  // для gauge
//	    "delta": 10       // для counter
//	}
//
// Коды ответа:
//   - 200 OK: метрика обновлена, обновлённая метрика в теле ответа
//   - 400 Bad Request: неверный JSON, Content-Type или отсутствуют обязательные поля
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
		if err := h.service.UpdateGauge(r.Context(), metric.ID, *metric.Value); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	case models.Counter:
		if metric.Delta == nil {
			http.Error(w, "Missing delta for counter metric", http.StatusBadRequest)
			return
		}
		// Сохраняем counter метрику
		if err := h.service.UpdateCounter(r.Context(), metric.ID, *metric.Delta); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	// Отправляем событие аудита
	if h.publisher != nil {
		event := audit.Event{
			Timestamp: time.Now().Unix(),
			Metrics:   []string{metric.ID},
			IPAddress: r.RemoteAddr,
		}
		h.publisher.NotifyAll(r.Context(), event)
	}

	// Возвращаем обновлённую метрику
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metric)
}

// GetMetricJSONHandler обрабатывает POST /value.
//
// Принимает запрос с идентификатором метрики в формате JSON
// и возвращает полную информацию о метрике.
// Требует заголовок Content-Type: application/json.
//
// Формат запроса:
//
//	{
//	    "id": "metric_name",
//	    "type": "gauge" | "counter"
//	}
//
// Формат ответа:
//
//	{
//	    "id": "metric_name",
//	    "type": "gauge",
//	    "value": 123.45
//	}
//
// Коды ответа:
//   - 200 OK: метрика найдена
//   - 400 Bad Request: неверный JSON или Content-Type
//   - 404 Not Found: метрика не найдена
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
	metric, err := h.service.GetMetric(r.Context(), request.ID, request.MType)
	if err != nil {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	// Возвращаем метрику в формате JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metric)
}

// UpdateMetricsBatchHandler обрабатывает POST /updates/.
//
// Принимает массив метрик в формате JSON и сохраняет их атомарно.
// Это наиболее эффективный способ отправки нескольких метрик одновременно.
// Требует заголовок Content-Type: application/json.
//
// Формат запроса:
//
//	[
//	    {"id": "metric1", "type": "gauge", "value": 123.45},
//	    {"id": "metric2", "type": "counter", "delta": 10}
//	]
//
// Коды ответа:
//   - 200 OK: все метрики успешно обновлены
//   - 400 Bad Request: неверный JSON, пустой массив или ошибки валидации
//   - 500 Internal Server Error: ошибка при сохранении метрик
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

	if err := h.service.UpdateMetrics(r.Context(), metrics); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при обновлении метрик: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем событие аудита
	if h.publisher != nil {
		// Собираем имена всех метрик
		metricNames := make([]string, len(metrics))
		for i, m := range metrics {
			metricNames[i] = m.ID
		}

		event := audit.Event{
			Timestamp: time.Now().Unix(),
			Metrics:   metricNames,
			IPAddress: r.RemoteAddr,
		}
		h.publisher.NotifyAll(r.Context(), event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
