package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProviderManager_NoProvidersConfigured(t *testing.T) {
	// With fail-fast approach, provider manager creation should fail
	manager, err := NewProviderManagerBuilder().Build()
	assert.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "at least one weather provider API key must be configured")
}

func TestProviderManager_WithProvidersConfigured(t *testing.T) {
	config := &ProviderConfiguration{
		WeatherAPIKey:     "test-weather-api-key",
		WeatherAPIBaseURL: "https://api.weatherapi.com/v1",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableLogging:     false,
		ProviderOrder:     []string{"weatherapi"},
		CacheType:         CacheTypeMemory,
		CacheConfig:       nil, // No caching
	}

	// With at least one provider configured, creation should succeed
	manager, err := NewProviderManagerBuilder().
		WithWeatherAPIKey(config.WeatherAPIKey).
		WithWeatherAPIBaseURL(config.WeatherAPIBaseURL).
		WithCacheTTL(config.CacheTTL).
		WithLogFilePath(config.LogFilePath).
		WithLoggingEnabled(config.EnableLogging).
		WithProviderOrder(config.ProviderOrder).
		WithCacheType(config.CacheType).
		WithCacheConfig(config.CacheConfig).
		Build()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Check provider info
	info := manager.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Equal(t, false, info["cache_enabled"])
	assert.Equal(t, false, info["logging_enabled"])
	// cache_ttl should not be present when caching is disabled
	assert.Nil(t, info["cache_ttl"])
	assert.NotEmpty(t, info["chain_name"])
}
