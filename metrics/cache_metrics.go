package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type CacheMetricsCollector struct {
	Hits     *prometheus.CounterVec
	Misses   *prometheus.CounterVec
	Requests *prometheus.CounterVec
	Latency  *prometheus.HistogramVec
	HitRatio *prometheus.GaugeVec
}

var globalCollector *CacheMetricsCollector

func getCollector() *CacheMetricsCollector {
	if globalCollector == nil {
		globalCollector = &CacheMetricsCollector{
			Hits: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "weather_cache_hits_total",
					Help: "The total number of cache hits",
				},
				[]string{"cache_type"},
			),
			Misses: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "weather_cache_misses_total",
					Help: "The total number of cache misses",
				},
				[]string{"cache_type"},
			),
			Requests: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "weather_cache_requests_total",
					Help: "The total number of cache requests",
				},
				[]string{"cache_type"},
			),
			Latency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "weather_cache_duration_seconds",
					Help:    "Cache operation duration in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"cache_type", "operation"},
			),
			HitRatio: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "weather_cache_hit_ratio",
					Help: "Cache hit ratio (hits/total requests)",
				},
				[]string{"cache_type"},
			),
		}
	}
	return globalCollector
}

type CacheMetrics struct {
	cacheType string
	hits      int64
	misses    int64
	total     int64
	collector *CacheMetricsCollector
	mu        sync.RWMutex
}

func NewCacheMetrics(cacheType string) *CacheMetrics {
	return &CacheMetrics{
		cacheType: cacheType,
		collector: getCollector(),
	}
}

func (m *CacheMetrics) RecordHit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hits++
	m.total++
	m.collector.Hits.WithLabelValues(m.cacheType).Inc()
	m.collector.Requests.WithLabelValues(m.cacheType).Inc()
	m.updateHitRatio()
}

func (m *CacheMetrics) RecordMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.misses++
	m.total++
	m.collector.Misses.WithLabelValues(m.cacheType).Inc()
	m.collector.Requests.WithLabelValues(m.cacheType).Inc()
	m.updateHitRatio()
}

func (m *CacheMetrics) RecordLatency(operation string, duration float64) {
	m.collector.Latency.WithLabelValues(m.cacheType, operation).Observe(duration)
}

// updateHitRatio updates the Prometheus hit ratio gauge.
// Must be called while holding the mutex.
func (m *CacheMetrics) updateHitRatio() {
	if m.total > 0 {
		ratio := float64(m.hits) / float64(m.total)
		m.collector.HitRatio.WithLabelValues(m.cacheType).Set(ratio)
	}
}

func (m *CacheMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var hitRatio float64
	if m.total > 0 {
		hitRatio = float64(m.hits) / float64(m.total)
	}

	return map[string]interface{}{
		"cache_type": m.cacheType,
		"hits":       m.hits,
		"misses":     m.misses,
		"total":      m.total,
		"hit_ratio":  hitRatio,
	}
}
