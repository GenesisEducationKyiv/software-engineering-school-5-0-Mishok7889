package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func setupWeatherTestRouter(t *testing.T) (*gin.Engine, *mocks.WeatherProviderManager, *mocks.WeatherCache) {
	gin.SetMode(gin.TestMode)

	// Mock the dependencies
	mockWeatherProvider := mocks.NewWeatherProviderManager(t)
	mockWeatherCache := mocks.NewWeatherCache(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := mocks.NewLogger(t)
	mockMetrics := mocks.NewWeatherMetrics(t)

	// Allow logger calls without strict expectations - handle variadic field parameters
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Mock config provider
	mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    5 * time.Minute,
	}).Maybe()

	// Create real use case with mocked dependencies
	weatherUseCase, err := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: mockWeatherProvider,
		Cache:           mockWeatherCache,
		Config:          mockConfig,
		Logger:          mockLogger,
		Metrics:         mockMetrics,
	})
	assert.NoError(t, err)

	server := &HTTPServerAdapter{
		weatherUseCase: weatherUseCase,
	}

	router := gin.New()
	router.GET("/api/weather", server.getWeather)

	return router, mockWeatherProvider, mockWeatherCache
}

func TestWeatherHandler_GetWeather_Success(t *testing.T) {
	router, mockWeatherProvider, mockWeatherCache := setupWeatherTestRouter(t)

	// Mock cache miss
	mockWeatherCache.EXPECT().
		Get(mock.Anything, "weather:London").
		Return(nil, errors.NewNotFoundError("not found"))

	// Mock weather provider response
	expectedWeatherData := &ports.WeatherData{
		Temperature: 20.5,
		Humidity:    65.0,
		Description: "Partly cloudy",
		City:        "London",
	}

	mockWeatherProvider.EXPECT().
		GetWeather(mock.Anything, "London").
		Return(expectedWeatherData, nil)

	// Mock cache set
	mockWeatherCache.EXPECT().
		Set(mock.Anything, "weather:London", mock.Anything, mock.Anything).
		Return(nil)

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedWeatherData.Temperature, response.Temperature)
	assert.Equal(t, expectedWeatherData.Humidity, response.Humidity)
	assert.Equal(t, expectedWeatherData.Description, response.Description)
	assert.Equal(t, expectedWeatherData.City, response.City)
}

func TestWeatherHandler_GetWeather_Success_FromCache(t *testing.T) {
	router, _, mockWeatherCache := setupWeatherTestRouter(t)

	// Mock cache hit
	cachedWeatherData := &ports.WeatherData{
		Temperature: 18.0,
		Humidity:    70.0,
		Description: "Cloudy",
		City:        "London",
	}

	mockWeatherCache.EXPECT().
		Get(mock.Anything, "weather:London").
		Return(cachedWeatherData, nil)

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, cachedWeatherData.Temperature, response.Temperature)
	assert.Equal(t, cachedWeatherData.Humidity, response.Humidity)
	assert.Equal(t, cachedWeatherData.Description, response.Description)
	assert.Equal(t, cachedWeatherData.City, response.City)
}

func TestWeatherHandler_GetWeather_MissingCity(t *testing.T) {
	router, _, _ := setupWeatherTestRouter(t)

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "city parameter is required")
}

func TestWeatherHandler_GetWeather_EmptyCity(t *testing.T) {
	router, _, _ := setupWeatherTestRouter(t)

	req := httptest.NewRequest("GET", "/api/weather?city=", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "city parameter is required")
}

func TestWeatherHandler_GetWeather_WeatherProviderError(t *testing.T) {
	router, mockWeatherProvider, mockWeatherCache := setupWeatherTestRouter(t)

	// Mock cache miss
	mockWeatherCache.EXPECT().
		Get(mock.Anything, "weather:InvalidCity").
		Return(nil, errors.NewNotFoundError("not found"))

	// Mock weather provider error
	mockWeatherProvider.EXPECT().
		GetWeather(mock.Anything, "InvalidCity").
		Return(nil, errors.NewExternalAPIError("city not found", nil))

	req := httptest.NewRequest("GET", "/api/weather?city=InvalidCity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "External service unavailable")
}

func TestWeatherHandler_GetWeather_ValidationError(t *testing.T) {
	router, _, _ := setupWeatherTestRouter(t)

	req := httptest.NewRequest("GET", "/api/weather?city=%20", nil) // URL encode the space
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "invalid weather request")
}

func TestWeatherHandler_GetWeather_CacheError(t *testing.T) {
	router, mockWeatherProvider, mockWeatherCache := setupWeatherTestRouter(t)

	// Mock cache miss
	mockWeatherCache.EXPECT().
		Get(mock.Anything, "weather:London").
		Return(nil, errors.NewNotFoundError("not found"))

	// Mock weather provider success
	expectedWeatherData := &ports.WeatherData{
		Temperature: 20.5,
		Humidity:    65.0,
		Description: "Partly cloudy",
		City:        "London",
	}

	mockWeatherProvider.EXPECT().
		GetWeather(mock.Anything, "London").
		Return(expectedWeatherData, nil)

	// Mock cache set error (should not fail the request)
	mockWeatherCache.EXPECT().
		Set(mock.Anything, "weather:London", mock.Anything, mock.Anything).
		Return(errors.NewDatabaseError("cache error", nil))

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still succeed despite cache error
	assert.Equal(t, http.StatusOK, w.Code)

	var response WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedWeatherData.Temperature, response.Temperature)
}
