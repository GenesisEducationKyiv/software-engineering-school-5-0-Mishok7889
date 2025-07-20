package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/mocks"
)

// Helper function to set up logger mock with variadic argument expectations
func setupLoggerMockAccuWeather(t *testing.T) *mocks.Logger {
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

func TestAccuWeatherProvider_GetCurrentWeather_Success(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, DefaultTemperature, weather.Temperature)
	assert.Equal(t, DefaultHumidity, weather.Humidity)
	assert.Equal(t, DefaultMockDescription, weather.Description)
	assert.Equal(t, "London", weather.City)
	assert.False(t, weather.Timestamp.IsZero())
}

func TestAccuWeatherProvider_GetCurrentWeather_EmptyCity(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var infraErr *infrastructure.InfrastructureError
	if assert.ErrorAs(t, err, &infraErr) {
		assert.Equal(t, "VALIDATION_ERROR", infraErr.Type)
		assert.Contains(t, infraErr.Message, EmptyCityValidationMsg)
	}
}

func TestAccuWeatherProvider_Constructor_NoAPIKey(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "AccuWeather provider validation failed")
}

func TestAccuWeatherProvider_GetCurrentWeather_DefaultBaseURL(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "", // Empty baseURL should use default
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
}

func TestAccuWeatherProvider_GetProviderName(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	name := provider.GetProviderName()
	assert.Equal(t, string(AccuWeather), name)
}

func TestAccuWeatherProvider_DifferentCities(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	cities := []string{"London", "Paris", "New York", "Tokyo"}
	ctx := context.Background()

	for _, city := range cities {
		weather, err := provider.GetCurrentWeather(ctx, city)
		assert.NoError(t, err)
		assert.NotNil(t, weather)
		assert.Equal(t, city, weather.City)
		assert.Equal(t, DefaultTemperature, weather.Temperature)
		assert.Equal(t, DefaultHumidity, weather.Humidity)
		assert.Equal(t, DefaultMockDescription, weather.Description)
	}
}

func TestAccuWeatherProvider_GetCurrentWeather_NonExistentCity(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})
	require.NoError(t, err)

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "NonExistentCity")

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), CityNotFoundMsg)
}
