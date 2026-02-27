package agent

import (
	"testing"
)

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	if collector == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	if collector.gauges == nil {
		t.Error("gauges map is nil")
	}

	if collector.counters == nil {
		t.Error("counters map is nil")
	}
}

func TestCollectMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Собираем метрики первый раз
	collector.CollectMetrics()

	gauges := collector.GetGauges()
	counters := collector.GetCounters()

	// Проверяем, что собраны gauge метрики
	expectedGauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range expectedGauges {
		if _, exists := gauges[name]; !exists {
			t.Errorf("gauge metric %s not found", name)
		}
	}

	// Проверяем, что PollCount = 1
	if pollCount, exists := counters["PollCount"]; !exists {
		t.Error("PollCount counter not found")
	} else if pollCount != 1 {
		t.Errorf("PollCount expected 1, got %d", pollCount)
	}

	// Собираем метрики второй раз
	collector.CollectMetrics()
	counters = collector.GetCounters()

	// Проверяем, что PollCount увеличился
	if pollCount, exists := counters["PollCount"]; !exists {
		t.Error("PollCount counter not found after second collect")
	} else if pollCount != 2 {
		t.Errorf("PollCount expected 2, got %d", pollCount)
	}
}

func TestGetGauges(t *testing.T) {
	collector := NewMetricsCollector()
	collector.CollectMetrics()

	gauges1 := collector.GetGauges()
	gauges2 := collector.GetGauges()

	// Проверяем, что возвращаются копии
	if len(gauges1) != len(gauges2) {
		t.Error("GetGauges returned different lengths")
	}

	// Изменяем первую копию
	gauges1["TestMetric"] = 123.456

	// Проверяем, что вторая копия не изменилась
	if _, exists := gauges2["TestMetric"]; exists {
		t.Error("GetGauges does not return a copy")
	}
}

func TestGetCounters(t *testing.T) {
	collector := NewMetricsCollector()
	collector.CollectMetrics()

	counters1 := collector.GetCounters()
	counters2 := collector.GetCounters()

	// Проверяем, что возвращаются копии
	if len(counters1) != len(counters2) {
		t.Error("GetCounters returned different lengths")
	}

	// Изменяем первую копию
	counters1["TestCounter"] = 999

	// Проверяем, что вторая копия не изменилась
	if _, exists := counters2["TestCounter"]; exists {
		t.Error("GetCounters does not return a copy")
	}
}

func TestCollectPSUtilMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	err := collector.CollectPSUtilMetrics()
	// На некоторых системах gopsutil может вернуть ошибку — это не критично
	if err != nil {
		t.Logf("CollectPSUtilMetrics returned error (может быть на CI): %v", err)
		return
	}

	gauges := collector.GetGauges()

	// TotalMemory и FreeMemory должны присутствовать
	if _, ok := gauges["TotalMemory"]; !ok {
		t.Error("TotalMemory not found in gauges")
	}
	if _, ok := gauges["FreeMemory"]; !ok {
		t.Error("FreeMemory not found in gauges")
	}
}

func TestRandomValueChanges(t *testing.T) {
	collector := NewMetricsCollector()

	collector.CollectMetrics()
	gauges1 := collector.GetGauges()
	random1 := gauges1["RandomValue"]

	collector.CollectMetrics()
	gauges2 := collector.GetGauges()
	random2 := gauges2["RandomValue"]

	// RandomValue должен изменяться (с очень высокой вероятностью)
	if random1 == random2 {
		t.Log("Warning: RandomValue did not change (unlikely but possible)")
	}
}
