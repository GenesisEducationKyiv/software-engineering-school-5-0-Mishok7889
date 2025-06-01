package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/models"
)

// Simple test for the WeatherService
func TestWeatherService_GetWeather(t *testing.T) {
	// Create a mock HTTP server that returns a fixed response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert that the request contains the expected query parameters
		assert.Contains(t, r.URL.String(), "/current.json")
		assert.Contains(t, r.URL.String(), "q=London")

		// Return a sample weather response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			{
				"location": {
					"name": "London",
					"region": "City of London, Greater London",
					"country": "United Kingdom"
				},
				"current": {
					"temp_c": 15.0,
					"humidity": 76,
					"condition": {
						"text": "Partly cloudy"
					}
				}
			}
		`))
	}))
	defer mockServer.Close()

	// Configure the WeatherService with the mock server URL
	cfg := &config.Config{
		Weather: config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		},
	}

	// Create the service and call GetWeather
	weatherService := NewWeatherService(cfg)
	weather, err := weatherService.GetWeather("London")

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, weather)
	assert.Equal(t, 15.0, weather.Temperature)
	assert.Equal(t, 76.0, weather.Humidity)
	assert.Equal(t, "Partly cloudy", weather.Description)
}

// Test for city not found scenario
func TestWeatherService_GetWeather_CityNotFound(t *testing.T) {
	// Create a mock HTTP server that returns a 404 status
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// Configure the WeatherService with the mock server URL
	cfg := &config.Config{
		Weather: config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		},
	}

	// Create the service and call GetWeather with a non-existent city
	weatherService := NewWeatherService(cfg)
	weather, err := weatherService.GetWeather("NonExistentCity")

	// Assert the error
	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Equal(t, "city not found", err.Error())
}

// mockWeatherService for testing
type mockWeatherService struct{}

// Ensure mockWeatherService implements WeatherServiceInterface
var _ WeatherServiceInterface = (*mockWeatherService)(nil)

func (m *mockWeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	return &models.WeatherResponse{
		Temperature: 15.0,
		Humidity:    76.0,
		Description: "Partly cloudy",
	}, nil
}

// MockEmailService for testing
type mockEmailService struct{}

// Ensure mockEmailService implements EmailServiceInterface
var _ EmailServiceInterface = (*mockEmailService)(nil)

func (m *mockEmailService) SendConfirmationEmail(email, confirmURL, city string) error {
	return nil
}

func (m *mockEmailService) SendWelcomeEmail(email, city, frequency, unsubscribeURL string) error {
	return nil
}

func (m *mockEmailService) SendUnsubscribeConfirmationEmail(email, city string) error {
	return nil
}

func (m *mockEmailService) SendWeatherUpdateEmail(email, city string, weather *models.WeatherResponse, unsubscribeURL string) error {
	return nil
}

// MockTokenRepository for testing
type mockTokenRepository struct{}

// Ensure mockTokenRepository implements TokenRepositoryInterface
var _ TokenRepositoryInterface = (*mockTokenRepository)(nil)

func (m *mockTokenRepository) CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error) {
	return &models.Token{
		ID:             1,
		Token:          "test-token",
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
		CreatedAt:      time.Now(),
	}, nil
}

func (m *mockTokenRepository) FindByToken(tokenStr string) (*models.Token, error) {
	if tokenStr == "valid-token" {
		return &models.Token{
			ID:             1,
			Token:          tokenStr,
			SubscriptionID: 1,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
			CreatedAt:      time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("record not found")
}

func (m *mockTokenRepository) DeleteToken(token *models.Token) error {
	return nil
}

func (m *mockTokenRepository) DeleteExpiredTokens() error {
	return nil
}

// MockSubscriptionRepository for testing
type mockSubscriptionRepository struct{}

// Ensure mockSubscriptionRepository implements SubscriptionRepositoryInterface
var _ SubscriptionRepositoryInterface = (*mockSubscriptionRepository)(nil)

func (m *mockSubscriptionRepository) FindByEmail(email, city string) (*models.Subscription, error) {
	if email == "existing@example.com" && city == "London" {
		return &models.Subscription{
			ID:        1,
			Email:     email,
			City:      city,
			Frequency: "daily",
			Confirmed: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	return nil, nil
}

func (m *mockSubscriptionRepository) FindByID(id uint) (*models.Subscription, error) {
	if id == 1 {
		return &models.Subscription{
			ID:        id,
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("record not found")
}

func (m *mockSubscriptionRepository) Create(subscription *models.Subscription) error {
	subscription.ID = 1
	return nil
}

func (m *mockSubscriptionRepository) Update(subscription *models.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepository) Delete(subscription *models.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepository) GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error) {
	return []models.Subscription{
		{
			ID:        1,
			Email:     "test@example.com",
			City:      "London",
			Frequency: frequency,
			Confirmed: true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

// TestSubscriptionService_Subscribe tests the Subscribe method
func TestSubscriptionService_Subscribe(t *testing.T) {
	// Set up a proper in-memory database with migrations
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations to create tables
	err = db.AutoMigrate(&models.Subscription{}, &models.Token{})
	assert.NoError(t, err)

	// Create necessary mocks
	mockRepo := &mockSubscriptionRepository{}
	mockTokenRepo := &mockTokenRepository{}
	mockEmailService := &mockEmailService{}
	mockWeatherService := &mockWeatherService{}

	// Configure the service with a real DB connection
	config := &config.Config{AppBaseURL: "http://localhost:8080"}
	service := &SubscriptionService{
		db:               db,
		subscriptionRepo: mockRepo,
		tokenRepo:        mockTokenRepo,
		emailService:     mockEmailService,
		weatherService:   mockWeatherService,
		config:           config,
	}

	// Test case: New subscription
	req := &models.SubscriptionRequest{
		Email:     "new@example.com",
		City:      "Paris",
		Frequency: "daily",
	}

	err = service.Subscribe(req)
	assert.NoError(t, err)

	// Test case: Already confirmed subscription
	req = &models.SubscriptionRequest{
		Email:     "existing@example.com",
		City:      "London",
		Frequency: "hourly",
	}

	err = service.Subscribe(req)
	assert.Error(t, err)
	assert.Equal(t, "email already subscribed", err.Error())
}
