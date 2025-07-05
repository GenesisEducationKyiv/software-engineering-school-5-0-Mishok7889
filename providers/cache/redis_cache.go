package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

type RedisCacheConfig struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewRedisCache(config *RedisCacheConfig) (GenericCache, error) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	slog.Info("Redis cache connected successfully", "addr", config.Addr)

	return &RedisCache{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, bool) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		slog.Error("Redis get error", "error", err, "key", key)
		return nil, false
	}

	return []byte(val), true
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	if value == nil {
		return
	}

	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		slog.Error("Redis set error", "error", err, "key", key)
	}
}

func (r *RedisCache) Delete(ctx context.Context, key string) {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		slog.Error("Redis delete error", "error", err, "key", key)
	}
}

func (r *RedisCache) Clear(ctx context.Context) {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		slog.Error("Redis clear error", "error", err)
	}
}
