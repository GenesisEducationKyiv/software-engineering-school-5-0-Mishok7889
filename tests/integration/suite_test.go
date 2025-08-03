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
	"weatherapi.app/internal/adapters/database"
	"weatherapi.app/internal/config"
	"weatherapi.app/tests/integration/helpers"
)

type IntegrationTestSuite struct {
	suite.Suite
	db     *gorm.DB
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
			APIKey:          "test-api-key",
			BaseURL:         "http://localhost:8081",
			EnableCache:     false,
			EnableLogging:   true, // Enable logging for tests
			CacheTTLMinutes: 5,
			LogFilePath:     "test.log",
			ProviderOrder:   []string{"weatherapi"},
		},
		Email: config.EmailConfig{
			SMTPHost:     "localhost",
			SMTPPort:     1025,
			SMTPUsername: "test@example.com",
			SMTPPassword: "test",
			FromName:     "Weather API Test",
			FromAddress:  "test@weatherapi.com",
		},
		Cache: config.CacheConfig{
			Type: config.CacheTypeMemory,
		},
		Scheduler: config.SchedulerConfig{
			HourlyInterval: 60,
			DailyInterval:  1440,
		},
		AppBaseURL: "http://localhost:8080",
	}

	s.config = testConfig

	// Initialize test database connection with retry
	var db *gorm.DB
	var err error

	s.Require().Eventually(func() bool {
		db, err = gorm.Open(postgres.Open(testConfig.Database.GetDSN()), &gorm.Config{})
		return err == nil
	}, 20*time.Second, 2*time.Second)

	s.Require().NoError(err, "Failed to connect to test database")
	s.db = db

	// Run migrations using the new database models
	err = db.AutoMigrate(
		&database.SubscriptionModel{},
		&database.TokenModel{},
	)
	s.Require().NoError(err)

	// Create a basic gin router for testing
	s.router = gin.New()
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
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
			_ = sqlDB.Close()
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

	// Wait for PostgreSQL to be ready
	testConfig := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5433,
		User:     "test_user",
		Password: "test_pass",
		Name:     "weatherapi_test",
		SSLMode:  "disable",
	}

	postgresReady := false
	s.Require().Eventually(func() bool {
		db, err := gorm.Open(postgres.Open(testConfig.GetDSN()), &gorm.Config{})
		if err != nil {
			return false
		}

		sqlDB, err := db.DB()
		if err != nil {
			return false
		}
		defer func() {
			if closeErr := sqlDB.Close(); closeErr != nil {
				slog.Warn("Failed to close database connection", "error", closeErr)
			}
		}()

		err = sqlDB.Ping()
		if err == nil {
			postgresReady = true
			return true
		}
		return false
	}, time.Duration(maxRetries)*retryDelay, retryDelay)

	if !postgresReady {
		s.T().Fatal("PostgreSQL not ready after maximum retries")
	}

	fmt.Println("Database service is ready")
}

func (s *IntegrationTestSuite) CreateTestSubscription(email, city, frequency string, confirmed bool) *database.SubscriptionModel {
	subscription := &database.SubscriptionModel{
		Email:     email,
		City:      city,
		Frequency: frequency,
		Confirmed: confirmed,
	}

	err := s.db.Create(subscription).Error
	s.Require().NoError(err)

	return subscription
}

func (s *IntegrationTestSuite) CreateTestToken(subscriptionID uint, tokenType string, expiresIn time.Duration) *database.TokenModel {
	token := &database.TokenModel{
		Token:          fmt.Sprintf("test-%s-%d-%d", tokenType, subscriptionID, time.Now().Unix()),
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
	}

	err := s.db.Create(token).Error
	s.Require().NoError(err)

	return token
}

func (s *IntegrationTestSuite) AssertSubscriptionExists(email, city string) *database.SubscriptionModel {
	var subscriptionModel database.SubscriptionModel
	err := s.db.Where("email = ? AND city = ?", email, city).First(&subscriptionModel).Error
	s.Require().NoError(err)
	return &subscriptionModel
}

func (s *IntegrationTestSuite) AssertTokenExists(subscriptionID uint, tokenType string) *database.TokenModel {
	var token database.TokenModel
	err := s.db.Where("subscription_id = ? AND type = ? AND expires_at > ?", subscriptionID, tokenType, time.Now()).First(&token).Error
	s.Require().NoError(err)
	return &token
}

func (s *IntegrationTestSuite) AssertEmailSent(to, subjectContains string) {
	sent := helpers.CheckEmailSent(to, subjectContains)
	s.Require().True(sent, "Expected email to %s with subject containing '%s' was not sent", to, subjectContains)
}

func (s *IntegrationTestSuite) TestBasicFunctionality() {
	// Basic test to ensure the suite setup works
	s.NotNil(s.db)
	s.NotNil(s.config)
	s.NotNil(s.router)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
