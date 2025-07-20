package external

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				ProviderOrder:  []string{string(AccuWeather)},
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
				ProviderOrder:     []string{string(WeatherAPI), string(OpenWeatherMap), string(AccuWeather)},
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
			manager, err := NewWeatherProviderManagerAdapter(tt.config)
			require.NoError(t, err)

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
					assert.Equal(t, DefaultTemperature, weather.Temperature)
					assert.Equal(t, DefaultHumidity, weather.Humidity)
					assert.Equal(t, DefaultMockDescription, weather.Description)
				}
			}

			// Test provider info
			info := manager.GetProviderInfo()
			assert.NotNil(t, info)
			assert.GreaterOrEqual(t, info.TotalProviders, 0)
			assert.True(t, info.ChainEnabled)
		})
	}
}

func TestProviderManagerConfig_Creation(t *testing.T) {
	// Test that we can create a manager with various configurations
	logger := &infrastructure.SlogLoggerAdapter{}

	configs := []ProviderManagerConfig{
		{
			WeatherAPIKey: "test-key",
			ProviderOrder: []string{string(WeatherAPI)},
			Logger:        logger,
		},
		{
			OpenWeatherKey: "test-key",
			ProviderOrder:  []string{string(OpenWeatherMap)},
			Logger:         logger,
		},
		{
			AccuWeatherKey: "test-key",
			ProviderOrder:  []string{string(AccuWeather)},
			Logger:         logger,
		},
		{
			WeatherAPIKey:  "key1",
			OpenWeatherKey: "key2",
			AccuWeatherKey: "key3",
			ProviderOrder:  []string{string(WeatherAPI), string(OpenWeatherMap), string(AccuWeather)},
			Logger:         logger,
		},
	}

	for i, config := range configs {
		t.Run(fmt.Sprintf("config_%d", i+1), func(t *testing.T) {
			manager, err := NewWeatherProviderManagerAdapter(config)
			require.NoError(t, err)
			assert.NotNil(t, manager)

			info := manager.GetProviderInfo()
			assert.Greater(t, info.TotalProviders, 0)
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
		ProviderOrder:  []string{string(AccuWeather)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

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
		ProviderOrder:  []string{string(OpenWeatherMap)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

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
		ProviderOrder:  []string{string(AccuWeather)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	info := manager.GetProviderInfo()

	assert.NotNil(t, info)
	assert.Equal(t, 1, info.TotalProviders)
	assert.True(t, info.ChainEnabled)
}
