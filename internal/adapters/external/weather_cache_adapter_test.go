package external

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// Interface compliance verification
var _ ports.WeatherCache = (*WeatherCacheAdapter)(nil)

// TestWeatherCacheAdapter_Integration tests the weather cache adapter integration
func TestWeatherCacheAdapter_Integration(t *testing.T) {
	tests := []struct {
		name       string
		cacheType  config.CacheType
		skipReason string
	}{
		{
			name:      "MemoryCache",
			cacheType: config.CacheTypeMemory,
		},
		{
			name:       "RedisCache",
			cacheType:  config.CacheTypeRedis,
			skipReason: "Redis connection required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create cache configuration
			cacheConfig := &config.CacheConfig{
				Type: tt.cacheType,
			}

			if tt.cacheType == config.CacheTypeRedis {
				cacheConfig.Redis = config.RedisConfig{
					Addr:         "localhost:6379",
					Password:     "",
					DB:           1,
					DialTimeout:  5,
					ReadTimeout:  3,
					WriteTimeout: 3,
				}
			}

			// Create cache provider using factory
			factory := NewCacheProviderFactory()
			genericCache, err := factory.CreateCacheProvider(cacheConfig)
			if err != nil && tt.skipReason != "" {
				t.Skipf("Skipping test: %s - %v", tt.skipReason, err)
			}
			require.NoError(t, err)

			// Create weather cache adapter
			weatherCache := NewWeatherCacheAdapter(genericCache)

			// Test weather cache operations
			ctx := context.Background()

			// Clear cache if supported
			if clearableCache, ok := genericCache.(interface{ Clear(context.Context) error }); ok {
				require.NoError(t, clearableCache.Clear(ctx))
			}

			// Test data - use UTC to avoid timezone issues with JSON serialization
			weatherData := &ports.WeatherData{
				Temperature: 25.5,
				Humidity:    65.0,
				Description: "Partly cloudy",
				City:        "London",
				Timestamp:   time.Now().UTC().Truncate(time.Second), // Use UTC for consistent JSON comparison
			}

			// Test Set and Get
			cacheKey := "weather:london:test"
			ttl := time.Minute

			err = weatherCache.Set(ctx, cacheKey, weatherData, ttl)
			require.NoError(t, err)

			retrievedData, err := weatherCache.Get(ctx, cacheKey)
			require.NoError(t, err)
			assert.Equal(t, weatherData, retrievedData)

			// Test metrics through generic cache provider
			if metricsProvider, ok := genericCache.(ports.CacheMetrics); ok {
				stats := metricsProvider.GetStats()
				assert.Equal(t, int64(1), stats.Hits)
				assert.Equal(t, int64(0), stats.Misses)
				assert.Equal(t, int64(1), stats.TotalOps)
				assert.Equal(t, float64(1), stats.HitRatio)
			}

			// Test cache miss
			_, err = weatherCache.Get(ctx, "non-existent-key")
			assert.Error(t, err)

			// Verify miss was recorded through generic cache
			if metricsProvider, ok := genericCache.(ports.CacheMetrics); ok {
				stats := metricsProvider.GetStats()
				assert.Equal(t, int64(1), stats.Hits)
				assert.Equal(t, int64(1), stats.Misses)
				assert.Equal(t, int64(2), stats.TotalOps)
				assert.Equal(t, float64(0.5), stats.HitRatio)
			}

			// Clean up Redis connection if applicable
			if redisCache, ok := genericCache.(*RedisCacheProviderAdapter); ok {
				_ = redisCache.Close()
			}
		})
	}
}

// TestWeatherCacheAdapter_ErrorHandling tests error handling
func TestWeatherCacheAdapter_ErrorHandling(t *testing.T) {
	// Create memory cache for testing
	factory := NewCacheProviderFactory()
	genericCache, err := factory.CreateCacheProvider(&config.CacheConfig{
		Type: config.CacheTypeMemory,
	})
	require.NoError(t, err)

	weatherCache := NewWeatherCacheAdapter(genericCache)
	ctx := context.Background()

	// Test setting nil weather data
	err = weatherCache.Set(ctx, "test-key", nil, time.Minute)
	assert.Error(t, err)

	// Test empty key
	err = weatherCache.Set(ctx, "", &ports.WeatherData{}, time.Minute)
	assert.Error(t, err)

	// Test cache miss
	_, err = weatherCache.Get(ctx, "non-existent-key")
	assert.Error(t, err)
}

// TestWeatherCacheAdapter_Serialization tests JSON serialization
func TestWeatherCacheAdapter_Serialization(t *testing.T) {
	// Create memory cache for testing
	factory := NewCacheProviderFactory()
	genericCache, err := factory.CreateCacheProvider(&config.CacheConfig{
		Type: config.CacheTypeMemory,
	})
	require.NoError(t, err)

	weatherCache := NewWeatherCacheAdapter(genericCache)
	ctx := context.Background()

	// Test complex weather data
	weatherData := &ports.WeatherData{
		Temperature: 22.5,
		Humidity:    78.3,
		Description: "Heavy rain with thunderstorms",
		City:        "New York",
		Timestamp:   time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
	}

	// Store and retrieve
	err = weatherCache.Set(ctx, "complex-weather", weatherData, time.Hour)
	require.NoError(t, err)

	retrieved, err := weatherCache.Get(ctx, "complex-weather")
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, weatherData.Temperature, retrieved.Temperature)
	assert.Equal(t, weatherData.Humidity, retrieved.Humidity)
	assert.Equal(t, weatherData.Description, retrieved.Description)
	assert.Equal(t, weatherData.City, retrieved.City)
	assert.True(t, weatherData.Timestamp.Equal(retrieved.Timestamp))

	// Verify that data is properly JSON-serialized in the underlying cache
	rawData, err := genericCache.Get(ctx, "complex-weather")
	require.NoError(t, err)

	var jsonData map[string]interface{}
	err = json.Unmarshal(rawData, &jsonData)
	require.NoError(t, err)

	assert.Equal(t, 22.5, jsonData["Temperature"])
	assert.Equal(t, 78.3, jsonData["Humidity"])
	assert.Equal(t, "Heavy rain with thunderstorms", jsonData["Description"])
	assert.Equal(t, "New York", jsonData["City"])
}
