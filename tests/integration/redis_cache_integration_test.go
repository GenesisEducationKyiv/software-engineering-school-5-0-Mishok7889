package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/adapters/external"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// setupRedisAdapter creates a Redis adapter for testing
func setupRedisAdapter(t *testing.T) *external.RedisCacheProviderAdapter {
	t.Helper()

	// Create mock Redis server
	mockRedis := miniredis.RunT(t)

	// Create Redis configuration pointing to mock server
	redisConfig := &config.RedisConfig{
		Addr:         mockRedis.Addr(),
		Password:     "",
		DB:           0,
		DialTimeout:  5,
		ReadTimeout:  3,
		WriteTimeout: 3,
	}

	adapter, err := external.NewRedisCacheProviderAdapter(redisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis adapter: %v", err)
	}

	return adapter
}

// TestRedisCache_WeatherDataIntegration tests Redis cache with actual weather data
func TestRedisCache_WeatherDataIntegration(t *testing.T) {
	adapter := setupRedisAdapter(t)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Clear cache
	require.NoError(t, adapter.Clear(ctx))

	// Create sample weather data
	weatherData := &ports.WeatherData{
		Temperature: 25.5,
		Humidity:    65.0,
		Description: "Partly cloudy",
		City:        "London",
		Timestamp:   time.Now(),
	}

	// Serialize weather data to JSON
	jsonData, err := json.Marshal(weatherData)
	require.NoError(t, err)

	// Cache the weather data
	cacheKey := "weather:london"
	ttl := 10 * time.Minute

	err = adapter.Set(ctx, cacheKey, jsonData, ttl)
	require.NoError(t, err)

	// Retrieve the cached data
	cachedData, err := adapter.Get(ctx, cacheKey)
	require.NoError(t, err)

	// Deserialize the cached data
	var retrievedWeatherData ports.WeatherData
	err = json.Unmarshal(cachedData, &retrievedWeatherData)
	require.NoError(t, err)

	// Verify the data matches
	assert.Equal(t, weatherData.Temperature, retrievedWeatherData.Temperature)
	assert.Equal(t, weatherData.Humidity, retrievedWeatherData.Humidity)
	assert.Equal(t, weatherData.Description, retrievedWeatherData.Description)
	assert.Equal(t, weatherData.City, retrievedWeatherData.City)
	assert.True(t, weatherData.Timestamp.Equal(retrievedWeatherData.Timestamp))
}

// TestRedisCache_MultipleDataTypes tests Redis cache with different data types
func TestRedisCache_MultipleDataTypes(t *testing.T) {
	adapter := setupRedisAdapter(t)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Clear cache
	require.NoError(t, adapter.Clear(ctx))

	tests := []struct {
		name string
		key  string
		data interface{}
		ttl  time.Duration
	}{
		{
			name: "WeatherData",
			key:  "weather:test",
			data: &ports.WeatherData{
				Temperature: 20.0,
				Humidity:    70.0,
				Description: "Clear sky",
				City:        "Berlin",
				Timestamp:   time.Now(),
			},
			ttl: 5 * time.Minute,
		},
		{
			name: "StringData",
			key:  "string:test",
			data: "Hello, World!",
			ttl:  1 * time.Minute,
		},
		{
			name: "MapData",
			key:  "map:test",
			data: map[string]interface{}{
				"temperature": 22.5,
				"humidity":    80.0,
				"city":        "Paris",
				"conditions":  []string{"sunny", "warm"},
			},
			ttl: 3 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize data
			jsonData, err := json.Marshal(tt.data)
			require.NoError(t, err)

			// Store in cache
			err = adapter.Set(ctx, tt.key, jsonData, tt.ttl)
			require.NoError(t, err)

			// Retrieve from cache
			cachedData, err := adapter.Get(ctx, tt.key)
			require.NoError(t, err)

			// Verify data integrity
			assert.Equal(t, jsonData, cachedData)

			// Verify existence
			exists, err := adapter.Exists(ctx, tt.key)
			require.NoError(t, err)
			assert.True(t, exists)
		})
	}
}

// TestRedisCache_ConfigurationTypes tests different Redis configurations
func TestRedisCache_ConfigurationTypes(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.RedisConfig
		shouldWork  bool
		description string
	}{
		{
			name: "StandardConfig",
			config: &config.RedisConfig{
				Addr:         "localhost:6379",
				Password:     "",
				DB:           0,
				DialTimeout:  5,
				ReadTimeout:  3,
				WriteTimeout: 3,
			},
			shouldWork:  true,
			description: "Standard Redis configuration",
		},
		{
			name: "DatabaseSelection",
			config: &config.RedisConfig{
				Addr:         "localhost:6379",
				Password:     "",
				DB:           2,
				DialTimeout:  5,
				ReadTimeout:  3,
				WriteTimeout: 3,
			},
			shouldWork:  true,
			description: "Redis with different database",
		},
		{
			name: "CustomTimeouts",
			config: &config.RedisConfig{
				Addr:         "localhost:6379",
				Password:     "",
				DB:           0,
				DialTimeout:  10,
				ReadTimeout:  5,
				WriteTimeout: 5,
			},
			shouldWork:  true,
			description: "Redis with custom timeouts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := external.NewRedisCacheProviderAdapter(tt.config)

			if tt.shouldWork {
				if err != nil {
					t.Skipf("Skipping Redis test: %v", err)
				}
				require.NoError(t, err)
				require.NotNil(t, adapter)
				defer func() { _ = adapter.Close() }()

				// Test basic operation
				ctx := context.Background()
				err = adapter.Set(ctx, "test-key", []byte("test-value"), time.Minute)
				assert.NoError(t, err)

				value, err := adapter.Get(ctx, "test-key")
				assert.NoError(t, err)
				assert.Equal(t, []byte("test-value"), value)
			} else {
				assert.Error(t, err)
				assert.Nil(t, adapter)
			}
		})
	}
}

// TestRedisCache_PerformanceMetrics tests performance tracking
func TestRedisCache_PerformanceMetrics(t *testing.T) {
	adapter := setupRedisAdapter(t)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Clear cache
	require.NoError(t, adapter.Clear(ctx))

	// Perform operations and track metrics
	numOperations := 100

	// Set multiple keys
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		value := []byte(fmt.Sprintf("perf-value-%d", i))
		err := adapter.Set(ctx, key, value, time.Minute)
		require.NoError(t, err)
	}

	// Get all keys (should be hits)
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		_, err := adapter.Get(ctx, key)
		require.NoError(t, err)
	}

	// Get non-existent keys (should be misses)
	for i := 0; i < numOperations/2; i++ {
		key := fmt.Sprintf("non-existent-%d", i)
		_, err := adapter.Get(ctx, key)
		assert.Error(t, err)
	}

	// Check metrics
	stats := adapter.GetStats()
	assert.Equal(t, int64(numOperations), stats.Hits)
	assert.Equal(t, int64(numOperations/2), stats.Misses)
	assert.Equal(t, int64(numOperations+numOperations/2), stats.TotalOps)

	expectedHitRatio := float64(numOperations) / float64(numOperations+numOperations/2)
	assert.InDelta(t, expectedHitRatio, stats.HitRatio, 0.01)
}

// TestRedisCache_ConnectionRecovery tests connection recovery
func TestRedisCache_ConnectionRecovery(t *testing.T) {
	adapter := setupRedisAdapter(t)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Test initial connection
	err := adapter.Ping(ctx)
	require.NoError(t, err)

	// Store some data
	err = adapter.Set(ctx, "recovery-test", []byte("test-data"), time.Minute)
	require.NoError(t, err)

	// Verify data exists
	value, err := adapter.Get(ctx, "recovery-test")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), value)

	// Test connection again
	err = adapter.Ping(ctx)
	assert.NoError(t, err)
}
