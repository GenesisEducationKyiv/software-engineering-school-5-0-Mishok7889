package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
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
	hits   int64
	misses int64
}

// NewMemoryCacheProviderAdapter creates a new in-memory cache adapter
func NewMemoryCacheProviderAdapter() *MemoryCacheProviderAdapter {
	return &MemoryCacheProviderAdapter{
		data: make(map[string]cachedItem),
	}
}

// Get retrieves weather data from cache
func (c *MemoryCacheProviderAdapter) Get(ctx context.Context, key string) (*ports.WeatherData, error) {
	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		c.RecordMiss()
		return nil, errors.NewNotFoundError("cache miss")
	}

	c.RecordHit()
	return item.data, nil
}

// Set stores weather data in cache
func (c *MemoryCacheProviderAdapter) Set(ctx context.Context, key string, weather *ports.WeatherData, ttl time.Duration) error {
	if weather == nil {
		return errors.NewValidationError("weather data cannot be nil")
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

// RecordOperation records a cache operation (placeholder for future metrics)
func (c *MemoryCacheProviderAdapter) RecordOperation(operation string, duration time.Duration) {
	// Placeholder - can be extended with operation-specific metrics
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
func (m *WeatherMetricsAdapter) GetProviderInfo() map[string]interface{} {
	// Get provider information from the provider manager
	providerInfo := m.providerManager.GetProviderInfo()

	// Merge with cache and status information
	result := map[string]interface{}{
		"providers_available": 1,
		"primary_provider":    "weatherapi",
		"status":              "active",
		"cache_enabled":       true, // Add cache_enabled field
	}

	// Add provider_order from the provider manager if available
	if providerOrder, ok := providerInfo["provider_order"]; ok {
		result["provider_order"] = providerOrder
	}

	return result
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
