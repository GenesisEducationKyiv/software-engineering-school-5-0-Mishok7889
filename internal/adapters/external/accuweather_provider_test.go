package external

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/pkg/errors"
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

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 22.5, weather.Temperature)
	assert.Equal(t, 65.0, weather.Humidity)
	assert.Equal(t, "Partly cloudy", weather.Description)
	assert.Equal(t, "London", weather.City)
	assert.False(t, weather.Timestamp.IsZero())
}

func TestAccuWeatherProvider_GetCurrentWeather_EmptyCity(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ValidationError, appErr.Type)
	assert.Contains(t, appErr.Message, "city cannot be empty")
}

func TestAccuWeatherProvider_GetCurrentWeather_NoAPIKey(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ExternalAPIError, appErr.Type)
	assert.Contains(t, appErr.Message, "API key not configured")
}

func TestAccuWeatherProvider_GetCurrentWeather_DefaultBaseURL(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "", // Empty baseURL should use default
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, "London", weather.City)
}

func TestAccuWeatherProvider_GetProviderName(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	name := provider.GetProviderName()
	assert.Equal(t, "accuweather", name)
}

func TestAccuWeatherProvider_DifferentCities(t *testing.T) {
	mockLogger := setupLoggerMockAccuWeather(t)

	provider := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		Logger:  mockLogger,
	})

	cities := []string{"London", "Paris", "New York", "Tokyo"}
	ctx := context.Background()

	for _, city := range cities {
		weather, err := provider.GetCurrentWeather(ctx, city)
		assert.NoError(t, err)
		assert.NotNil(t, weather)
		assert.Equal(t, city, weather.City)
		assert.Equal(t, 22.5, weather.Temperature)
		assert.Equal(t, 65.0, weather.Humidity)
		assert.Equal(t, "Partly cloudy", weather.Description)
	}
}
