package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/models"
)

func TestRedisCacheBasicOperations(t *testing.T) {
	config := &RedisCacheConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           1, // Use test database
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	genericCache, err := NewRedisCache(config)
	if err != nil {
		t.Skipf("Redis not available, skipping test: %v", err)
		return
	}

	cache := NewWeatherCache(genericCache)
	defer cache.Clear()

	testWeather := &models.WeatherResponse{
		Temperature: 25.5,
		Humidity:    60.0,
		Description: "Sunny",
	}

	t.Run("Set and Get", func(t *testing.T) {
		key := "test:london"
		cache.Set(key, testWeather, 5*time.Minute)

		result, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, testWeather.Temperature, result.Temperature)
		assert.Equal(t, testWeather.Humidity, result.Humidity)
		assert.Equal(t, testWeather.Description, result.Description)
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		result, found := cache.Get("test:nonexistent")
		assert.False(t, found)
		assert.Nil(t, result)
	})

	t.Run("Delete", func(t *testing.T) {
		key := "test:delete"
		cache.Set(key, testWeather, 5*time.Minute)

		_, found := cache.Get(key)
		assert.True(t, found)

		cache.Delete(key)

		_, found = cache.Get(key)
		assert.False(t, found)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		key := "test:ttl"
		cache.Set(key, testWeather, 100*time.Millisecond)

		result, found := cache.Get(key)
		assert.True(t, found)
		assert.NotNil(t, result)

		require.Eventually(t, func() bool {
			_, found = cache.Get(key)
			return !found
		}, 500*time.Millisecond, 10*time.Millisecond)
	})
}

func TestMemoryCache(t *testing.T) {
	genericCache := NewMemoryCache()
	cache := NewWeatherCache(genericCache)

	testWeather := &models.WeatherResponse{
		Temperature: 20.0,
		Humidity:    70.0,
		Description: "Cloudy",
	}

	t.Run("Basic operations", func(t *testing.T) {
		key := "test:memory:london"
		cache.Set(key, testWeather, 5*time.Minute)

		result, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, testWeather.Temperature, result.Temperature)
		assert.Equal(t, testWeather.Humidity, result.Humidity)
		assert.Equal(t, testWeather.Description, result.Description)

		cache.Delete(key)
		_, found = cache.Get(key)
		assert.False(t, found)
	})
}
