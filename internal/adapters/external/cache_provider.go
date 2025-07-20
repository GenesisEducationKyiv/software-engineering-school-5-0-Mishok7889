package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// MemoryCacheProviderAdapter implements WeatherCache port using in-memory storage
type MemoryCacheProviderAdapter struct {
	data  map[string]cachedItem
	mutex sync.RWMutex
	stats cacheStats
}

type cachedItem struct {
	data      *ports.WeatherData
	expiresAt time.Time
}

type cacheStats struct {
	hits       int64
	misses     int64
	operations map[string]*operationStats
}

type operationStats struct {
	count        int64
	totalLatency time.Duration
	avgLatency   time.Duration
	maxLatency   time.Duration
	minLatency   time.Duration
}

// NewMemoryCacheProviderAdapter creates a new in-memory cache adapter
func NewMemoryCacheProviderAdapter() *MemoryCacheProviderAdapter {
	return &MemoryCacheProviderAdapter{
		data: make(map[string]cachedItem),
		stats: cacheStats{
			operations: make(map[string]*operationStats),
		},
	}
}

// Get retrieves weather data from cache
func (c *MemoryCacheProviderAdapter) Get(ctx context.Context, key string) (*ports.WeatherData, error) {
	start := time.Now()
	defer func() {
		c.RecordOperation("weather_get", time.Since(start))
	}()

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		c.RecordMiss()
		return nil, ports.NewNotFoundError("cache miss")
	}

	c.RecordHit()
	return item.data, nil
}

func (c *MemoryCacheProviderAdapter) Set(ctx context.Context, key string, weather *ports.WeatherData, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		c.RecordOperation("weather_set", time.Since(start))
	}()

	if weather == nil {
		return infrastructure.NewValidationError("weather data cannot be nil")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = cachedItem{
		data:      weather,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// GetStats returns cache statistics
func (c *MemoryCacheProviderAdapter) GetStats() ports.CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

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

// RecordHit increments the cache hit counter
func (c *MemoryCacheProviderAdapter) RecordHit() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stats.hits++
}

// RecordMiss increments the cache miss counter
func (c *MemoryCacheProviderAdapter) RecordMiss() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.stats.misses++
}

// RecordOperation records a cache operation with duration metrics
func (c *MemoryCacheProviderAdapter) RecordOperation(operation string, duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	stats, exists := c.stats.operations[operation]
	if !exists {
		stats = &operationStats{
			minLatency: duration,
			maxLatency: duration,
		}
		c.stats.operations[operation] = stats
	}

	stats.count++
	stats.totalLatency += duration
	stats.avgLatency = time.Duration(int64(stats.totalLatency) / stats.count)

	if duration > stats.maxLatency {
		stats.maxLatency = duration
	}
	if duration < stats.minLatency {
		stats.minLatency = duration
	}
}

// GetOperationMetrics returns operation-specific performance metrics
func (c *MemoryCacheProviderAdapter) GetOperationMetrics() map[string]ports.OperationMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make(map[string]ports.OperationMetrics)
	for operation, stats := range c.stats.operations {
		result[operation] = ports.OperationMetrics{
			Count:        stats.count,
			TotalLatency: stats.totalLatency,
			AvgLatency:   stats.avgLatency,
			MaxLatency:   stats.maxLatency,
			MinLatency:   stats.minLatency,
		}
	}
	return result
}

// WeatherMetricsAdapter implements WeatherMetrics port
type WeatherMetricsAdapter struct {
	cache           ports.WeatherCache
	providerManager ports.WeatherProviderManager
}

// NewWeatherMetricsAdapter creates a new weather metrics adapter
func NewWeatherMetricsAdapter(cache ports.WeatherCache, manager ports.WeatherProviderManager) ports.WeatherMetrics {
	return &WeatherMetricsAdapter{
		cache:           cache,
		providerManager: manager,
	}
}

// GetProviderInfo returns provider information
func (m *WeatherMetricsAdapter) GetProviderInfo() ports.ProviderInfo {
	return m.providerManager.GetProviderInfo()
}

// GetCacheMetrics returns cache performance metrics
func (m *WeatherMetricsAdapter) GetCacheMetrics() (ports.CacheStats, error) {
	if cacheWithStats, ok := m.cache.(interface{ GetStats() ports.CacheStats }); ok {
		return cacheWithStats.GetStats(), nil
	}

	return ports.CacheStats{
		LastUpdated: time.Now(),
	}, nil
}
