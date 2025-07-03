package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"weatherapi.app/models"
)

// GenericCacheInterface defines generic cache operations
type GenericCacheInterface interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
	Delete(ctx context.Context, key string)
	Clear(ctx context.Context)
}

// CacheInterface defines the interface for weather caching operations
type CacheInterface interface {
	Get(key string) (*models.WeatherResponse, bool)
	Set(key string, value *models.WeatherResponse, ttl time.Duration)
	Delete(key string)
	Clear()
}

type cacheEntry struct {
	Data      []byte
	ExpiresAt time.Time
}

type MemoryCache struct {
	data   map[string]cacheEntry
	mutex  sync.RWMutex
	ticker *time.Ticker
	stopCh chan struct{}
}

func NewMemoryCache() GenericCacheInterface {
	cache := &MemoryCache{
		data:   make(map[string]cacheEntry),
		ticker: time.NewTicker(5 * time.Minute),
		stopCh: make(chan struct{}),
	}

	go cache.cleanup()
	return cache
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
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

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	if value == nil {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = cacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) Delete(ctx context.Context, key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

func (c *MemoryCache) Clear(ctx context.Context) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]cacheEntry)
}

// WeatherCache wraps generic cache with weather-specific operations
type WeatherCache struct {
	cache GenericCacheInterface
}

func NewWeatherCache(cache GenericCacheInterface) CacheInterface {
	return &WeatherCache{
		cache: cache,
	}
}

func (w *WeatherCache) Get(key string) (*models.WeatherResponse, bool) {
	data, found := w.cache.Get(context.Background(), key)
	if !found {
		return nil, false
	}

	var weather models.WeatherResponse
	if err := json.Unmarshal(data, &weather); err != nil {
		return nil, false
	}

	return &weather, true
}

func (w *WeatherCache) Set(key string, value *models.WeatherResponse, ttl time.Duration) {
	if value == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	w.cache.Set(context.Background(), key, data, ttl)
}

func (w *WeatherCache) Delete(key string) {
	w.cache.Delete(context.Background(), key)
}

func (w *WeatherCache) Clear() {
	w.cache.Clear(context.Background())
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
