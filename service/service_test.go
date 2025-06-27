package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/config"
	weathererr "weatherapi.app/errors"
	"weatherapi.app/models"
	"weatherapi.app/providers"
)

// Mock Weather Provider for testing
type mockWeatherProvider struct {
	mock.Mock
}

func (m *mockWeatherProvider) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), nil
}

// Mock Email Provider for testing
type mockEmailProvider struct {
	mock.Mock
}

func (m *mockEmailProvider) SendEmail(to, subject, body string, isHTML bool) error {
	args := m.Called(to, subject, body, isHTML)
	return args.Error(0)
}

// Test WeatherService with provider
func TestWeatherService_GetWeather_WithProvider(t *testing.T) {
	mockProvider := new(mockWeatherProvider)
	weatherService := NewWeatherService(mockProvider)

	expectedWeather := &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}

	mockProvider.On("GetCurrentWeather", "London").Return(expectedWeather, nil)

	weather, err := weatherService.GetWeather("London")

	assert.NoError(t, err)
	assert.Equal(t, expectedWeather, weather)
	mockProvider.AssertExpectations(t)
}

func TestWeatherService_GetWeather_EmptyCity(t *testing.T) {
	mockProvider := new(mockWeatherProvider)
	weatherService := NewWeatherService(mockProvider)

	weather, err := weatherService.GetWeather("")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.ValidationError, appErr.Type)
}

func TestWeatherService_GetWeather_ProviderError(t *testing.T) {
	mockProvider := new(mockWeatherProvider)
	weatherService := NewWeatherService(mockProvider)

	mockProvider.On("GetCurrentWeather", "InvalidCity").Return(nil, weathererr.NewNotFoundError("city not found"))

	weather, err := weatherService.GetWeather("InvalidCity")

	assert.Error(t, err)
	assert.Nil(t, weather)
	mockProvider.AssertExpectations(t)
}

// Test EmailService with provider
func TestEmailService_SendConfirmationEmailWithParams(t *testing.T) {
	mockProvider := new(mockEmailProvider)
	emailService := NewEmailService(mockProvider)

	mockProvider.On("SendEmail", "test@example.com", "Confirm your weather subscription for London", mock.AnythingOfType("string"), true).Return(nil)

	params := ConfirmationEmailParams{
		Email:      "test@example.com",
		ConfirmURL: "http://example.com/confirm/token",
		City:       "London",
	}

	err := emailService.SendConfirmationEmailWithParams(params)

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

func TestEmailService_SendConfirmationEmailWithParams_EmptyEmail(t *testing.T) {
	mockProvider := new(mockEmailProvider)
	emailService := NewEmailService(mockProvider)

	params := ConfirmationEmailParams{
		Email:      "",
		ConfirmURL: "http://example.com/confirm/token",
		City:       "London",
	}

	err := emailService.SendConfirmationEmailWithParams(params)

	assert.Error(t, err)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.ValidationError, appErr.Type)
}

// Test WeatherAPIProvider with real HTTP server
func TestWeatherAPIProvider_GetCurrentWeather(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), "/current.json")
		assert.Contains(t, r.URL.String(), "q=London")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"current": {
				"temp_c": 15.0,
				"humidity": 76,
				"condition": {
					"text": "Partly cloudy"
				}
			}
		}`))
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	config := &config.WeatherConfig{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
	}

	provider := providers.NewWeatherAPIProvider(config)
	weather, err := provider.GetCurrentWeather("London")

	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 15.0, weather.Temperature)
	assert.Equal(t, 76.0, weather.Humidity)
	assert.Equal(t, "Partly cloudy", weather.Description)
}

func TestWeatherAPIProvider_GetCurrentWeather_NotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	config := &config.WeatherConfig{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
	}

	provider := providers.NewWeatherAPIProvider(config)
	weather, err := provider.GetCurrentWeather("NonExistentCity")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.NotFoundError, appErr.Type)
}

// Mock implementations for SubscriptionService tests
type mockSubscriptionRepository struct {
	mock.Mock
}

func (m *mockSubscriptionRepository) FindByEmail(email, city string) (*models.Subscription, error) {
	args := m.Called(email, city)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), nil
}

func (m *mockSubscriptionRepository) FindByID(id uint) (*models.Subscription, error) {
	args := m.Called(id)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Subscription), nil
}

func (m *mockSubscriptionRepository) Create(subscription *models.Subscription) error {
	args := m.Called(subscription)
	subscription.ID = 1 // Set ID for testing
	return args.Error(0)
}

func (m *mockSubscriptionRepository) Update(subscription *models.Subscription) error {
	args := m.Called(subscription)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) Delete(subscription *models.Subscription) error {
	args := m.Called(subscription)
	return args.Error(0)
}

func (m *mockSubscriptionRepository) GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error) {
	args := m.Called(frequency)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Subscription), nil
}

type mockTokenRepository struct {
	mock.Mock
}

func (m *mockTokenRepository) CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error) {
	args := m.Called(subscriptionID, tokenType, expiresIn)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Token), nil
}

func (m *mockTokenRepository) FindByToken(tokenStr string) (*models.Token, error) {
	args := m.Called(tokenStr)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Token), nil
}

func (m *mockTokenRepository) DeleteToken(token *models.Token) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *mockTokenRepository) DeleteExpiredTokens() error {
	args := m.Called()
	return args.Error(0)
}

type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendConfirmationEmailWithParams(params ConfirmationEmailParams) error {
	args := m.Called(params)
	return args.Error(0)
}

func (m *mockEmailService) SendWelcomeEmailWithParams(params WelcomeEmailParams) error {
	args := m.Called(params)
	return args.Error(0)
}

func (m *mockEmailService) SendUnsubscribeConfirmationEmailWithParams(params UnsubscribeEmailParams) error {
	args := m.Called(params)
	return args.Error(0)
}

func (m *mockEmailService) SendWeatherUpdateEmailWithParams(params WeatherUpdateEmailParams) error {
	args := m.Called(params)
	return args.Error(0)
}

type mockWeatherService struct {
	mock.Mock
}

func (m *mockWeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), nil
}

// Test SubscriptionService with improved architecture
func TestSubscriptionService_Subscribe_Success(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.Subscription{}, &models.Token{})
	require.NoError(t, err)

	mockSubRepo := new(mockSubscriptionRepository)
	mockTokenRepo := new(mockTokenRepository)
	mockEmailService := new(mockEmailService)
	mockWeatherService := new(mockWeatherService)

	config := &config.Config{AppBaseURL: "http://localhost:8080"}

	service := NewSubscriptionService(
		db,
		mockSubRepo,
		mockTokenRepo,
		mockEmailService,
		mockWeatherService,
		config,
	)

	req := &models.SubscriptionRequest{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
	}

	// Setup mocks - Note: The service uses direct DB operations for transactions,
	// so we only mock the repository calls that actually happen
	mockSubRepo.On("FindByEmail", "test@example.com", "London").Return((*models.Subscription)(nil), nil)
	// CreateToken is called with subscription ID = 1 (auto-incremented)
	mockTokenRepo.On("CreateToken", uint(1), "confirmation", 24*time.Hour).Return(&models.Token{
		ID:    1,
		Token: "test-token",
	}, nil)
	mockEmailService.On("SendConfirmationEmailWithParams", ConfirmationEmailParams{
		Email:      "test@example.com",
		ConfirmURL: "http://localhost:8080/api/confirm/test-token",
		City:       "London",
	}).Return(nil)

	err = service.Subscribe(req)

	assert.NoError(t, err)
	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}

func TestSubscriptionService_Subscribe_ValidationError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	service := NewSubscriptionService(db, nil, nil, nil, nil, &config.Config{})

	req := &models.SubscriptionRequest{
		Email:     "", // Empty email should cause validation error
		City:      "London",
		Frequency: "daily",
	}

	err = service.Subscribe(req)

	assert.Error(t, err)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.ValidationError, appErr.Type)
	assert.Contains(t, appErr.Message, "email is required")
}

func TestSubscriptionService_Subscribe_AlreadyExists(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	mockSubRepo := new(mockSubscriptionRepository)

	service := NewSubscriptionService(db, mockSubRepo, nil, nil, nil, &config.Config{})

	req := &models.SubscriptionRequest{
		Email:     "existing@example.com",
		City:      "London",
		Frequency: "daily",
	}

	existingSub := &models.Subscription{
		ID:        1,
		Email:     "existing@example.com",
		City:      "London",
		Confirmed: true,
	}

	mockSubRepo.On("FindByEmail", "existing@example.com", "London").Return(existingSub, nil)

	err = service.Subscribe(req)

	assert.Error(t, err)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.AlreadyExistsError, appErr.Type)
	mockSubRepo.AssertExpectations(t)
}
