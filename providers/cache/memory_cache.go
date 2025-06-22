package cache

import (
	"sync"
	"time"

	"weatherapi.app/models"
)

// WeatherCache represents a cached weather entry
type WeatherCache struct {
	Data      *models.WeatherResponse
	ExpiresAt time.Time
}

// CacheInterface defines the interface for caching operations
type CacheInterface interface {
	Get(key string) (*models.WeatherResponse, bool)
	Set(key string, value *models.WeatherResponse, ttl time.Duration)
	Delete(key string)
	Clear()
}

type MemoryCache struct {
	data   map[string]WeatherCache
	mutex  sync.RWMutex
	ticker *time.Ticker
	stopCh chan struct{}
}

func NewMemoryCache() CacheInterface {
	cache := &MemoryCache{
		data:   make(map[string]WeatherCache),
		ticker: time.NewTicker(5 * time.Minute),
		stopCh: make(chan struct{}),
	}

	go cache.cleanup()
	return cache
}

func (c *MemoryCache) Get(key string) (*models.WeatherResponse, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Data, true
}

func (c *MemoryCache) Set(key string, value *models.WeatherResponse, ttl time.Duration) {
	if value == nil {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = WeatherCache{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a specific key from cache
func (c *MemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

// Clear removes all entries from cache
func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]WeatherCache)
}

func (c *MemoryCache) cleanup() {
	for {
		select {
		case <-c.ticker.C:
			c.removeExpiredEntries()
		case <-c.stopCh:
			c.ticker.Stop()
			return
		}
	}
}

// removeExpiredEntries removes all expired cache entries
func (c *MemoryCache) removeExpiredEntries() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
}
