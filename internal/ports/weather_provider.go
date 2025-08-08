package ports

import (
	"context"
	"time"
)

// WeatherData represents weather information
type WeatherData struct {
	Temperature float64
	Humidity    float64
	Description string
	City        string
	Timestamp   time.Time
}

// CacheStats represents cache performance metrics
type CacheStats struct {
	Hits        int64
	Misses      int64
	TotalOps    int64
	HitRatio    float64
	LastUpdated time.Time
}

// ProviderInfo contains information about configured weather providers
type ProviderInfo struct {
	TotalProviders  int      `json:"total_providers"`
	ProviderOrder   []string `json:"provider_order"`
	ChainEnabled    bool     `json:"chain_enabled"`
	FallbackEnabled bool     `json:"fallback_enabled"`
}

// WeatherProvider defines the contract for weather data providers
type WeatherProvider interface {
	GetCurrentWeather(ctx context.Context, city string) (*WeatherData, error)
	GetProviderName() string
}

// WeatherProviderManager defines the contract for managing multiple weather providers
type WeatherProviderManager interface {
	GetWeather(ctx context.Context, city string) (*WeatherData, error)
	GetProviderInfo() ProviderInfo
}

// WeatherCache defines the contract for caching weather data
type WeatherCache interface {
	Get(ctx context.Context, key string) (*WeatherData, error)
	Set(ctx context.Context, key string, weather *WeatherData, ttl time.Duration) error
}

// WeatherMetrics defines the contract for weather provider metrics
type WeatherMetrics interface {
	GetProviderInfo() ProviderInfo
	GetCacheMetrics() (CacheStats, error)
}
