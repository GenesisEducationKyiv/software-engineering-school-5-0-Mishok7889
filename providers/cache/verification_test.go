package cache

import (
	"context"
	"testing"
	"time"

	"weatherapi.app/models"
)

// TestInterfaceCompliance verifies that our cache implementations satisfy the interfaces at compile time
func TestInterfaceCompliance(t *testing.T) {
	// Verify MemoryCache implements GenericCache
	var _ GenericCache = (*MemoryCache)(nil)

	// Verify WeatherCache implements Cache
	var _ Cache = (*WeatherCache)(nil)
}

// TestWeatherCacheOperations tests the actual functionality of the cache implementations
func TestWeatherCacheOperations(t *testing.T) {
	// Create cache instances
	memCache := NewMemoryCache()
	weatherCache := NewWeatherCache(memCache)

	// Test weather cache operations
	weather := &models.WeatherResponse{
		Temperature: 25.0,
		Humidity:    60.0,
		Description: "Test weather",
	}

	weatherCache.Set("test:key", weather, time.Minute)
	result, found := weatherCache.Get("test:key")

	if !found {
		t.Error("Expected to find cached weather data")
	}

	if result.Temperature != weather.Temperature {
		t.Errorf("Expected temperature %v, got %v", weather.Temperature, result.Temperature)
	}

	// Test generic cache operations
	data := []byte(`{"temperature":20.0,"humidity":50.0,"description":"Generic test"}`)
	memCache.Set(context.Background(), "test:generic", data, time.Minute)
	genericResult, genericFound := memCache.Get(context.Background(), "test:generic")

	if !genericFound {
		t.Error("Expected to find generic cached data")
	}

	if string(genericResult) != string(data) {
		t.Errorf("Expected data %s, got %s", string(data), string(genericResult))
	}
}
