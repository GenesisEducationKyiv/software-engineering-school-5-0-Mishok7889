package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheMetrics(t *testing.T) {
	metrics := NewCacheMetrics("test")

	t.Run("Initial state", func(t *testing.T) {
		stats := metrics.GetStats()
		assert.Equal(t, "test", stats["cache_type"])
		assert.Equal(t, int64(0), stats["hits"])
		assert.Equal(t, int64(0), stats["misses"])
		assert.Equal(t, int64(0), stats["total"])
	})

	t.Run("Record hits and misses", func(t *testing.T) {
		metrics.RecordHit()
		metrics.RecordHit()
		metrics.RecordMiss()

		stats := metrics.GetStats()
		assert.Equal(t, int64(2), stats["hits"])
		assert.Equal(t, int64(1), stats["misses"])
		assert.Equal(t, int64(3), stats["total"])
		assert.Equal(t, float64(2)/float64(3), stats["hit_ratio"])
	})

	t.Run("Hit ratio calculation", func(t *testing.T) {
		newMetrics := NewCacheMetrics("ratio_test")

		for i := 0; i < 7; i++ {
			newMetrics.RecordHit()
		}
		for i := 0; i < 3; i++ {
			newMetrics.RecordMiss()
		}

		stats := newMetrics.GetStats()
		assert.Equal(t, int64(7), stats["hits"])
		assert.Equal(t, int64(3), stats["misses"])
		assert.Equal(t, int64(10), stats["total"])
		assert.Equal(t, 0.7, stats["hit_ratio"])
	})

	t.Run("Record latency", func(t *testing.T) {
		metrics.RecordLatency("get", 0.001)
		metrics.RecordLatency("set", 0.002)
	})
}
