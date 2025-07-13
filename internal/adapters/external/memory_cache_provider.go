package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

type MemoryCacheProvider struct {
	data  map[string]memoryCacheItem
	mutex sync.RWMutex
	stats struct {
		hits       int64
		misses     int64
		operations map[string]*operationMetrics
		mutex      sync.RWMutex
	}
}

type operationMetrics struct {
	count        int64
	totalLatency time.Duration
	avgLatency   time.Duration
	maxLatency   time.Duration
	minLatency   time.Duration
}

type memoryCacheItem struct {
	data      []byte
	expiresAt time.Time
}

func NewMemoryCacheProvider() *MemoryCacheProvider {
	return &MemoryCacheProvider{
		data: make(map[string]memoryCacheItem),
		stats: struct {
			hits       int64
			misses     int64
			operations map[string]*operationMetrics
			mutex      sync.RWMutex
		}{
			operations: make(map[string]*operationMetrics),
		},
	}
}

func (c *MemoryCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		c.RecordOperation("get", time.Since(start))
	}()

	if key == "" {
		return nil, infrastructure.NewValidationError("cache key cannot be empty")
	}

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		c.recordMiss()
		return nil, ports.NewNotFoundError("cache miss")
	}

	c.recordHit()
	return item.data, nil
}

func (c *MemoryCacheProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		c.RecordOperation("set", time.Since(start))
	}()

	if key == "" {
		return infrastructure.NewValidationError("cache key cannot be empty")
	}
	if value == nil {
		return infrastructure.NewValidationError("cache value cannot be nil")
	}
	if ttl <= 0 {
		return infrastructure.NewValidationError("cache TTL must be positive")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = memoryCacheItem{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *MemoryCacheProvider) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		c.RecordOperation("delete", time.Since(start))
	}()

	if key == "" {
		return infrastructure.NewValidationError("cache key cannot be empty")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	return nil
}

func (c *MemoryCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		c.RecordOperation("exists", time.Since(start))
	}()

	if key == "" {
		return false, infrastructure.NewValidationError("cache key cannot be empty")
	}

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return false, nil
	}

	return !time.Now().After(item.expiresAt), nil
}

func (c *MemoryCacheProvider) Clear(ctx context.Context) error {
	start := time.Now()
	defer func() {
		c.RecordOperation("clear", time.Since(start))
	}()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]memoryCacheItem)
	return nil
}

func (c *MemoryCacheProvider) GetStats() ports.CacheStats {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	total := c.stats.hits + c.stats.misses
	hitRatio := float64(0)
	if total > 0 {
		hitRatio = float64(c.stats.hits) / float64(total)
	}

	return ports.CacheStats{
		Hits:        c.stats.hits,
		Misses:      c.stats.misses,
		TotalOps:    total,
		HitRatio:    hitRatio,
		LastUpdated: time.Now(),
	}
}

func (c *MemoryCacheProvider) RecordHit() {
	c.recordHit()
}

func (c *MemoryCacheProvider) RecordMiss() {
	c.recordMiss()
}

func (c *MemoryCacheProvider) RecordOperation(operation string, duration time.Duration) {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	metrics, exists := c.stats.operations[operation]
	if !exists {
		metrics = &operationMetrics{
			minLatency: duration,
			maxLatency: duration,
		}
		c.stats.operations[operation] = metrics
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
func (c *MemoryCacheProvider) GetOperationMetrics() map[string]ports.OperationMetrics {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	result := make(map[string]ports.OperationMetrics)
	for operation, metrics := range c.stats.operations {
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

// recordHit increments the cache hit counter (internal method)
func (c *MemoryCacheProvider) recordHit() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	c.stats.hits++
}

// recordMiss increments the cache miss counter (internal method)
func (c *MemoryCacheProvider) recordMiss() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	c.stats.misses++
}
