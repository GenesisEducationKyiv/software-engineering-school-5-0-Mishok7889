package external

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// RedisCacheProviderAdapter implements CacheProvider port using Redis
type RedisCacheProviderAdapter struct {
	client *redis.Client
	stats  struct {
		hits   int64
		misses int64
		mutex  sync.RWMutex
	}
}

// NewRedisCacheProviderAdapter creates a new Redis cache provider adapter
func NewRedisCacheProviderAdapter(config *config.RedisConfig) (*RedisCacheProviderAdapter, error) {
	if config == nil {
		return nil, errors.NewConfigurationError("redis config cannot be nil", nil)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  time.Duration(config.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.NewExternalAPIError("failed to connect to Redis", err)
	}

	return &RedisCacheProviderAdapter{
		client: client,
	}, nil
}

// Get retrieves a value from Redis cache
func (r *RedisCacheProviderAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, errors.NewValidationError("cache key cannot be empty")
	}

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			r.recordMiss()
			return nil, errors.NewNotFoundError("cache miss")
		}
		return nil, errors.NewExternalAPIError("redis get operation failed", err)
	}

	r.recordHit()
	return []byte(val), nil
}

// Set stores a value in Redis cache with TTL
func (r *RedisCacheProviderAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return errors.NewValidationError("cache key cannot be empty")
	}
	if value == nil {
		return errors.NewValidationError("cache value cannot be nil")
	}
	if ttl <= 0 {
		return errors.NewValidationError("cache TTL must be positive")
	}

	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return errors.NewExternalAPIError("redis set operation failed", err)
	}

	return nil
}

// Delete removes a value from Redis cache
func (r *RedisCacheProviderAdapter) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.NewValidationError("cache key cannot be empty")
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return errors.NewExternalAPIError("redis delete operation failed", err)
	}

	return nil
}

// Exists checks if a key exists in Redis cache
func (r *RedisCacheProviderAdapter) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, errors.NewValidationError("cache key cannot be empty")
	}

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.NewExternalAPIError("redis exists operation failed", err)
	}

	return count > 0, nil
}

// Clear removes all keys from the Redis database
func (r *RedisCacheProviderAdapter) Clear(ctx context.Context) error {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return errors.NewExternalAPIError("redis clear operation failed", err)
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

// RecordOperation records a cache operation with duration
func (r *RedisCacheProviderAdapter) RecordOperation(operation string, duration time.Duration) {
	// Placeholder for future metrics implementation
	// Could be extended to track operation-specific metrics
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
		return errors.NewExternalAPIError("failed to close Redis connection", err)
	}
	return nil
}

// Ping checks if Redis connection is alive
func (r *RedisCacheProviderAdapter) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return errors.NewExternalAPIError("Redis ping failed", err)
	}
	return nil
}
