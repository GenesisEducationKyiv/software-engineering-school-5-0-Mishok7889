package infrastructure

import (
	"context"

	"weatherapi.app/internal/ports"
)

// MetricsCollectorAdapter implements the MetricsCollector interface for HTTPServerAdapter
// This adapter aggregates metrics from various domain services
type MetricsCollectorAdapter struct {
	weatherMetrics ports.WeatherMetrics
	cacheMetrics   ports.CacheMetrics
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
	}
}

// IncrementCounter increments a named counter with labels
func (m *MetricsCollectorAdapter) IncrementCounter(name string, labels map[string]string) {
	// Simple implementation - could be extended to actually track counters
	// In a production system, this might integrate with Prometheus, StatsD, etc.
}

// GetMetrics returns aggregated metrics from all monitored services
func (m *MetricsCollectorAdapter) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"weather": m.weatherMetrics.GetProviderInfo(),
	}

	if cacheStats, err := m.weatherMetrics.GetCacheMetrics(); err == nil {
		metrics["cache"] = map[string]interface{}{
			"hits":      cacheStats.Hits,
			"misses":    cacheStats.Misses,
			"total_ops": cacheStats.TotalOps,
			"hit_ratio": cacheStats.HitRatio,
			"updated":   cacheStats.LastUpdated,
		}
	}

	return metrics, nil
}
