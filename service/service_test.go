package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

// Mock Provider Manager for testing - implements WeatherProviderManagerInterface
type mockProviderManager struct {
	mock.Mock
}

func (m *mockProviderManager) GetWeather(city string) (*models.WeatherResponse, error) {
	args := m.Called(city)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WeatherResponse), nil
}

// Ensure mock implements the interface
var _ WeatherProviderManagerInterface = (*mockProviderManager)(nil)

// Mock Email Provider for testing
type mockEmailProvider struct {
	mock.Mock
}

func (m *mockEmailProvider) SendEmail(to, subject, body string, isHTML bool) error {
	args := m.Called(to, subject, body, isHTML)
	return args.Error(0)
}

// Test WeatherService with provider manager
func TestWeatherService_GetWeather_WithProviderManager(t *testing.T) {
	mockManager := new(mockProviderManager)
	weatherService := NewWeatherService(mockManager)

	expectedWeather := &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}

	mockManager.On("GetWeather", "London").Return(expectedWeather, nil)

	weather, err := weatherService.GetWeather("London")

	assert.NoError(t, err)
	assert.Equal(t, expectedWeather, weather)
	mockManager.AssertExpectations(t)
}

func TestWeatherService_GetWeather_EmptyCity(t *testing.T) {
	mockManager := new(mockProviderManager)
	weatherService := NewWeatherService(mockManager)

	weather, err := weatherService.GetWeather("")

	assert.Error(t, err)
	assert.Nil(t, weather)

	var appErr *weathererr.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, weathererr.ValidationError, appErr.Type)
}

func TestWeatherService_GetWeather_ProviderError(t *testing.T) {
	mockManager := new(mockProviderManager)
	weatherService := NewWeatherService(mockManager)

	mockManager.On("GetWeather", "InvalidCity").Return(nil, weathererr.NewNotFoundError("city not found"))

	weather, err := weatherService.GetWeather("InvalidCity")

	assert.Error(t, err)
	assert.Nil(t, weather)
	mockManager.AssertExpectations(t)
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

func (m *mockTokenRepository) FindBySubscriptionIDAndType(subscriptionID uint, tokenType string) (*models.Token, error) {
	args := m.Called(subscriptionID, tokenType)
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

// Test ProviderManager Integration (Optional - demonstrates real usage)
func TestProviderManager_Integration(t *testing.T) {
	// This test demonstrates integration but won't make actual API calls
	// Use AccuWeather since it has mock data and doesn't require base URL
	manager, err := providers.NewProviderManagerBuilder().
		WithAccuWeatherKey("test-key"). // Use AccuWeather for simple integration test
		WithCacheTTL(5 * time.Minute).
		WithLogFilePath("test.log").
		WithLoggingEnabled(false).
		WithProviderOrder([]string{"accuweather"}).
		WithCacheType(providers.CacheTypeMemory).
		WithCacheConfig(nil).
		Build()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Test that the provider info is returned correctly
	info := manager.GetProviderInfo()
	assert.NotNil(t, info)
	assert.Equal(t, false, info["cache_enabled"])
	assert.Equal(t, false, info["logging_enabled"])
	assert.Equal(t, []string{"accuweather"}, info["provider_order"])
}

func TestProviderManager_ChainOfResponsibility_Complete(t *testing.T) {
	// Test the complete Chain of Responsibility with multiple providers
	// This demonstrates the task requirements: multiple providers with fallback

	tests := []struct {
		name           string
		config         *providers.ProviderConfiguration
		expectedError  bool
		expectProvider string // Which provider should succeed
	}{
		{
			name: "Primary provider fails, secondary succeeds",
			config: &providers.ProviderConfiguration{
				WeatherAPIKey:     "",         // Disabled - will fail
				OpenWeatherMapKey: "",         // Disabled - will fail
				AccuWeatherKey:    "test-key", // Enabled - will succeed with mock
				CacheTTL:          5 * time.Minute,
				LogFilePath:       "test.log",
				EnableLogging:     false,
				ProviderOrder:     []string{"weatherapi", "openweathermap", "accuweather"},
				CacheType:         providers.CacheTypeMemory,
				CacheConfig:       &config.CacheConfig{Type: "memory"},
			},
			expectedError:  false,
			expectProvider: "accuweather", // AccuWeather uses mock data
		},
		{
			name: "All providers fail",
			config: &providers.ProviderConfiguration{
				WeatherAPIKey:     "",
				OpenWeatherMapKey: "",
				AccuWeatherKey:    "",
				CacheTTL:          5 * time.Minute,
				LogFilePath:       "test.log",
				EnableLogging:     false,
				ProviderOrder:     []string{"weatherapi", "openweathermap", "accuweather"},
				CacheType:         providers.CacheTypeMemory,
				CacheConfig:       &config.CacheConfig{Type: "memory"},
			},
			expectedError: true,
		},
		{
			name: "With caching enabled",
			config: &providers.ProviderConfiguration{
				AccuWeatherKey: "test-key",
				CacheTTL:       1 * time.Minute,
				LogFilePath:    "test.log",
				EnableLogging:  false,
				ProviderOrder:  []string{"accuweather"},
				CacheType:      providers.CacheTypeMemory,
				CacheConfig:    &config.CacheConfig{Type: "memory"}, // Cache config present = caching enabled
			},
			expectedError:  false,
			expectProvider: "accuweather",
		},
		{
			name: "With logging enabled",
			config: &providers.ProviderConfiguration{
				AccuWeatherKey: "test-key",
				CacheTTL:       5 * time.Minute,
				LogFilePath:    "test_weather.log",
				EnableLogging:  true, // Test Decorator pattern
				ProviderOrder:  []string{"accuweather"},
				CacheType:      providers.CacheTypeMemory,
				CacheConfig:    nil, // No cache config = caching disabled
			},
			expectedError:  false,
			expectProvider: "accuweather",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up log file if it exists
			if tt.config.EnableLogging {
				defer func() {
					_ = os.Remove(tt.config.LogFilePath) // Ignore cleanup errors
				}()
			}

			manager, err := providers.NewProviderManagerBuilder().
				WithWeatherAPIKey(tt.config.WeatherAPIKey).
				WithWeatherAPIBaseURL(tt.config.WeatherAPIBaseURL).
				WithOpenWeatherMapKey(tt.config.OpenWeatherMapKey).
				WithAccuWeatherKey(tt.config.AccuWeatherKey).
				WithCacheTTL(tt.config.CacheTTL).
				WithLogFilePath(tt.config.LogFilePath).
				WithLoggingEnabled(tt.config.EnableLogging).
				WithProviderOrder(tt.config.ProviderOrder).
				WithCacheType(tt.config.CacheType).
				WithCacheConfig(tt.config.CacheConfig).
				Build()

			// Handle the "All providers fail" case - now fails at creation time due to fail-fast
			if tt.name == "All providers fail" {
				assert.Error(t, err)
				assert.Nil(t, manager)
				assert.Contains(t, err.Error(), "at least one weather provider API key must be configured")
				return // Exit early for this test case
			}

			// For all other cases, manager creation should succeed
			assert.NoError(t, err)
			assert.NotNil(t, manager)

			weatherService := NewWeatherService(manager)

			// Test the chain
			weather, err := weatherService.GetWeather("London")

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, weather)
				assert.Contains(t, err.Error(), "all weather providers failed")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, weather)

				// Verify AccuWeather mock data
				if tt.expectProvider == "accuweather" {
					assert.Equal(t, 22.5, weather.Temperature)
					assert.Equal(t, 65.0, weather.Humidity)
					assert.Equal(t, "Partly cloudy", weather.Description)
				}

				// Verify provider info from manager
				info := manager.GetProviderInfo()
				assert.NotNil(t, info)
				assert.Equal(t, tt.config.CacheConfig != nil, info["cache_enabled"])
				assert.Equal(t, tt.config.EnableLogging, info["logging_enabled"])

				// Test caching if enabled
				if tt.config.CacheConfig != nil {
					// Second call should be cached
					weather2, err2 := weatherService.GetWeather("London")
					assert.NoError(t, err2)
					assert.Equal(t, weather.Temperature, weather2.Temperature)
				}

				// Verify logging if enabled
				if tt.config.EnableLogging {
					time.Sleep(100 * time.Millisecond) // Allow log writing

					_, err := os.Stat(tt.config.LogFilePath)
					assert.NoError(t, err, "Log file should exist")

					logData, err := os.ReadFile(tt.config.LogFilePath)
					assert.NoError(t, err)
					logContent := string(logData)

					assert.Contains(t, logContent, "London")
					assert.Contains(t, logContent, "response")
					assert.Contains(t, logContent, "AccuWeather")
				}
			}
		})
	}
}

func TestProviderManager_Builder_Pattern(t *testing.T) {
	// Test the Builder pattern for ProviderManager
	manager, err := providers.NewProviderManagerBuilder().
		WithAccuWeatherKey("test-key").
		WithCacheTTL(15 * time.Minute).
		WithLoggingEnabled(true).
		WithLogFilePath("test_builder.log").
		WithProviderOrder([]string{"accuweather"}).
		WithCacheType(providers.CacheTypeMemory).
		WithCacheConfig(&config.CacheConfig{Type: "memory"}).
		Build()

	defer func() {
		_ = os.Remove("test_builder.log") // Ignore cleanup errors
	}()

	assert.NoError(t, err)
	assert.NotNil(t, manager)

	info := manager.GetProviderInfo()
	assert.Equal(t, true, info["cache_enabled"])
	assert.Equal(t, true, info["logging_enabled"])
	assert.Equal(t, "15m0s", info["cache_ttl"])
	assert.Equal(t, []string{"accuweather"}, info["provider_order"])

	// Test that it actually works
	weatherService := NewWeatherService(manager)
	weather, err := weatherService.GetWeather("London")
	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 22.5, weather.Temperature)
}

func TestWeatherProviders_Individual(t *testing.T) {
	tests := []struct {
		name     string
		provider providers.WeatherProvider
		city     string
		expected *models.WeatherResponse
		hasError bool
	}{
		{
			name:     "AccuWeather with mock data",
			provider: providers.NewAccuWeatherProvider("test-key", "http://dataservice.accuweather.com/currentconditions/v1"),
			city:     "London",
			expected: &models.WeatherResponse{
				Temperature: 22.5,
				Humidity:    65.0,
				Description: "Partly cloudy",
			},
			hasError: false,
		},
		{
			name:     "AccuWeather with empty city",
			provider: providers.NewAccuWeatherProvider("test-key", "http://dataservice.accuweather.com/currentconditions/v1"),
			city:     "",
			expected: nil,
			hasError: true,
		},
		{
			name:     "OpenWeatherMap with invalid key (will fail)",
			provider: providers.NewOpenWeatherMapProvider("invalid-key", "https://api.openweathermap.org/data/2.5"),
			city:     "London",
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather, err := tt.provider.GetCurrentWeather(tt.city)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, weather)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, weather)
				if tt.expected != nil {
					assert.Equal(t, tt.expected.Temperature, weather.Temperature)
					assert.Equal(t, tt.expected.Humidity, weather.Humidity)
					assert.Equal(t, tt.expected.Description, weather.Description)
				}
			}
		})
	}
}
