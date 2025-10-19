package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Agamariel/go-metrics/internal/models"
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

	// HTML шаблон для отображения метрик
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Metrics</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        h1 {
            color: #333;
        }
        table {
            border-collapse: collapse;
            width: 100%;
            max-width: 800px;
            background-color: white;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        th, td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        th {
            background-color: #4CAF50;
            color: white;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .gauge {
            color: #2196F3;
        }
        .counter {
            color: #FF9800;
        }
    </style>
</head>
<body>
    <h1>Метрики сервера</h1>
    <table>
        <thead>
            <tr>
                <th>Тип</th>
                <th>Имя</th>
                <th>Значение</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td class="{{.Type}}">{{.Type}}</td>
                <td>{{.Name}}</td>
                <td>{{.Value}}</td>
            </tr>
            {{else}}
            <tr>
                <td colspan="3" style="text-align: center;">Нет доступных метрик</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>`

	t, err := template.New("metrics").Parse(tmpl)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	
	if err := t.Execute(w, views); err != nil {
		http.Error(w, "template execution error", http.StatusInternalServerError)
	}
}
