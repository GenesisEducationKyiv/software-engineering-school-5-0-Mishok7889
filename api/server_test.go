package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/config"
	"weatherapi.app/errors"
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

// Helper function to set up a test server with mocks
func setupTestServer() (*gin.Engine, *MockWeatherService, *MockSubscriptionService) {
	gin.SetMode(gin.TestMode)

	mockWeather := new(MockWeatherService)
	mockSubscription := new(MockSubscriptionService)

	server := NewServer(
		nil, // db not needed for these tests
		&config.Config{AppBaseURL: "http://localhost:8080"},
		mockWeather,
		mockSubscription,
	)

	return server.GetRouter(), mockWeather, mockSubscription
}

// Test for GET /weather endpoint
func TestGetWeather_Success(t *testing.T) {
	router, mockWeather, _ := setupTestServer()

	expectedWeather := &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}
	mockWeather.On("GetWeather", "London").Return(expectedWeather, nil)

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, expectedWeather.Temperature, response.Temperature)
	assert.Equal(t, expectedWeather.Humidity, response.Humidity)
	assert.Equal(t, expectedWeather.Description, response.Description)

	mockWeather.AssertExpectations(t)
}

func TestGetWeather_CityNotFound(t *testing.T) {
	router, mockWeather, _ := setupTestServer()

	mockWeather.On("GetWeather", "NonExistentCity").Return(nil, errors.NewNotFoundError("city not found"))

	req := httptest.NewRequest("GET", "/api/weather?city=NonExistentCity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city not found", errorResponse.Error)

	mockWeather.AssertExpectations(t)
}

func TestGetWeather_MissingCity(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city parameter is required", errorResponse.Error)
}

func TestGetWeather_ExternalAPIError(t *testing.T) {
	router, mockWeather, _ := setupTestServer()

	mockWeather.On("GetWeather", "London").Return(nil, errors.NewExternalAPIError("weather service unavailable", nil))

	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "External service unavailable", errorResponse.Error)

	mockWeather.AssertExpectations(t)
}

func TestSubscribe_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	mockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(nil)

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription successful")

	mockSubscription.AssertExpectations(t)
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	mockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewAlreadyExistsError("email already subscribed"))

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "email already subscribed", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestSubscribe_ServiceValidationError(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	mockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewValidationError("city not supported"))

	formData := "email=test%40example.com&city=London&frequency=daily" // Valid form data
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "city not supported", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestSubscribe_BindingValidationError(t *testing.T) {
	router, _, _ := setupTestServer()

	// No mock expectation because the service should NOT be called when binding fails

	formData := "city=London&frequency=daily" // Missing required email field
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid request format", errorResponse.Error)
}

func TestSubscribe_EmailError(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	mockSubscription.On("Subscribe", mock.AnythingOfType("*models.SubscriptionRequest")).Return(errors.NewEmailError("failed to send email", nil))

	formData := "email=test%40example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "Unable to send email", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "valid-confirmation-token"
	mockSubscription.On("ConfirmSubscription", token).Return(nil)

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Subscription confirmed")

	mockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_InvalidToken(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "invalid-token"
	mockSubscription.On("ConfirmSubscription", token).Return(errors.NewTokenError("invalid token type"))

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token type", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestConfirmSubscription_NotFound(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "nonexistent-token"
	mockSubscription.On("ConfirmSubscription", token).Return(errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/confirm/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "token not found", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_Success(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "valid-unsubscribe-token"
	mockSubscription.On("Unsubscribe", token).Return(nil)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Unsubscribed successfully")

	mockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_InvalidToken(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "invalid-token"
	mockSubscription.On("Unsubscribe", token).Return(errors.NewTokenError("invalid token type"))

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token type", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

func TestUnsubscribe_NotFound(t *testing.T) {
	router, _, mockSubscription := setupTestServer()

	token := "nonexistent-token"
	mockSubscription.On("Unsubscribe", token).Return(errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "token not found", errorResponse.Error)

	mockSubscription.AssertExpectations(t)
}

// Test validation for empty token parameter
func TestConfirmSubscription_EmptyToken(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/confirm/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 404 since the route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnsubscribe_EmptyToken(t *testing.T) {
	router, _, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/unsubscribe/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 404 since the route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}
