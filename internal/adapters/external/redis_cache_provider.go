package external

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// RedisCacheProviderAdapter implements CacheProvider port using Redis
type RedisCacheProviderAdapter struct {
	client *redis.Client
	stats  struct {
		hits       int64
		misses     int64
		operations map[string]*redisOperationMetrics
		mutex      sync.RWMutex
	}
}

type redisOperationMetrics struct {
	count        int64
	totalLatency time.Duration
	avgLatency   time.Duration
	maxLatency   time.Duration
	minLatency   time.Duration
}

// NewRedisCacheProviderAdapter creates a new Redis cache provider adapter
func NewRedisCacheProviderAdapter(ctx context.Context, config *config.RedisConfig) (*RedisCacheProviderAdapter, error) {
	if config == nil {
		return nil, infrastructure.NewConfigurationError("redis config cannot be nil")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, infrastructure.NewExternalAPIError("failed to connect to Redis", err)
	}

	return &RedisCacheProviderAdapter{
		client: client,
		stats: struct {
			hits       int64
			misses     int64
			operations map[string]*redisOperationMetrics
			mutex      sync.RWMutex
		}{
			operations: make(map[string]*redisOperationMetrics),
		},
	}, nil
}

// Get retrieves a value from Redis cache
func (r *RedisCacheProviderAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		r.RecordOperation("get", time.Since(start))
	}()

	if key == "" {
		return nil, infrastructure.NewValidationError(ErrCacheKeyEmpty)
	}

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			r.recordMiss()
			return nil, ports.NewNotFoundError("cache miss")
		}
		return nil, infrastructure.NewExternalAPIError("redis get operation failed", err)
	}

	r.recordHit()
	return []byte(val), nil
}

// Set stores a value in Redis cache with TTL
func (r *RedisCacheProviderAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		r.RecordOperation("set", time.Since(start))
	}()

	if key == "" {
		return infrastructure.NewValidationError(ErrCacheKeyEmpty)
	}
	if value == nil {
		return infrastructure.NewValidationError(ErrCacheValueNil)
	}
	if ttl <= 0 {
		return infrastructure.NewValidationError(ErrCacheTTLNonPositive)
	}

	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis set operation failed", err)
	}

	return nil
}

// Delete removes a value from Redis cache
func (r *RedisCacheProviderAdapter) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		r.RecordOperation("delete", time.Since(start))
	}()

	if key == "" {
		return infrastructure.NewValidationError(ErrCacheKeyEmpty)
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis delete operation failed", err)
	}

	return nil
}

// Exists checks if a key exists in Redis cache
func (r *RedisCacheProviderAdapter) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	defer func() {
		r.RecordOperation("exists", time.Since(start))
	}()

	if key == "" {
		return false, infrastructure.NewValidationError(ErrCacheKeyEmpty)
	}

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, infrastructure.NewExternalAPIError("redis exists operation failed", err)
	}

	return count > 0, nil
}

// Clear removes all keys from the Redis database
func (r *RedisCacheProviderAdapter) Clear(ctx context.Context) error {
	start := time.Now()
	defer func() {
		r.RecordOperation("clear", time.Since(start))
	}()

	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis clear operation failed", err)
	}

	return nil
}

// GetStats returns cache statistics
func (r *RedisCacheProviderAdapter) GetStats() ports.CacheStats {
	r.stats.mutex.RLock()
	defer r.stats.mutex.RUnlock()

	total := r.stats.hits + r.stats.misses
	hitRatio := float64(0)
	if total > 0 {
		hitRatio = float64(r.stats.hits) / float64(total)
	}

	return ports.CacheStats{
		Hits:        r.stats.hits,
		Misses:      r.stats.misses,
		TotalOps:    total,
		HitRatio:    hitRatio,
		LastUpdated: time.Now(),
	}
}

// RecordHit increments the cache hit counter
func (r *RedisCacheProviderAdapter) RecordHit() {
	r.recordHit()
}

// RecordMiss increments the cache miss counter
func (r *RedisCacheProviderAdapter) RecordMiss() {
	r.recordMiss()
}

// RecordOperation records a cache operation with duration metrics
func (r *RedisCacheProviderAdapter) RecordOperation(operation string, duration time.Duration) {
	r.stats.mutex.Lock()
	defer r.stats.mutex.Unlock()

	metrics, exists := r.stats.operations[operation]
	if !exists {
		metrics = &redisOperationMetrics{
			minLatency: duration,
			maxLatency: duration,
		}
		r.stats.operations[operation] = metrics
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
func (r *RedisCacheProviderAdapter) GetOperationMetrics() map[string]ports.OperationMetrics {
	r.stats.mutex.RLock()
	defer r.stats.mutex.RUnlock()

	result := make(map[string]ports.OperationMetrics)
	for operation, metrics := range r.stats.operations {
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
func (r *RedisCacheProviderAdapter) recordHit() {
	r.stats.mutex.Lock()
	defer r.stats.mutex.Unlock()
	r.stats.hits++
}

// recordMiss increments the cache miss counter (internal method)
func (r *RedisCacheProviderAdapter) recordMiss() {
	r.stats.mutex.Lock()
	defer r.stats.mutex.Unlock()
	r.stats.misses++
}

// Close closes the Redis client connection
func (r *RedisCacheProviderAdapter) Close() error {
	if err := r.client.Close(); err != nil {
		return infrastructure.NewExternalAPIError("failed to close Redis connection", err)
	}
	return nil
}

// Ping checks if Redis connection is alive
func (r *RedisCacheProviderAdapter) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return infrastructure.NewExternalAPIError("Redis ping failed", err)
	}
	return nil
}
