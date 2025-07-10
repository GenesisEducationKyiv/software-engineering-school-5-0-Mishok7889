package providers

import (
	"time"

	"weatherapi.app/metrics"
	"weatherapi.app/models"
	"weatherapi.app/providers/cache"
)

// WeatherProvider defines the interface for weather data providers
type WeatherProvider interface {
	GetCurrentWeather(city string) (*models.WeatherResponse, error)
}

// WeatherProviderChain defines the interface for Chain of Responsibility pattern
type WeatherProviderChain interface {
	Handle(city string) (*models.WeatherResponse, error)
	SetNext(handler WeatherProviderChain)
	GetProviderName() string
}

// Cache is an alias to avoid circular imports
type Cache = cache.Cache

// FileLogger defines the interface for file logging operations
type FileLogger interface {
	LogRequest(providerName, city string)
	LogResponse(providerName, city string, response *models.WeatherResponse, duration time.Duration)
	LogError(providerName, city string, err error, duration time.Duration)
}

// EmailProvider defines the interface for email providers
type EmailProvider interface {
	SendEmail(to, subject, body string, isHTML bool) error
}

// WeatherManager defines the interface for weather provider management
type WeatherManager interface {
	GetWeather(city string) (*models.WeatherResponse, error)
}

type WeatherProviderMetrics interface {
	GetProviderInfo() map[string]interface{}
	GetCacheMetrics() (metrics.CacheStats, error)
}
