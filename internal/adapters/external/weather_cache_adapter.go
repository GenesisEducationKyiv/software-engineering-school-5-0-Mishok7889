package external

import (
	"context"
	"encoding/json"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// WeatherCacheAdapter bridges generic CacheProvider to weather-specific WeatherCache
type WeatherCacheAdapter struct {
	cacheProvider ports.CacheProvider
}

// NewWeatherCacheAdapter creates a weather cache adapter using generic cache provider
func NewWeatherCacheAdapter(cacheProvider ports.CacheProvider) ports.WeatherCache {
	return &WeatherCacheAdapter{
		cacheProvider: cacheProvider,
	}
}

// Get retrieves weather data from cache
func (w *WeatherCacheAdapter) Get(ctx context.Context, key string) (*ports.WeatherData, error) {
	// Get raw data from generic cache
	data, err := w.cacheProvider.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// Deserialize JSON to WeatherData
	var weatherData ports.WeatherData
	if err := json.Unmarshal(data, &weatherData); err != nil {
		return nil, errors.NewExternalAPIError("failed to deserialize weather data", err)
	}

	return &weatherData, nil
}

// Set stores weather data in cache
func (w *WeatherCacheAdapter) Set(ctx context.Context, key string, weather *ports.WeatherData, ttl time.Duration) error {
	if weather == nil {
		return errors.NewValidationError("weather data cannot be nil")
	}

	// Serialize WeatherData to JSON
	data, err := json.Marshal(weather)
	if err != nil {
		return errors.NewExternalAPIError("failed to serialize weather data", err)
	}

	// Store in generic cache
	return w.cacheProvider.Set(ctx, key, data, ttl)
}
