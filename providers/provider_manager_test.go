package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	}

	// This should now succeed in creating the provider manager
	manager, err := NewProviderManager(config)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// But getting weather should fail with the appropriate error
	weather, err := manager.GetWeather("London")
	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "no weather providers configured")
}
