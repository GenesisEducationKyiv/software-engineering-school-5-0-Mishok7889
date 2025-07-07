package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

type MemoryCacheProvider struct {
	data  map[string]memoryCacheItem
	mutex sync.RWMutex
	stats struct {
		hits   int64
		misses int64
		mutex  sync.RWMutex
	}
}

type memoryCacheItem struct {
	data      []byte
	expiresAt time.Time
}

func NewMemoryCacheProvider() *MemoryCacheProvider {
	return &MemoryCacheProvider{
		data: make(map[string]memoryCacheItem),
	}
}

func (c *MemoryCacheProvider) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, errors.NewValidationError("cache key cannot be empty")
	}

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		c.recordMiss()
		return nil, errors.NewNotFoundError("cache miss")
	}

	c.recordHit()
	return item.data, nil
}

func (c *MemoryCacheProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return errors.NewValidationError("cache key cannot be empty")
	}
	if value == nil {
		return errors.NewValidationError("cache value cannot be nil")
	}
	if ttl <= 0 {
		return errors.NewValidationError("cache TTL must be positive")
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
	if key == "" {
		return errors.NewValidationError("cache key cannot be empty")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	return nil
}

func (c *MemoryCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, errors.NewValidationError("cache key cannot be empty")
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
	// Placeholder for future metrics implementation
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
