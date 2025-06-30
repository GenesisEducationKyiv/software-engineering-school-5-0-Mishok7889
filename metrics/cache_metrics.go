package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_cache_hits_total",
			Help: "The total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_cache_misses_total",
			Help: "The total number of cache misses",
		},
		[]string{"cache_type"},
	)

	CacheRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_cache_requests_total",
			Help: "The total number of cache requests",
		},
		[]string{"cache_type"},
	)

	CacheLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "weather_cache_duration_seconds",
			Help:    "Cache operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"cache_type", "operation"},
	)

	CacheHitRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "weather_cache_hit_ratio",
			Help: "Cache hit ratio (hits/total requests)",
		},
		[]string{"cache_type"},
	)
)

type CacheMetrics struct {
	cacheType string
	hits      int64
	misses    int64
	total     int64
}

func NewCacheMetrics(cacheType string) *CacheMetrics {
	return &CacheMetrics{
		cacheType: cacheType,
	}
}

func (m *CacheMetrics) RecordHit() {
	m.hits++
	m.total++
	CacheHits.WithLabelValues(m.cacheType).Inc()
	CacheRequests.WithLabelValues(m.cacheType).Inc()
	m.updateHitRatio()
}

func (m *CacheMetrics) RecordMiss() {
	m.misses++
	m.total++
	CacheMisses.WithLabelValues(m.cacheType).Inc()
	CacheRequests.WithLabelValues(m.cacheType).Inc()
	m.updateHitRatio()
}

func (m *CacheMetrics) RecordLatency(operation string, duration float64) {
	CacheLatency.WithLabelValues(m.cacheType, operation).Observe(duration)
}

func (m *CacheMetrics) updateHitRatio() {
	if m.total > 0 {
		ratio := float64(m.hits) / float64(m.total)
		CacheHitRatio.WithLabelValues(m.cacheType).Set(ratio)
	}
}

func (m *CacheMetrics) GetStats() map[string]interface{} {
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
