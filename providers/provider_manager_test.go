package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/config"
)

func TestProviderManager_NoProvidersConfigured(t *testing.T) {
	config := &ProviderConfiguration{
		WeatherAPIKey:     "",
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false,
		EnableLogging:     false,
		ProviderOrder:     []string{"weatherapi", "openweathermap", "accuweather"},
		CacheType:         "memory",
		CacheConfig:       &config.CacheConfig{Type: "memory"},
	}

	// With fail-fast approach, provider manager creation should fail
	manager, err := NewProviderManager(config)
	assert.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "no weather providers configured")
	assert.Contains(t, err.Error(), "at least one API key must be provided")
}

func TestProviderManager_WithProvidersConfigured(t *testing.T) {
	config := &ProviderConfiguration{
		WeatherAPIKey:     "test-weather-api-key",
		WeatherAPIBaseURL: "https://api.weatherapi.com/v1",
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false,
		EnableLogging:     false,
		ProviderOrder:     []string{"weatherapi"},
		CacheType:         "memory",
		CacheConfig:       &config.CacheConfig{Type: "memory"},
	}

	// With at least one provider configured, creation should succeed
	manager, err := NewProviderManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Check provider info
	info := manager.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Equal(t, false, info["cache_enabled"])
	assert.Equal(t, false, info["logging_enabled"])
	assert.Equal(t, "5m0s", info["cache_ttl"])
	assert.NotEmpty(t, info["chain_name"])
}
