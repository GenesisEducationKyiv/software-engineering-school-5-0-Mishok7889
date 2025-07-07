package external

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/adapters/infrastructure"

	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/mocks"
)

func TestWeatherProviderManagerAdapter_ChainOfResponsibility(t *testing.T) {
	tests := []struct {
		name           string
		config         ProviderManagerConfig
		city           string
		expectedError  bool
		expectedSource string
	}{
		{
			name: "single_provider_success",
			config: ProviderManagerConfig{
				AccuWeatherKey: "test-key", // Only AccuWeather (mock data)
				ProviderOrder:  []string{"accuweather"},
				Logger:         &infrastructure.SlogLoggerAdapter{},
			},
			city:           "London",
			expectedError:  false,
			expectedSource: "accuweather",
		},
		{
			name: "chain_fallback_to_accuweather",
			config: ProviderManagerConfig{
				WeatherAPIKey:     "invalid-key", // Will fail
				OpenWeatherKey:    "invalid-key", // Will fail
				AccuWeatherKey:    "test-key",    // Will succeed with mock
				WeatherAPIBaseURL: "https://invalid.com",
				ProviderOrder:     []string{"weatherapi", "openweathermap", "accuweather"},
				Logger:            &infrastructure.SlogLoggerAdapter{},
			},
			city:           "London",
			expectedError:  false,
			expectedSource: "accuweather",
		},
		{
			name: "no_providers_configured",
			config: ProviderManagerConfig{
				ProviderOrder: []string{},
				Logger:        &infrastructure.SlogLoggerAdapter{},
			},
			city:          "London",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewWeatherProviderManagerAdapter(tt.config)

			ctx := context.Background()
			weather, err := manager.GetWeather(ctx, tt.city)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, weather)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, weather)
				assert.Equal(t, tt.city, weather.City)

				if tt.expectedSource == "accuweather" {
					// AccuWeather returns mock data
					assert.Equal(t, 22.5, weather.Temperature)
					assert.Equal(t, 65.0, weather.Humidity)
					assert.Equal(t, "Partly cloudy", weather.Description)
				}
			}

			// Test provider info
			info := manager.GetProviderInfo()
			assert.NotNil(t, info)
			assert.Contains(t, info, "total_providers")
			assert.Contains(t, info, "provider_order")
			assert.Contains(t, info, "chain_enabled")
			assert.Equal(t, true, info["chain_enabled"])
		})
	}
}

func TestProviderManagerConfig_Creation(t *testing.T) {
	// Test that we can create a manager with various configurations
	logger := &infrastructure.SlogLoggerAdapter{}

	configs := []ProviderManagerConfig{
		{
			WeatherAPIKey: "test-key",
			ProviderOrder: []string{"weatherapi"},
			Logger:        logger,
		},
		{
			OpenWeatherKey: "test-key",
			ProviderOrder:  []string{"openweathermap"},
			Logger:         logger,
		},
		{
			AccuWeatherKey: "test-key",
			ProviderOrder:  []string{"accuweather"},
			Logger:         logger,
		},
		{
			WeatherAPIKey:  "key1",
			OpenWeatherKey: "key2",
			AccuWeatherKey: "key3",
			ProviderOrder:  []string{"weatherapi", "openweathermap", "accuweather"},
			Logger:         logger,
		},
	}

	for i, config := range configs {
		t.Run(fmt.Sprintf("config_%d", i+1), func(t *testing.T) {
			manager := NewWeatherProviderManagerAdapter(config)
			assert.NotNil(t, manager)

			info := manager.GetProviderInfo()
			assert.Greater(t, info["total_providers"].(int), 0)
		})
	}
}

func setupLoggerMock(t *testing.T) *mocks.Logger {
	mockLogger := mocks.NewLogger(t)

	// Set up flexible mock expectations for variadic logger calls
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	return mockLogger
}

func TestWeatherProviderManagerAdapter_WithMocks(t *testing.T) {
	mockLogger := setupLoggerMock(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-accuweather-key",
		ProviderOrder:  []string{"accuweather"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
}

func TestWeatherProviderManagerAdapter_AllProvidersFail(t *testing.T) {
	mockLogger := setupLoggerMock(t)

	config := ProviderManagerConfig{
		OpenWeatherKey: "invalid-key",
		ProviderOrder:  []string{"openweathermap"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "all weather providers failed")
}

func TestWeatherProviderManagerAdapter_GetProviderInfo(t *testing.T) {
	mockLogger := setupLoggerMock(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-accuweather-key",
		ProviderOrder:  []string{"accuweather"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	info := manager.GetProviderInfo()

	assert.NotNil(t, info)
	assert.Contains(t, info, "total_providers")
	assert.Contains(t, info, "provider_order")
	assert.Contains(t, info, "chain_enabled")
	assert.Contains(t, info, "fallback_enabled")
	assert.Equal(t, 1, info["total_providers"])
	assert.Equal(t, true, info["chain_enabled"])
}
