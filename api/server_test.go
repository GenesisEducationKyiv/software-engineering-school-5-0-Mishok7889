package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/config"
	"weatherapi.app/errors"
	"weatherapi.app/metrics"
	"weatherapi.app/models"
)

// MockWeatherService for testing
type MockWeatherService struct {
	mock.Mock
}

func (m *MockWeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), args.Error(1)
}

// MockSubscriptionService for testing
type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) Subscribe(req *models.SubscriptionRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockSubscriptionService) ConfirmSubscription(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSubscriptionService) Unsubscribe(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSubscriptionService) SendWeatherUpdate(frequency string) error {
	args := m.Called(frequency)
	return args.Error(0)
}

// MockProviderManager for testing
type MockProviderManager struct {
	mock.Mock
}

func (m *MockProviderManager) GetWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), args.Error(1)
}

// MockProviderMetricsService for testing
type MockProviderMetricsService struct {
	mock.Mock
}

func (m *MockProviderMetricsService) GetProviderInfo() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockProviderMetricsService) GetCacheMetrics() (metrics.CacheStats, error) {
	args := m.Called()
	return args.Get(0).(metrics.CacheStats), args.Error(1)
}

// TestServerSetup contains all the components needed for testing
type TestServerSetup struct {
	Router              *gin.Engine
	MockWeather         *MockWeatherService
	MockSubscription    *MockSubscriptionService
	MockProviderManager *MockProviderManager
	MockProviderMetrics *MockProviderMetricsService
}

// Helper function to set up a test server with mocks
func setupTestServer() *TestServerSetup {
	gin.SetMode(gin.TestMode)

	mockWeather := new(MockWeatherService)
	mockSubscription := new(MockSubscriptionService)
	mockProviderManager := new(MockProviderManager)
	mockProviderMetrics := new(MockProviderMetricsService)

	server, err := NewServer(ServerOptions{
		DB:                  nil, // db not needed for these tests
		Config:              &config.Config{AppBaseURL: "http://localhost:8080"},
		WeatherService:      mockWeather,
		SubscriptionService: mockSubscription,
		ProviderManager:     mockProviderManager,
		ProviderMetrics:     mockProviderMetrics,
	})
	if err != nil {
		panic("Failed to create test server: " + err.Error())
	}

	return &TestServerSetup{
		Router:              server.GetRouter(),
		MockWeather:         mockWeather,
		MockSubscription:    mockSubscription,
		MockProviderManager: mockProviderManager,
		MockProviderMetrics: mockProviderMetrics,
	}
}

// Test for GET /weather endpoint
func TestGetWeather_Success(t *testing.T) {
	setup := setupTestServer()

	expectedWeather := &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}
	setup.MockWeather.On("GetWeather", "London").Return(expectedWeather, nil)

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, expectedWeather.Temperature, response.Temperature)
	assert.Equal(t, expectedWeather.Humidity, response.Humidity)
	assert.Equal(t, expectedWeather.Description, response.Description)

	setup.MockWeather.AssertExpectations(t)
}

func TestGetWeather_CityNotFound(t *testing.T) {
	setup := setupTestServer()

	setup.MockWeather.On("GetWeather", "NonExistentCity").Return(nil, errors.NewNotFoundError("city not found"))

	req := httptest.NewRequest("GET", "/api/weather?city=NonExistentCity", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city not found", errorResponse.Error)

	setup.MockWeather.AssertExpectations(t)
}

func TestGetWeather_MissingCity(t *testing.T) {
	setup := setupTestServer()

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city parameter is required", errorResponse.Error)
}

func TestGetWeather_ExternalAPIError(t *testing.T) {
	setup := setupTestServer()

	setup.MockWeather.On("GetWeather", "London").Return(nil, errors.NewExternalAPIError("weather service unavailable", nil))

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "External service unavailable", errorResponse.Error)

	setup.MockWeather.AssertExpectations(t)
}

func TestSubscribe_Success(t *testing.T) {
	setup := setupTestServer()

	setup.MockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(nil)

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription successful")

	setup.MockSubscription.AssertExpectations(t)
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	setup := setupTestServer()

	setup.MockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewAlreadyExistsError("email already subscribed"))

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "email already subscribed", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestSubscribe_ServiceValidationError(t *testing.T) {
	setup := setupTestServer()

	setup.MockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewValidationError("city not supported"))

	formData := "email=test%40example.com&city=London&frequency=daily" // Valid form data
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city not supported", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestSubscribe_BindingValidationError(t *testing.T) {
	setup := setupTestServer()

	// No mock expectation because the service should NOT be called when binding fails

	formData := "city=London&frequency=daily" // Missing required email field
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid request format", errorResponse.Error)
}

func TestSubscribe_EmailError(t *testing.T) {
	setup := setupTestServer()

	setup.MockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewEmailError("failed to send email", nil))

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Unable to send email", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_Success(t *testing.T) {
	setup := setupTestServer()

	token := "valid-confirmation-token"
	setup.MockSubscription.On("ConfirmSubscription", token).Return(nil)

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription confirmed")

	setup.MockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_InvalidToken(t *testing.T) {
	setup := setupTestServer()

	token := "invalid-token"
	setup.MockSubscription.On("ConfirmSubscription", token).Return(errors.NewTokenError("invalid token type"))

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token type", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_NotFound(t *testing.T) {
	setup := setupTestServer()

	token := "nonexistent-token"
	setup.MockSubscription.On("ConfirmSubscription", token).Return(errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "token not found", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_Success(t *testing.T) {
	setup := setupTestServer()

	token := "valid-unsubscribe-token"
	setup.MockSubscription.On("Unsubscribe", token).Return(nil)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Unsubscribed successfully")

	setup.MockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_InvalidToken(t *testing.T) {
	setup := setupTestServer()

	token := "invalid-token"
	setup.MockSubscription.On("Unsubscribe", token).Return(errors.NewTokenError("invalid token type"))

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token type", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_NotFound(t *testing.T) {
	setup := setupTestServer()

	token := "nonexistent-token"
	setup.MockSubscription.On("Unsubscribe", token).Return(errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "token not found", errorResponse.Error)

	setup.MockSubscription.AssertExpectations(t)
}

// Test validation for empty token parameter
func TestConfirmSubscription_EmptyToken(t *testing.T) {
	setup := setupTestServer()

	req := httptest.NewRequest("GET", "/api/confirm/", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	// Should return 404 since the route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test ServerOptions validation
func TestServerOptions_Validation(t *testing.T) {
	tests := []struct {
		name        string
		opts        ServerOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid options",
			opts: ServerOptions{
				DB:                  nil,
				Config:              &config.Config{},
				WeatherService:      new(MockWeatherService),
				SubscriptionService: new(MockSubscriptionService),
				ProviderManager:     new(MockProviderManager),
				ProviderMetrics:     new(MockProviderMetricsService),
			},
			expectError: false,
		},
		{
			name: "Missing config",
			opts: ServerOptions{
				DB:                  nil,
				Config:              nil,
				WeatherService:      new(MockWeatherService),
				SubscriptionService: new(MockSubscriptionService),
				ProviderManager:     new(MockProviderManager),
			},
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "Missing weather service",
			opts: ServerOptions{
				DB:                  nil,
				Config:              &config.Config{},
				WeatherService:      nil,
				SubscriptionService: new(MockSubscriptionService),
				ProviderManager:     new(MockProviderManager),
				ProviderMetrics:     new(MockProviderMetricsService),
			},
			expectError: true,
			errorMsg:    "weather service is required",
		},
		{
			name: "Missing subscription service",
			opts: ServerOptions{
				DB:                  nil,
				Config:              &config.Config{},
				WeatherService:      new(MockWeatherService),
				SubscriptionService: nil,
				ProviderManager:     new(MockProviderManager),
				ProviderMetrics:     new(MockProviderMetricsService),
			},
			expectError: true,
			errorMsg:    "subscription service is required",
		},
		{
			name: "Missing provider manager",
			opts: ServerOptions{
				DB:                  nil,
				Config:              &config.Config{},
				WeatherService:      new(MockWeatherService),
				SubscriptionService: new(MockSubscriptionService),
				ProviderManager:     nil,
				ProviderMetrics:     new(MockProviderMetricsService),
			},
			expectError: true,
			errorMsg:    "provider manager is required",
		},
		{
			name: "Missing provider metrics",
			opts: ServerOptions{
				DB:                  nil,
				Config:              &config.Config{},
				WeatherService:      new(MockWeatherService),
				SubscriptionService: new(MockSubscriptionService),
				ProviderManager:     new(MockProviderManager),
				ProviderMetrics:     nil,
			},
			expectError: true,
			errorMsg:    "provider metrics is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewServer_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test that NewServer returns error when validation fails
	server, err := NewServer(ServerOptions{
		Config: nil, // Missing required config
	})

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "invalid server options")
	assert.Contains(t, err.Error(), "config is required")
}

// Test for the new metrics endpoint
func TestMetricsEndpoint_Success(t *testing.T) {
	setup := setupTestServer()

	// Set up mock expectations
	expectedCacheStats := metrics.CacheStats{
		CacheType: "memory",
		Hits:      100,
		Misses:    25,
		Total:     125,
		HitRatio:  0.8,
	}
	setup.MockProviderMetrics.On("GetCacheMetrics").Return(expectedCacheStats, nil)
	setup.MockProviderMetrics.On("GetProviderInfo").Return(map[string]interface{}{
		"cache_enabled": true,
		"cache_type":    "memory",
	})

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response structure
	assert.Contains(t, response, "cache")
	assert.Contains(t, response, "provider_info")
	assert.Contains(t, response, "endpoints")

	endpoints := response["endpoints"].(map[string]interface{})
	assert.Equal(t, "/metrics", endpoints["prometheus_metrics"])
	assert.Equal(t, "/api/metrics", endpoints["cache_metrics"])
	setup.MockProviderMetrics.AssertExpectations(t)
}

// Test for the metrics endpoint error case
func TestMetricsEndpoint_CacheError(t *testing.T) {
	setup := setupTestServer()

	// Set up mock expectations for error case
	setup.MockProviderMetrics.On("GetCacheMetrics").Return(metrics.CacheStats{}, fmt.Errorf("cache not enabled"))

	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify error response
	assert.Contains(t, response, "error")
	assert.Equal(t, "cache metrics unavailable", response["error"])
	setup.MockProviderMetrics.AssertExpectations(t)
}

func TestUnsubscribe_EmptyToken(t *testing.T) {
	setup := setupTestServer()

	req := httptest.NewRequest("GET", "/api/unsubscribe/", nil)
	w := httptest.NewRecorder()

	setup.Router.ServeHTTP(w, req)

	// Should return 404 since the route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}
