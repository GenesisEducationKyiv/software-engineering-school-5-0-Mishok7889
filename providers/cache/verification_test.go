package cache

import (
	"context"
	"testing"
	"time"

	"weatherapi.app/models"
)

// This test verifies that our cache implementations satisfy the interfaces
func TestInterfaceCompliance(t *testing.T) {
	// Test that MemoryCache implements GenericCache
	var memCache = NewMemoryCache()
	_ = memCache

	// Test that WeatherCache implements Cache
	var weatherCache = NewWeatherCache(memCache)
	_ = weatherCache

	// Test basic functionality
	weather := &models.WeatherResponse{
		Temperature: 25.0,
		Humidity:    60.0,
		Description: "Test weather",
	}

	// Test weather cache operations
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
