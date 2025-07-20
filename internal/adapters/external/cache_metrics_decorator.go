package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/ports"
)

// CacheMetricsDecorator decorates any CacheProvider with metrics tracking
type CacheMetricsDecorator struct {
	provider ports.CacheProvider
	stats    struct {
		hits       int64
		misses     int64
		operations map[string]*cacheOperationMetrics
		mutex      sync.RWMutex
	}
}

type cacheOperationMetrics struct {
	count        int64
	totalLatency time.Duration
	avgLatency   time.Duration
	maxLatency   time.Duration
	minLatency   time.Duration
}

// NewCacheMetricsDecorator creates a new metrics decorator for any cache provider
func NewCacheMetricsDecorator(provider ports.CacheProvider) *CacheMetricsDecorator {
	return &CacheMetricsDecorator{
		provider: provider,
		stats: struct {
			hits       int64
			misses     int64
			operations map[string]*cacheOperationMetrics
			mutex      sync.RWMutex
		}{
			operations: make(map[string]*cacheOperationMetrics),
		},
	}
}

// GetProvider returns the underlying cache provider
func (d *CacheMetricsDecorator) GetProvider() ports.CacheProvider {
	return d.provider
}

// Get retrieves a value from cache with metrics tracking
func (d *CacheMetricsDecorator) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		d.RecordOperation("get", time.Since(start))
	}()

	data, err := d.provider.Get(ctx, key)
	if err != nil {
		if ports.IsNotFoundError(err) {
			d.RecordMiss()
		}
		return nil, err
	}

	d.RecordHit()
	return data, nil
}

// Set stores a value in cache with metrics tracking
func (d *CacheMetricsDecorator) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		d.RecordOperation("set", time.Since(start))
	}()

	return d.provider.Set(ctx, key, value, ttl)
}

// Delete removes a value from cache with metrics tracking
func (d *CacheMetricsDecorator) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		d.RecordOperation("delete", time.Since(start))
	}()

	return d.provider.Delete(ctx, key)
}

// Exists checks if a key exists in cache with metrics tracking
func (d *CacheMetricsDecorator) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		d.RecordOperation("exists", time.Since(start))
	}()

	return d.provider.Exists(ctx, key)
}

// Clear removes all keys from cache with metrics tracking
func (d *CacheMetricsDecorator) Clear(ctx context.Context) error {
	start := time.Now()
	defer func() {
		d.RecordOperation("clear", time.Since(start))
	}()

	return d.provider.Clear(ctx)
}

// GetStats returns cache statistics
func (d *CacheMetricsDecorator) GetStats() ports.CacheStats {
	d.stats.mutex.RLock()
	defer d.stats.mutex.RUnlock()

	total := d.stats.hits + d.stats.misses
	hitRatio := float64(0)
	if total > 0 {
		hitRatio = float64(d.stats.hits) / float64(total)
	}

	return ports.CacheStats{
		Hits:        d.stats.hits,
		Misses:      d.stats.misses,
		TotalOps:    total,
		HitRatio:    hitRatio,
		LastUpdated: time.Now(),
	}
}

// RecordHit increments the cache hit counter
func (d *CacheMetricsDecorator) RecordHit() {
	d.stats.mutex.Lock()
	defer d.stats.mutex.Unlock()
	d.stats.hits++
}

// RecordMiss increments the cache miss counter
func (d *CacheMetricsDecorator) RecordMiss() {
	d.stats.mutex.Lock()
	defer d.stats.mutex.Unlock()
	d.stats.misses++
}

// RecordOperation records a cache operation with duration metrics
func (d *CacheMetricsDecorator) RecordOperation(operation string, duration time.Duration) {
	d.stats.mutex.Lock()
	defer d.stats.mutex.Unlock()

	metrics, exists := d.stats.operations[operation]
	if !exists {
		metrics = &cacheOperationMetrics{
			minLatency: duration,
			maxLatency: duration,
		}
		d.stats.operations[operation] = metrics
	}

	metrics.count++
	metrics.totalLatency += duration
	metrics.avgLatency = time.Duration(int64(metrics.totalLatency) / metrics.count)

	if duration > metrics.maxLatency {
		metrics.maxLatency = duration
	}
	if duration < metrics.minLatency {
		metrics.minLatency = duration
	}
}

// GetOperationMetrics returns operation-specific performance metrics
func (d *CacheMetricsDecorator) GetOperationMetrics() map[string]ports.OperationMetrics {
	d.stats.mutex.RLock()
	defer d.stats.mutex.RUnlock()

	result := make(map[string]ports.OperationMetrics)
	for operation, metrics := range d.stats.operations {
		result[operation] = ports.OperationMetrics{
			Count:        metrics.count,
			TotalLatency: metrics.totalLatency,
			AvgLatency:   metrics.avgLatency,
			MaxLatency:   metrics.maxLatency,
			MinLatency:   metrics.minLatency,
		}
	}
	return result
}
