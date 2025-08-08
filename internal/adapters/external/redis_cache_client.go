package external

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// RedisCacheClient implements CacheProvider port using Redis without metrics
type RedisCacheClient struct {
	client *redis.Client
}

// NewRedisCacheClient creates a new Redis cache client
func NewRedisCacheClient(ctx context.Context, config *config.RedisConfig) (*RedisCacheClient, error) {
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

	return &RedisCacheClient{
		client: client,
	}, nil
}

func (r *RedisCacheClient) ValidateKey(key string) error {
	if key == "" {
		return infrastructure.NewValidationError("cache key cannot be empty")
	}
	return nil
}

func (r *RedisCacheClient) ValidateValue(value []byte) error {
	if value == nil {
		return infrastructure.NewValidationError("cache value cannot be nil")
	}
	return nil
}

func (r *RedisCacheClient) ValidateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return infrastructure.NewValidationError("cache TTL must be positive")
	}
	return nil
}

// Get retrieves a value from Redis cache
func (r *RedisCacheClient) Get(ctx context.Context, key string) ([]byte, error) {
	if err := r.ValidateKey(key); err != nil {
		return nil, err
	}

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ports.NewNotFoundError("cache miss")
		}
		return nil, infrastructure.NewExternalAPIError("redis get operation failed", err)
	}

	return []byte(val), nil
}

// Set stores a value in Redis cache with TTL
func (r *RedisCacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.ValidateKey(key); err != nil {
		return err
	}
	if err := r.ValidateValue(value); err != nil {
		return err
	}
	if err := r.ValidateTTL(ttl); err != nil {
		return err
	}

	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis set operation failed", err)
	}

	return nil
}

// Delete removes a value from Redis cache
func (r *RedisCacheClient) Delete(ctx context.Context, key string) error {
	if err := r.ValidateKey(key); err != nil {
		return err
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis delete operation failed", err)
	}

	return nil
}

// Exists checks if a key exists in Redis cache
func (r *RedisCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	if err := r.ValidateKey(key); err != nil {
		return false, err
	}

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, infrastructure.NewExternalAPIError("redis exists operation failed", err)
	}

	return count > 0, nil
}

// Clear removes all keys from the Redis database
func (r *RedisCacheClient) Clear(ctx context.Context) error {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return infrastructure.NewExternalAPIError("redis clear operation failed", err)
	}

	return nil
}

// Close closes the Redis client connection
func (r *RedisCacheClient) Close() error {
	if err := r.client.Close(); err != nil {
		return infrastructure.NewExternalAPIError("failed to close Redis connection", err)
	}
	return nil
}

// Ping checks if Redis connection is alive
func (r *RedisCacheClient) Ping(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return infrastructure.NewExternalAPIError("Redis ping failed", err)
	}
	return nil
}
