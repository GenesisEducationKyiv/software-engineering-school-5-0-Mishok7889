package external

import (
	"context"
	"encoding/json"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
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
	data, err := w.cacheProvider.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var weatherData ports.WeatherData
	if err := json.Unmarshal(data, &weatherData); err != nil {
		return nil, infrastructure.NewExternalAPIError("failed to deserialize weather data", err)
	}

	return &weatherData, nil
}

func (w *WeatherCacheAdapter) Set(ctx context.Context, key string, weather *ports.WeatherData, ttl time.Duration) error {
	if weather == nil {
		return infrastructure.NewValidationError("weather data cannot be nil")
	}

	data, err := json.Marshal(weather)
	if err != nil {
		return infrastructure.NewExternalAPIError("failed to serialize weather data", err)
	}

	return w.cacheProvider.Set(ctx, key, data, ttl)
}
