package external

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/mocks"
	appErrors "weatherapi.app/pkg/errors"
)

// Helper function to set up logger mock with variadic argument expectations
func setupLoggerMockExtended(t *testing.T) *mocks.Logger {
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

func TestWeatherProviderManager_ChainOfResponsibility_AccuWeatherSuccess(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	// Create manager with AccuWeather provider (uses mock data)
	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		ProviderOrder:  []string{"accuweather"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
	assert.Equal(t, 22.5, weather.Temperature) // AccuWeather mock data
	assert.Equal(t, 65.0, weather.Humidity)
	assert.Equal(t, "Partly cloudy", weather.Description)
}

func TestWeatherProviderManager_ChainOfResponsibility_Fallback(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	// Create manager with invalid first provider, valid second provider
	config := ProviderManagerConfig{
		OpenWeatherKey: "invalid-key", // Will fail
		AccuWeatherKey: "test-key",    // Will succeed with mock
		ProviderOrder:  []string{"openweathermap", "accuweather"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	// Should succeed with AccuWeather after OpenWeatherMap fails
	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
	assert.Equal(t, 22.5, weather.Temperature) // AccuWeather mock data
}

func TestWeatherProviderManager_AllProvidersFail(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	// Create manager with invalid providers
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

func TestWeatherProviderManager_ValidationError(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		ProviderOrder:  []string{"accuweather"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		assert.Equal(t, appErrors.ValidationError, appErr.Type)
	}
}

func TestWeatherProviderManager_NoProviders(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		Logger: mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "no weather providers configured")
}

func TestWeatherProviderManager_GetProviderInfo(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		OpenWeatherKey: "test-key",
		ProviderOrder:  []string{"accuweather", "openweathermap"},
		Logger:         mockLogger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	info := manager.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Contains(t, info, "total_providers")
	assert.Contains(t, info, "provider_order")
	assert.Contains(t, info, "chain_enabled")
	assert.Contains(t, info, "fallback_enabled")
	assert.Equal(t, 2, info["total_providers"])
	assert.Equal(t, true, info["chain_enabled"])
	assert.Equal(t, true, info["fallback_enabled"])
}

func TestWeatherProviderManager_SingleProvider(t *testing.T) {
	logger := &infrastructure.SlogLoggerAdapter{}

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		ProviderOrder:  []string{"accuweather"},
		Logger:         logger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)

	// Test provider info
	info := manager.GetProviderInfo()
	assert.Equal(t, 1, info["total_providers"])
	assert.Equal(t, false, info["fallback_enabled"]) // Only one provider, no fallback
}

func TestWeatherProviderManager_MultipleProviders(t *testing.T) {
	logger := &infrastructure.SlogLoggerAdapter{}

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		OpenWeatherKey: "test-key",
		WeatherAPIKey:  "test-key",
		ProviderOrder:  []string{"weatherapi", "openweathermap", "accuweather"},
		Logger:         logger,
	}

	manager := NewWeatherProviderManagerAdapter(config)

	// Test provider info
	info := manager.GetProviderInfo()
	assert.Equal(t, 3, info["total_providers"])
	assert.Equal(t, true, info["fallback_enabled"])

	providerOrder := info["provider_order"].([]string)
	assert.Equal(t, []string{"weatherapi", "openweathermap", "accuweather"}, providerOrder)
}
