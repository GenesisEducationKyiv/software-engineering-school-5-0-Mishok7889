package external

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/mocks"
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
	assert.Equal(t, DefaultTemperature, weather.Temperature)
	assert.Equal(t, DefaultHumidity, weather.Humidity)
	assert.Equal(t, DefaultMockDescription, weather.Description)
}

func TestWeatherProviderManager_ChainOfResponsibility_Fallback(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	// Create manager with invalid first provider, valid second provider
	config := ProviderManagerConfig{
		OpenWeatherKey: "invalid-key", // Will fail
		AccuWeatherKey: "test-key",    // Will succeed with mock
		ProviderOrder:  []string{string(OpenWeatherMap), string(AccuWeather)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	// Should succeed with AccuWeather after OpenWeatherMap fails
	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
	assert.Equal(t, DefaultTemperature, weather.Temperature)
}

func TestWeatherProviderManager_AllProvidersFail(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	// Create manager with invalid providers
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

func TestWeatherProviderManager_ValidationError(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		ProviderOrder:  []string{string(AccuWeather)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var infraErr *infrastructure.InfrastructureError
	if errors.As(err, &infraErr) {
		assert.Equal(t, "VALIDATION_ERROR", infraErr.Type)
	}
}

func TestWeatherProviderManager_NoProviders(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		Logger: mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), NoProvidersConfiguredMsg)
}

func TestWeatherProviderManager_GetProviderInfo(t *testing.T) {
	mockLogger := setupLoggerMockExtended(t)

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		OpenWeatherKey: "test-key",
		ProviderOrder:  []string{string(AccuWeather), string(OpenWeatherMap)},
		Logger:         mockLogger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	info := manager.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Equal(t, 2, info.TotalProviders)
	assert.True(t, info.ChainEnabled)
	assert.True(t, info.FallbackEnabled)
}

func TestWeatherProviderManager_SingleProvider(t *testing.T) {
	logger := &infrastructure.SlogLoggerAdapter{}

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		ProviderOrder:  []string{string(AccuWeather)},
		Logger:         logger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := manager.GetWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)

	// Test provider info
	info := manager.GetProviderInfo()
	assert.Equal(t, 1, info.TotalProviders)
	assert.False(t, info.FallbackEnabled) // Only one provider, no fallback
}

func TestWeatherProviderManager_MultipleProviders(t *testing.T) {
	logger := &infrastructure.SlogLoggerAdapter{}

	config := ProviderManagerConfig{
		AccuWeatherKey: "test-key",
		OpenWeatherKey: "test-key",
		WeatherAPIKey:  "test-key",
		ProviderOrder:  []string{string(WeatherAPI), string(OpenWeatherMap), string(AccuWeather)},
		Logger:         logger,
	}

	manager, err := NewWeatherProviderManagerAdapter(config)
	require.NoError(t, err)

	// Test provider info
	info := manager.GetProviderInfo()
	assert.Equal(t, 3, info.TotalProviders)
	assert.True(t, info.FallbackEnabled)
	assert.Equal(t, []string{string(WeatherAPI), string(OpenWeatherMap), string(AccuWeather)}, info.ProviderOrder)
}
