package infrastructure

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/ports"
)

// CounterMetric represents a single counter with labels
type CounterMetric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Value  int64             `json:"value"`
}

// MetricsCollectorAdapter implements the MetricsCollector interface for HTTPServerAdapter
// This adapter aggregates metrics from various domain services
type MetricsCollectorAdapter struct {
	weatherMetrics ports.WeatherMetrics
	cacheMetrics   ports.CacheMetrics
	counters       map[string]*CounterMetric
	countersMutex  sync.RWMutex
	lastUpdated    time.Time
}

// MetricsCollectorConfig holds configuration for creating the metrics collector
type MetricsCollectorConfig struct {
	WeatherMetrics ports.WeatherMetrics
	CacheMetrics   ports.CacheMetrics
}

// NewMetricsCollectorAdapter creates a new metrics collector adapter
func NewMetricsCollectorAdapter(config MetricsCollectorConfig) *MetricsCollectorAdapter {
	return &MetricsCollectorAdapter{
		weatherMetrics: config.WeatherMetrics,
		cacheMetrics:   config.CacheMetrics,
		counters:       make(map[string]*CounterMetric),
		lastUpdated:    time.Now(),
	}
}

// IncrementCounter increments a named counter with labels
func (m *MetricsCollectorAdapter) IncrementCounter(name string, labels map[string]string) {
	m.countersMutex.Lock()
	defer m.countersMutex.Unlock()

	counterKey := m.generateCounterKey(name, labels)

	if counter, exists := m.counters[counterKey]; exists {
		counter.Value++
	} else {
		m.counters[counterKey] = &CounterMetric{
			Name:   name,
			Labels: m.copyLabels(labels),
			Value:  1,
		}
	}

	m.lastUpdated = time.Now()
}

// GetMetrics returns aggregated metrics from all monitored services
func (m *MetricsCollectorAdapter) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"weather": m.weatherMetrics.GetProviderInfo(),
	}

	if cacheStats, err := m.weatherMetrics.GetCacheMetrics(); err == nil {
		cacheMetrics := map[string]interface{}{
			"hits":      cacheStats.Hits,
			"misses":    cacheStats.Misses,
			"total_ops": cacheStats.TotalOps,
			"hit_ratio": cacheStats.HitRatio,
			"updated":   cacheStats.LastUpdated,
		}

		if m.cacheMetrics != nil {
			if operationMetrics := m.cacheMetrics.GetOperationMetrics(); len(operationMetrics) > 0 {
				cacheMetrics["operations"] = operationMetrics
			}
		}

		metrics["cache"] = cacheMetrics
	}

	// Add counter metrics
	m.countersMutex.RLock()
	counterMetrics := make(map[string]interface{})
	if len(m.counters) > 0 {
		countersList := make([]*CounterMetric, 0, len(m.counters))
		for _, counter := range m.counters {
			countersList = append(countersList, counter)
		}
		counterMetrics["counters"] = countersList
		counterMetrics["last_updated"] = m.lastUpdated
		counterMetrics["total_counters"] = len(m.counters)
	}
	m.countersMutex.RUnlock()

	if len(counterMetrics) > 0 {
		metrics["api"] = counterMetrics
	}

	return metrics, nil
}

// generateCounterKey creates a unique key for a counter based on name and labels
func (m *MetricsCollectorAdapter) generateCounterKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		for k, v := range labels {
			key += "|" + k + "=" + v
		}
	}
	return key
}

// copyLabels creates a deep copy of the labels map
func (m *MetricsCollectorAdapter) copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// GetCounterByName returns all counters with the specified name
func (m *MetricsCollectorAdapter) GetCounterByName(name string) []*CounterMetric {
	m.countersMutex.RLock()
	defer m.countersMutex.RUnlock()

	var result []*CounterMetric
	for _, counter := range m.counters {
		if counter.Name == name {
			result = append(result, counter)
		}
	}
	return result
}

// GetCounterValue returns the value of a specific counter
func (m *MetricsCollectorAdapter) GetCounterValue(name string, labels map[string]string) int64 {
	m.countersMutex.RLock()
	defer m.countersMutex.RUnlock()

	counterKey := m.generateCounterKey(name, labels)
	if counter, exists := m.counters[counterKey]; exists {
		return counter.Value
	}
	return 0
}

// ResetCounters clears all counter metrics
func (m *MetricsCollectorAdapter) ResetCounters() {
	m.countersMutex.Lock()
	defer m.countersMutex.Unlock()

	m.counters = make(map[string]*CounterMetric)
	m.lastUpdated = time.Now()
}
