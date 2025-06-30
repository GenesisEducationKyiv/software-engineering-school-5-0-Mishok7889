package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"weatherapi.app/models"
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

func NewRedisCache(config *RedisCacheConfig) (CacheInterface, error) {
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

func (r *RedisCache) Get(key string) (*models.WeatherResponse, bool) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false
		}
		slog.Error("Redis get error", "error", err, "key", key)
		return nil, false
	}

	var weather models.WeatherResponse
	if err := json.Unmarshal([]byte(val), &weather); err != nil {
		slog.Error("Redis unmarshal error", "error", err, "key", key)
		return nil, false
	}

	return &weather, true
}

func (r *RedisCache) Set(key string, value *models.WeatherResponse, ttl time.Duration) {
	if value == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		slog.Error("Redis marshal error", "error", err, "key", key)
		return
	}

	if err := r.client.Set(r.ctx, key, data, ttl).Err(); err != nil {
		slog.Error("Redis set error", "error", err, "key", key)
	}
}

func (r *RedisCache) Delete(key string) {
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		slog.Error("Redis delete error", "error", err, "key", key)
	}
}

func (r *RedisCache) Clear() {
	if err := r.client.FlushDB(r.ctx).Err(); err != nil {
		slog.Error("Redis clear error", "error", err)
	}
}
