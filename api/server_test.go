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
	"weatherapi.app/models"
	"weatherapi.app/service"
)

// MockWeatherService for testing
type mockWeatherService struct {
	mock.Mock
}

// Ensure mockWeatherService implements service.WeatherServiceInterface
var _ service.WeatherServiceInterface = (*mockWeatherService)(nil)

func (m *mockWeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), args.Error(1)
}

// Test for GET /weather endpoint
func TestGetWeather(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	// Create mock weather service
	mockService := new(mockWeatherService)
	
	// Create server with mock service
	server := &Server{
		router:         router,
		weatherService: mockService,
		config:         &config.Config{},
	}
	
	// Set up routes
	router.GET("/api/weather", server.getWeather)
	
	// Configure mock to return weather data
	expectedWeather := &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}
	mockService.On("GetWeather", "London").Return(expectedWeather, nil)
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Parse response body
	var response models.WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	// Verify response data
	assert.Equal(t, expectedWeather.Temperature, response.Temperature)
	assert.Equal(t, expectedWeather.Humidity, response.Humidity)
	assert.Equal(t, expectedWeather.Description, response.Description)
	
	// Verify mock expectations
	mockService.AssertExpectations(t)
}

// Test for city not found scenario
func TestGetWeather_CityNotFound(t *testing.T) {
	// Set up test
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockService := new(mockWeatherService)
	
	server := &Server{
		router:         router,
		weatherService: mockService,
		config:         &config.Config{},
	}
	
	router.GET("/api/weather", server.getWeather)
	
	// Configure mock to return error
	mockService.On("GetWeather", "NonExistentCity").Return(nil, fmt.Errorf("city not found"))
	
	// Create request
	req := httptest.NewRequest("GET", "/api/weather?city=NonExistentCity", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Verify status code
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	// Parse error response
	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city not found", errorResponse.Error)
	
	mockService.AssertExpectations(t)
}

// Test for missing city parameter
func TestGetWeather_MissingCity(t *testing.T) {
	// Set up test
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockService := new(mockWeatherService)
	
	server := &Server{
		router:         router,
		weatherService: mockService,
		config:         &config.Config{},
	}
	
	router.GET("/api/weather", server.getWeather)
	
	// Create request without city parameter
	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city is required", errorResponse.Error)
}

// MockSubscriptionService implements a mock subscription service for testing
type mockSubscriptionService struct {
	mock.Mock
}

func (m *mockSubscriptionService) Subscribe(req *models.SubscriptionRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *mockSubscriptionService) ConfirmSubscription(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *mockSubscriptionService) Unsubscribe(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *mockSubscriptionService) SendWeatherUpdate(frequency string) error {
	args := m.Called(frequency)
	return args.Error(0)
}

// Helper function to set up a test server with mocks
func setupTestServer() (*gin.Engine, *mockWeatherService, *mockSubscriptionService) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockWeather := new(mockWeatherService)
	mockSubscription := new(mockSubscriptionService)
	
	server := &Server{
		router:              router,
		weatherService:      mockWeather,
		subscriptionService: mockSubscription,
		config:              &config.Config{AppBaseURL: "http://localhost:8080"},
	}
	
	// Set up routes
	router.GET("/api/weather", server.getWeather)
	router.POST("/api/subscribe", server.subscribe)
	router.GET("/api/confirm/:token", server.confirmSubscription)
	router.GET("/api/unsubscribe/:token", server.unsubscribe)
	
	return router, mockWeather, mockSubscription
}

// Test for POST /subscribe endpoint with valid subscription
func TestSubscribe_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()
	
	// Configure mock to return success
	mockSubscription.On("Subscribe", mock.Anything).Return(nil)
	
	// Create form data for request
	formData := "email=test%40example.com&city=London&frequency=daily"
	
	// Create test request
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription successful")
	
	// Verify mock expectations
	mockSubscription.AssertExpectations(t)
}

// Test for POST /subscribe endpoint with already subscribed email
func TestSubscribe_AlreadySubscribed(t *testing.T) {
	router, _, mockSubscription := setupTestServer()
	
	// Configure mock to return duplicate subscription error
	mockSubscription.On("Subscribe", mock.Anything).Return(fmt.Errorf("email already subscribed"))
	
	// Create form data for request
	formData := "email=test%40example.com&city=London&frequency=daily"
	
	// Create test request
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusConflict, w.Code)
	
	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	
	assert.NoError(t, err)
	assert.Equal(t, "email already subscribed", errorResponse.Error)
	
	// Verify mock expectations
	mockSubscription.AssertExpectations(t)
}

// Test for GET /confirm/:token endpoint
func TestConfirmSubscription_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()
	
	// Configure mock to return success
	token := "valid-confirmation-token"
	mockSubscription.On("ConfirmSubscription", token).Return(nil)
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription confirmed")
	
	// Verify mock expectations
	mockSubscription.AssertExpectations(t)
}

// Test for GET /confirm/:token endpoint with invalid token
func TestConfirmSubscription_InvalidToken(t *testing.T) {
	router, _, mockSubscription := setupTestServer()
	
	// Configure mock to return invalid token error
	token := "invalid-token"
	mockSubscription.On("ConfirmSubscription", token).Return(fmt.Errorf("invalid token type"))
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	
	assert.NoError(t, err)
	assert.Equal(t, "invalid token", errorResponse.Error)
	
	// Verify mock expectations
	mockSubscription.AssertExpectations(t)
}

// Test for GET /unsubscribe/:token endpoint
func TestUnsubscribe_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()
	
	// Configure mock to return success
	token := "valid-unsubscribe-token"
	mockSubscription.On("Unsubscribe", token).Return(nil)
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()
	
	// Serve the request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Unsubscribed successfully")
	
	// Verify mock expectations
	mockSubscription.AssertExpectations(t)
}
