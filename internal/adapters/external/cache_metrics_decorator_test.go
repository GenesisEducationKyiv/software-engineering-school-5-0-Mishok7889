package external

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/ports"
)

// TestCacheMetricsDecorator_WithMemoryCache tests metrics decorator with memory cache
func TestCacheMetricsDecorator_WithMemoryCache(t *testing.T) {
	memoryClient := NewMemoryCacheClient()
	decorator := NewCacheMetricsDecorator(memoryClient)

	ctx := context.Background()

	// Clear and verify initial stats
	require.NoError(t, decorator.Clear(ctx))
	stats := decorator.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)

	t.Run("CacheHit", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := time.Minute

		// Set value
		err := decorator.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		// Get existing value (should be a hit)
		retrieved, err := decorator.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Verify hit was recorded
		stats := decorator.GetStats()
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, float64(1), stats.HitRatio)
	})

	t.Run("CacheMiss", func(t *testing.T) {
		// Get non-existent key (should be a miss)
		_, err := decorator.Get(ctx, "non-existent-key")
		assert.Error(t, err)
		assert.True(t, ports.IsNotFoundError(err))

		// Verify miss was recorded
		stats := decorator.GetStats()
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, int64(2), stats.TotalOps)
		assert.Equal(t, float64(0.5), stats.HitRatio)
	})

	t.Run("OperationMetrics", func(t *testing.T) {
		// Perform various operations
		key := "metrics-key"
		value := []byte("metrics-value")

		err := decorator.Set(ctx, key, value, time.Minute)
		require.NoError(t, err)

		_, err = decorator.Get(ctx, key)
		require.NoError(t, err)

		exists, err := decorator.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		err = decorator.Delete(ctx, key)
		require.NoError(t, err)

		// Check operation metrics
		opMetrics := decorator.GetOperationMetrics()
		assert.Contains(t, opMetrics, "set")
		assert.Contains(t, opMetrics, "get")
		assert.Contains(t, opMetrics, "exists")
		assert.Contains(t, opMetrics, "delete")

		// Verify each operation has positive counts and reasonable latencies
		for op, metrics := range opMetrics {
			assert.Greater(t, metrics.Count, int64(0), "Operation %s should have positive count", op)
			assert.GreaterOrEqual(t, metrics.TotalLatency, time.Duration(0), "Operation %s should have non-negative total latency", op)
			// For very fast operations, latency might be 0, so we only assert positive if total latency > 0
			if metrics.TotalLatency > 0 {
				assert.Greater(t, metrics.AvgLatency, time.Duration(0), "Operation %s should have positive avg latency when total > 0", op)
			} else {
				// For extremely fast operations, avg latency can be 0
				assert.GreaterOrEqual(t, metrics.AvgLatency, time.Duration(0), "Operation %s avg latency should be non-negative", op)
			}
			// Min and max latency should be non-negative
			assert.GreaterOrEqual(t, metrics.MinLatency, time.Duration(0), "Operation %s should have non-negative min latency", op)
			assert.GreaterOrEqual(t, metrics.MaxLatency, time.Duration(0), "Operation %s should have non-negative max latency", op)
		}
	})

	t.Run("ManualRecording", func(t *testing.T) {
		initialStats := decorator.GetStats()

		// Manually record hit and miss
		decorator.RecordHit()
		decorator.RecordMiss()
		decorator.RecordOperation("manual", 100*time.Millisecond)

		// Verify manual recordings
		stats := decorator.GetStats()
		assert.Equal(t, initialStats.Hits+1, stats.Hits)
		assert.Equal(t, initialStats.Misses+1, stats.Misses)

		opMetrics := decorator.GetOperationMetrics()
		assert.Contains(t, opMetrics, "manual")
		assert.Equal(t, int64(1), opMetrics["manual"].Count)
	})
}

// TestCacheMetricsDecorator_InterfaceCompliance tests interface implementations
func TestCacheMetricsDecorator_InterfaceCompliance(t *testing.T) {
	memoryClient := NewMemoryCacheClient()
	decorator := NewCacheMetricsDecorator(memoryClient)

	// Test CacheProvider interface compliance
	var _ ports.CacheProvider = decorator

	// Test CacheMetrics interface compliance
	var _ ports.CacheMetrics = decorator
}

// TestCacheMetricsDecorator_ErrorHandling tests error handling
func TestCacheMetricsDecorator_ErrorHandling(t *testing.T) {
	memoryClient := NewMemoryCacheClient()
	decorator := NewCacheMetricsDecorator(memoryClient)

	ctx := context.Background()

	t.Run("ValidationErrors", func(t *testing.T) {
		// Empty key should cause validation error
		_, err := decorator.Get(ctx, "")
		assert.Error(t, err)

		// Nil value should cause validation error
		err = decorator.Set(ctx, "key", nil, time.Minute)
		assert.Error(t, err)

		// Zero TTL should cause validation error
		err = decorator.Set(ctx, "key", []byte("value"), 0)
		assert.Error(t, err)

		// Errors should not affect hit/miss counts since they occur before cache operations
		stats := decorator.GetStats()
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
	})
}

// TestCacheMetricsDecorator_ConcurrentAccess tests concurrent operations
func TestCacheMetricsDecorator_ConcurrentAccess(t *testing.T) {
	memoryClient := NewMemoryCacheClient()
	decorator := NewCacheMetricsDecorator(memoryClient)

	ctx := context.Background()
	const numGoroutines = 10
	const operationsPerGoroutine = 100

	// Set up initial data
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		err := decorator.Set(ctx, key, []byte("value"), time.Minute)
		require.NoError(t, err)
	}

	// Run concurrent operations
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				key := string(rune('a' + (j % 10)))

				// Mix of hits and misses
				switch j % 3 {
				case 0:
					// Hit - existing key
					_, _ = decorator.Get(ctx, key)
				case 1:
					// Miss - non-existent key
					_, _ = decorator.Get(ctx, "non-existent")
				default:
					// Set operation
					_ = decorator.Set(ctx, key, []byte("updated"), time.Minute)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify that stats are consistent (no race conditions)
	stats := decorator.GetStats()
	assert.GreaterOrEqual(t, stats.TotalOps, int64(0))
	assert.Equal(t, stats.TotalOps, stats.Hits+stats.Misses)

	if stats.TotalOps > 0 {
		expectedHitRatio := float64(stats.Hits) / float64(stats.TotalOps)
		assert.Equal(t, expectedHitRatio, stats.HitRatio)
	}

	// Verify operation metrics are present
	opMetrics := decorator.GetOperationMetrics()
	assert.Contains(t, opMetrics, "get")
	assert.Contains(t, opMetrics, "set")
}
