package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Test case 1: Required fields - should return error when missing
	t.Run("RequiredFieldsMissing", func(t *testing.T) {
		// Clear environment variables
		os.Clearenv()

		// Load config
		config, err := LoadConfig()

		// Verify error is returned
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "required key WEATHER_API_KEY missing")
	})

	// Test case 2: Default values - should use defaults when not provided
	t.Run("DefaultValues", func(t *testing.T) {
		// Clear environment variables
		os.Clearenv()

		// Set required fields
		require.NoError(t, os.Setenv("WEATHER_API_KEY", "test-api-key"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", "test-username"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", "test-password"))

		// Load config
		config, err := LoadConfig()

		// Verify no error and defaults are used
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, 8080, config.Server.Port)
		assert.Equal(t, "localhost", config.Database.Host)
		assert.Equal(t, 5432, config.Database.Port)
		assert.Equal(t, "postgres", config.Database.User)
		assert.Equal(t, "weatherapi", config.Database.Name)
		assert.Equal(t, "disable", config.Database.SSLMode)
		assert.Equal(t, "https://api.weatherapi.com/v1", config.Weather.BaseURL)
		assert.Equal(t, "smtp.gmail.com", config.Email.SMTPHost)
		assert.Equal(t, 587, config.Email.SMTPPort)
		assert.Equal(t, "Weather API", config.Email.FromName)
		assert.Equal(t, "no-reply@weatherapi.app", config.Email.FromAddress)
		assert.Equal(t, 60, config.Scheduler.HourlyInterval)
		assert.Equal(t, 1440, config.Scheduler.DailyInterval)
		assert.Equal(t, "http://localhost:8080", config.AppBaseURL)
	})

	// Test case 3: Custom values - should use provided values
	t.Run("CustomValues", func(t *testing.T) {
		// Clear environment variables
		os.Clearenv()

		// Set custom values
		require.NoError(t, os.Setenv("SERVER_PORT", "9090"))
		require.NoError(t, os.Setenv("DB_HOST", "test-db-host"))
		require.NoError(t, os.Setenv("DB_PORT", "5433"))
		require.NoError(t, os.Setenv("DB_USER", "test-user"))
		require.NoError(t, os.Setenv("DB_PASSWORD", "test-db-password"))
		require.NoError(t, os.Setenv("DB_NAME", "test-db"))
		require.NoError(t, os.Setenv("DB_SSL_MODE", "require"))
		require.NoError(t, os.Setenv("WEATHER_API_KEY", "custom-api-key"))
		require.NoError(t, os.Setenv("WEATHER_API_BASE_URL", "https://test-api.example.com"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_HOST", "smtp.test.com"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_PORT", "465"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", "custom-username"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", "custom-password"))
		require.NoError(t, os.Setenv("EMAIL_FROM_NAME", "Custom Name"))
		require.NoError(t, os.Setenv("EMAIL_FROM_ADDRESS", "custom@example.com"))
		require.NoError(t, os.Setenv("HOURLY_INTERVAL", "30"))
		require.NoError(t, os.Setenv("DAILY_INTERVAL", "720"))
		require.NoError(t, os.Setenv("APP_URL", "https://custom.example.com"))

		// Load config
		config, err := LoadConfig()

		// Verify no error and custom values are used
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, 9090, config.Server.Port)
		assert.Equal(t, "test-db-host", config.Database.Host)
		assert.Equal(t, 5433, config.Database.Port)
		assert.Equal(t, "test-user", config.Database.User)
		assert.Equal(t, "test-db-password", config.Database.Password)
		assert.Equal(t, "test-db", config.Database.Name)
		assert.Equal(t, "require", config.Database.SSLMode)
		assert.Equal(t, "custom-api-key", config.Weather.APIKey)
		assert.Equal(t, "https://test-api.example.com", config.Weather.BaseURL)
		assert.Equal(t, "smtp.test.com", config.Email.SMTPHost)
		assert.Equal(t, 465, config.Email.SMTPPort)
		assert.Equal(t, "custom-username", config.Email.SMTPUsername)
		assert.Equal(t, "custom-password", config.Email.SMTPPassword)
		assert.Equal(t, "Custom Name", config.Email.FromName)
		assert.Equal(t, "custom@example.com", config.Email.FromAddress)
		assert.Equal(t, 30, config.Scheduler.HourlyInterval)
		assert.Equal(t, 720, config.Scheduler.DailyInterval)
		assert.Equal(t, "https://custom.example.com", config.AppBaseURL)
	})

	// Test case 4: Test DSN generation
	t.Run("GetDSN", func(t *testing.T) {
		dbConfig := DatabaseConfig{
			Host:     "test-host",
			Port:     5432,
			User:     "test-user",
			Password: "test-password",
			Name:     "test-db",
			SSLMode:  "prefer",
		}

		expectedDSN := "host=test-host port=5432 user=test-user password=test-password dbname=test-db sslmode=prefer"
		assert.Equal(t, expectedDSN, dbConfig.GetDSN())
	})
}
