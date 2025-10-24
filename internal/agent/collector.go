package agent

import (
	"math/rand"
	"runtime"
	"sync"
)

// MetricsCollector собирает метрики из runtime
type MetricsCollector struct {
	mu          sync.RWMutex
	gauges      map[string]float64
	counters    map[string]int64
	pollCount   int64
	randomValue float64
}

// NewMetricsCollector создает новый коллектор метрик
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// CollectMetrics собирает метрики из runtime.MemStats
func (c *MetricsCollector) CollectMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Собираем gauge метрики из runtime
	c.gauges["Alloc"] = float64(memStats.Alloc)
	c.gauges["BuckHashSys"] = float64(memStats.BuckHashSys)
	c.gauges["Frees"] = float64(memStats.Frees)
	c.gauges["GCCPUFraction"] = memStats.GCCPUFraction
	c.gauges["GCSys"] = float64(memStats.GCSys)
	c.gauges["HeapAlloc"] = float64(memStats.HeapAlloc)
	c.gauges["HeapIdle"] = float64(memStats.HeapIdle)
	c.gauges["HeapInuse"] = float64(memStats.HeapInuse)
	c.gauges["HeapObjects"] = float64(memStats.HeapObjects)
	c.gauges["HeapReleased"] = float64(memStats.HeapReleased)
	c.gauges["HeapSys"] = float64(memStats.HeapSys)
	c.gauges["LastGC"] = float64(memStats.LastGC)
	c.gauges["Lookups"] = float64(memStats.Lookups)
	c.gauges["MCacheInuse"] = float64(memStats.MCacheInuse)
	c.gauges["MCacheSys"] = float64(memStats.MCacheSys)
	c.gauges["MSpanInuse"] = float64(memStats.MSpanInuse)
	c.gauges["MSpanSys"] = float64(memStats.MSpanSys)
	c.gauges["Mallocs"] = float64(memStats.Mallocs)
	c.gauges["NextGC"] = float64(memStats.NextGC)
	c.gauges["NumForcedGC"] = float64(memStats.NumForcedGC)
	c.gauges["NumGC"] = float64(memStats.NumGC)
	c.gauges["OtherSys"] = float64(memStats.OtherSys)
	c.gauges["PauseTotalNs"] = float64(memStats.PauseTotalNs)
	c.gauges["StackInuse"] = float64(memStats.StackInuse)
	c.gauges["StackSys"] = float64(memStats.StackSys)
	c.gauges["Sys"] = float64(memStats.Sys)
	c.gauges["TotalAlloc"] = float64(memStats.TotalAlloc)

	// Добавляем RandomValue
	c.randomValue = rand.Float64()
	c.gauges["RandomValue"] = c.randomValue

	// Увеличиваем PollCount
	c.pollCount++
	c.counters["PollCount"] = c.pollCount
}

// GetGauges возвращает копию gauge метрик
func (c *MetricsCollector) GetGauges() map[string]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]float64, len(c.gauges))
	for k, v := range c.gauges {
		result[k] = v
	}
	return result
}

// GetCounters возвращает копию counter метрик
func (c *MetricsCollector) GetCounters() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]int64, len(c.counters))
	for k, v := range c.counters {
		result[k] = v
	}
	return result
}
