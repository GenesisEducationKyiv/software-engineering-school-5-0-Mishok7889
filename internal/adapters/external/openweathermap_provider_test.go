package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/pkg/errors"
)

// Helper function to set up logger mock with variadic argument expectations
func setupLoggerMockOpenWeatherMap(t *testing.T) *mocks.Logger {
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

func TestOpenWeatherMapProvider_GetCurrentWeather_Success(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "q=London")
		assert.Contains(t, r.URL.String(), "appid=test-api-key")
		assert.Contains(t, r.URL.String(), "units=metric")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"main": {
				"temp": 15.5,
				"humidity": 78
			},
			"weather": [
				{
					"description": "light rain"
				}
			]
		}`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 15.5, weather.Temperature)
	assert.Equal(t, 78.0, weather.Humidity)
	assert.Equal(t, "light rain", weather.Description)
	assert.Equal(t, "London", weather.City)
	assert.False(t, weather.Timestamp.IsZero())
}

func TestOpenWeatherMapProvider_GetCurrentWeather_EmptyCity(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
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

func TestOpenWeatherMapProvider_GetCurrentWeather_APIError(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	// Create a mock server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte(`{"message": "Invalid API key"}`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "invalid-key",
		BaseURL: mockServer.URL,
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ExternalAPIError, appErr.Type)
	assert.Contains(t, appErr.Message, "returned status 401")
}

func TestOpenWeatherMapProvider_GetCurrentWeather_InvalidJSON(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	// Create a mock server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"invalid": json`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ExternalAPIError, appErr.Type)
	assert.Contains(t, appErr.Message, "failed to decode")
}

func TestOpenWeatherMapProvider_GetCurrentWeather_NoWeatherData(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	// Create a mock server that returns response without weather array
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"main": {
				"temp": 20.0,
				"humidity": 60
			},
			"weather": []
		}`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 20.0, weather.Temperature)
	assert.Equal(t, 60.0, weather.Humidity)
	assert.Equal(t, "Clear", weather.Description) // Default description
	assert.Equal(t, "London", weather.City)
}

func TestOpenWeatherMapProvider_GetCurrentWeather_DefaultBaseURL(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "", // Empty baseURL should use default
		Logger:  mockLogger,
	})

	// This will fail with real API, but we're testing the provider creation
	assert.Equal(t, "openweathermap", provider.GetProviderName())
}

func TestOpenWeatherMapProvider_GetProviderName(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "https://api.openweathermap.org/data/2.5",
		Logger:  mockLogger,
	})

	name := provider.GetProviderName()
	assert.Equal(t, "openweathermap", name)
}

func TestOpenWeatherMapProvider_NetworkError(t *testing.T) {
	mockLogger := setupLoggerMockOpenWeatherMap(t)

	// Use a URL that will cause a connection timeout/network error
	// Using a non-routable IP address to ensure network failure
	provider := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
		APIKey:  "test-api-key",
		BaseURL: "http://192.0.2.1:9999", // Non-routable IP address (RFC 5737)
		Logger:  mockLogger,
	})

	ctx := context.Background()
	weather, err := provider.GetCurrentWeather(ctx, "London")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ExternalAPIError, appErr.Type)
	assert.Contains(t, appErr.Message, "failed to call OpenWeatherMap")
}
