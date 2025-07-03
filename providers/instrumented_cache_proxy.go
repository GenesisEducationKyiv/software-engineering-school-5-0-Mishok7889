package providers

import (
	"context"
	"log/slog"
	"time"

	"weatherapi.app/metrics"
	"weatherapi.app/providers/cache"
)

type InstrumentedCache struct {
	cache   cache.GenericCacheInterface
	metrics *metrics.CacheMetrics
}

func NewInstrumentedCache(cache cache.GenericCacheInterface, cacheType string) *InstrumentedCache {
	return &InstrumentedCache{
		cache:   cache,
		metrics: metrics.NewCacheMetrics(cacheType),
	}
}

func (c *InstrumentedCache) measureLatency(operation string, fn func()) {
	start := time.Now()
	fn()
	latency := time.Since(start).Seconds()
	c.metrics.RecordLatency(operation, latency)
}

func (c *InstrumentedCache) Get(ctx context.Context, key string) ([]byte, bool) {
	var data []byte
	var found bool

	c.measureLatency("get", func() {
		data, found = c.cache.Get(ctx, key)
	})

	if found {
		c.metrics.RecordHit()
		slog.Debug("cache hit", "key", key)
	} else {
		c.metrics.RecordMiss()
		slog.Debug("cache miss", "key", key)
	}

	return data, found
}

func (c *InstrumentedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	c.measureLatency("set", func() {
		c.cache.Set(ctx, key, value, ttl)
	})
	slog.Debug("cache set", "key", key)
}

func (c *InstrumentedCache) Delete(ctx context.Context, key string) {
	c.cache.Delete(ctx, key)
}

func (c *InstrumentedCache) Clear(ctx context.Context) {
	c.cache.Clear(ctx)
}

func (c *InstrumentedCache) GetMetrics() *metrics.CacheMetrics {
	return c.metrics
}
