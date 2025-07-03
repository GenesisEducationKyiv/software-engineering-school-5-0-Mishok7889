package integration

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"weatherapi.app/api"
	"weatherapi.app/config"
	"weatherapi.app/database"
	"weatherapi.app/models"
	"weatherapi.app/providers"
	"weatherapi.app/repository"
	"weatherapi.app/service"
	"weatherapi.app/tests/integration/helpers"
)

type IntegrationTestSuite struct {
	suite.Suite
	db     *gorm.DB
	server *api.Server
	router *gin.Engine
	config *config.Config
}

func (s *IntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Wait for services to be ready before initializing
	s.waitForServices()

	testConfig := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5433,
			User:     "test_user",
			Password: "test_pass",
			Name:     "weatherapi_test",
			SSLMode:  "disable",
		},
		Weather: config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: "http://localhost:8081",
		},
		Email: config.EmailConfig{
			SMTPHost:     "localhost",
			SMTPPort:     1025,
			SMTPUsername: "test@example.com",
			SMTPPassword: "test",
			FromName:     "Weather API Test",
			FromAddress:  "test@weatherapi.com",
		},
		AppBaseURL: "http://localhost:8080",
	}

	s.config = testConfig

	// Retry database connection
	var db *gorm.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(testConfig.Database.GetDSN()), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	s.Require().NoError(err, "Failed to connect to test database after retries")
	s.db = db

	err = database.RunMigrations(db)
	s.Require().NoError(err)

	// Create provider manager instead of individual provider
	providerConfig := &providers.ProviderConfiguration{
		WeatherAPIKey:     testConfig.Weather.APIKey,
		WeatherAPIBaseURL: testConfig.Weather.BaseURL, // Use mock API URL
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false, // Disable cache for testing
		EnableLogging:     false, // Disable logging for testing
		ProviderOrder:     []string{"weatherapi"},
		CacheType:         providers.CacheTypeMemory,
		CacheConfig:       &config.CacheConfig{Type: "memory"},
	}

	providerManager, err := providers.NewProviderManager(providerConfig)
	s.Require().NoError(err)

	emailProvider := providers.NewSMTPEmailProvider(&testConfig.Email)

	weatherService := service.NewWeatherService(providerManager)
	emailService := service.NewEmailService(emailProvider)

	subscriptionRepo := repository.NewSubscriptionRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	subscriptionService := service.NewSubscriptionService(
		db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		testConfig,
	)

	server, err := api.NewServer(
		api.NewServerOptionsBuilder().
			WithDB(db).
			WithConfig(testConfig).
			WithWeatherService(weatherService).
			WithSubscriptionService(subscriptionService).
			WithProviderManager(providerManager).
			WithProviderMetrics(providerManager).
			Build(),
	)
	s.Require().NoError(err)
	s.server = server
	s.router = s.server.GetRouter()
}

func (s *IntegrationTestSuite) SetupTest() {
	s.cleanDatabase()
}

func (s *IntegrationTestSuite) TearDownTest() {
	s.cleanDatabase()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				slog.Warn("Failed to close database connection", "error", closeErr)
			}
		}
	}
}

func (s *IntegrationTestSuite) cleanDatabase() {
	s.db.Exec("DELETE FROM tokens")
	s.db.Exec("DELETE FROM subscriptions")
}

func (s *IntegrationTestSuite) waitForServices() {
	maxRetries := 30
	retryDelay := 2 * time.Second

	fmt.Println("Waiting for integration test services to be ready...")

	// Wait for PostgreSQL
	postgresReady := false
	for i := 0; i < maxRetries; i++ {
		testConfig := config.DatabaseConfig{
			Host:     "localhost",
			Port:     5433,
			User:     "test_user",
			Password: "test_pass",
			Name:     "weatherapi_test",
			SSLMode:  "disable",
		}
		db, err := gorm.Open(postgres.Open(testConfig.GetDSN()), &gorm.Config{})
		if err == nil {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				err = sqlDB.Ping()
				if err == nil {
					if closeErr := sqlDB.Close(); closeErr != nil {
						slog.Warn("Failed to close database connection", "error", closeErr)
					}
					postgresReady = true
					break
				}
				if closeErr := sqlDB.Close(); closeErr != nil {
					slog.Warn("Failed to close database connection", "error", closeErr)
				}
			}
		}
		fmt.Printf("Waiting for PostgreSQL... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(retryDelay)
	}

	if !postgresReady {
		s.T().Fatal("PostgreSQL not ready after maximum retries")
	}

	// Wait for Mock Weather API
	weatherReady := false
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:8081/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			if closeErr := resp.Body.Close(); closeErr != nil {
				slog.Warn("Failed to close response body", "error", closeErr)
			}
			weatherReady = true
			break
		}
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				slog.Warn("Failed to close response body", "error", closeErr)
			}
		}
		time.Sleep(retryDelay)
	}

	if !weatherReady {
		s.T().Fatal("Mock Weather API not ready after maximum retries")
	}

	// Wait for MailHog
	mailReady := false
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:8025")
		if err == nil && resp.StatusCode == http.StatusOK {
			if closeErr := resp.Body.Close(); closeErr != nil {
				slog.Warn("Failed to close response body", "error", closeErr)
			}
			mailReady = true
			break
		}
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				slog.Warn("Failed to close response body", "error", closeErr)
			}
		}
		time.Sleep(retryDelay)
	}

	if !mailReady {
		s.T().Fatal("MailHog not ready after maximum retries")
	}

	fmt.Println("All integration test services are ready")
}

func (s *IntegrationTestSuite) CreateTestSubscription(email, city, frequency string, confirmed bool) *models.Subscription {
	subscription := &models.Subscription{
		Email:     email,
		City:      city,
		Frequency: frequency,
		Confirmed: confirmed,
	}

	err := s.db.Create(subscription).Error
	s.Require().NoError(err)

	return subscription
}

func (s *IntegrationTestSuite) CreateTestToken(subscriptionID uint, tokenType string, expiresIn time.Duration) *models.Token {
	token := &models.Token{
		Token:          fmt.Sprintf("test-%s-%d-%d", tokenType, subscriptionID, time.Now().Unix()),
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
	}

	err := s.db.Create(token).Error
	s.Require().NoError(err)

	return token
}

func (s *IntegrationTestSuite) AssertSubscriptionExists(email, city string) *models.Subscription {
	var subscription models.Subscription
	err := s.db.Where("email = ? AND city = ?", email, city).First(&subscription).Error
	s.Require().NoError(err)
	return &subscription
}

func (s *IntegrationTestSuite) AssertTokenExists(subscriptionID uint, tokenType string) *models.Token {
	var token models.Token
	err := s.db.Where("subscription_id = ? AND type = ? AND expires_at > ?", subscriptionID, tokenType, time.Now()).First(&token).Error
	s.Require().NoError(err)
	return &token
}

func (s *IntegrationTestSuite) AssertEmailSent(to, subjectContains string) {
	sent := helpers.CheckEmailSent(to, subjectContains)
	s.Require().True(sent, "Expected email to %s with subject containing '%s' was not sent", to, subjectContains)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
