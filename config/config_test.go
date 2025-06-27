package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	weathererr "weatherapi.app/errors"
)

func TestLoadConfig(t *testing.T) {
	// Test case 1: Required fields - should return error when missing
	t.Run("RequiredFieldsMissing", func(t *testing.T) {
		os.Clearenv()

		config, err := LoadConfig()

		assert.Error(t, err)
		assert.Nil(t, config)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ConfigurationError, appErr.Type)
	})

	// Test case 2: Default values with required fields set
	t.Run("DefaultValues", func(t *testing.T) {
		os.Clearenv()

		require.NoError(t, os.Setenv("WEATHER_API_KEY", "test-api-key"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", "test-username"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", "test-password"))

		config, err := LoadConfig()

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

	// Test case 3: Custom values
	t.Run("CustomValues", func(t *testing.T) {
		os.Clearenv()

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

		config, err := LoadConfig()

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

	// Test case 4: DSN generation
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

func TestConfigValidation(t *testing.T) {
	t.Run("ServerConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			port      int
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name:    "ValidPort",
				port:    8080,
				wantErr: false,
			},
			{
				name:      "InvalidPortZero",
				port:      0,
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "SERVER_PORT must be between 1 and 65535",
			},
			{
				name:      "InvalidPortNegative",
				port:      -1,
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "SERVER_PORT must be between 1 and 65535",
			},
			{
				name:      "InvalidPortTooHigh",
				port:      65536,
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "SERVER_PORT must be between 1 and 65535",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := &ServerConfig{Port: tt.port}
				err := config.Validate()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("DatabaseConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			config    DatabaseConfig
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name: "ValidConfig",
				config: DatabaseConfig{
					Host:    "localhost",
					Port:    5432,
					User:    "user",
					Name:    "db",
					SSLMode: "disable",
				},
				wantErr: false,
			},
			{
				name: "EmptyHost",
				config: DatabaseConfig{
					Host:    "",
					Port:    5432,
					User:    "user",
					Name:    "db",
					SSLMode: "disable",
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "DB_HOST cannot be empty",
			},
			{
				name: "InvalidSSLMode",
				config: DatabaseConfig{
					Host:    "localhost",
					Port:    5432,
					User:    "user",
					Name:    "db",
					SSLMode: "invalid",
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "DB_SSL_MODE must be one of",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("WeatherConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			config    WeatherConfig
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name: "ValidConfig",
				config: WeatherConfig{
					APIKey:  "test-key",
					BaseURL: "https://api.example.com",
				},
				wantErr: false,
			},
			{
				name: "EmptyAPIKey",
				config: WeatherConfig{
					APIKey:  "",
					BaseURL: "https://api.example.com",
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "WEATHER_API_KEY is required",
			},
			{
				name: "InvalidBaseURL",
				config: WeatherConfig{
					APIKey:  "key",
					BaseURL: "invalid-url",
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "WEATHER_API_BASE_URL must start with http:// or https://",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("EmailConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			config    EmailConfig
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name: "ValidConfig",
				config: EmailConfig{
					SMTPHost:     "smtp.example.com",
					SMTPPort:     587,
					SMTPUsername: "user",
					SMTPPassword: "pass",
					FromName:     "Test",
					FromAddress:  "test@example.com",
				},
				wantErr: false,
			},
			{
				name: "InvalidEmailAddress",
				config: EmailConfig{
					SMTPHost:     "smtp.example.com",
					SMTPPort:     587,
					SMTPUsername: "user",
					SMTPPassword: "pass",
					FromName:     "Test",
					FromAddress:  "invalid-email",
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "EMAIL_FROM_ADDRESS must be a valid email address",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("SchedulerConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			config    SchedulerConfig
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name: "ValidConfig",
				config: SchedulerConfig{
					HourlyInterval: 60,
					DailyInterval:  1440,
				},
				wantErr: false,
			},
			{
				name: "InvalidHourlyInterval",
				config: SchedulerConfig{
					HourlyInterval: 0,
					DailyInterval:  1440,
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "HOURLY_INTERVAL must be at least 1 minute",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("AppBaseURL", func(t *testing.T) {
		tests := []struct {
			name      string
			baseURL   string
			wantErr   bool
			errorType weathererr.ErrorType
			errorMsg  string
		}{
			{
				name:    "ValidHTTPURL",
				baseURL: "http://localhost:8080",
				wantErr: false,
			},
			{
				name:    "ValidHTTPSURL",
				baseURL: "https://example.com",
				wantErr: false,
			},
			{
				name:      "InvalidURL",
				baseURL:   "invalid-url",
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "APP_URL must start with http:// or https://",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := &Config{AppBaseURL: tt.baseURL}
				err := config.validateAppBaseURL()

				if tt.wantErr {
					assert.Error(t, err)
					var appErr *weathererr.AppError
					assert.True(t, errors.As(err, &appErr))
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Contains(t, appErr.Message, tt.errorMsg)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("CompleteConfigValidation", func(t *testing.T) {
		config := &Config{
			Server: ServerConfig{Port: 8080},
			Database: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Name:     "db",
				SSLMode:  "disable",
			},
			Weather: WeatherConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.example.com",
			},
			Email: EmailConfig{
				SMTPHost:     "smtp.example.com",
				SMTPPort:     587,
				SMTPUsername: "user",
				SMTPPassword: "pass",
				FromName:     "Test",
				FromAddress:  "test@example.com",
			},
			Scheduler: SchedulerConfig{
				HourlyInterval: 60,
				DailyInterval:  1440,
			},
			AppBaseURL: "http://localhost:8080",
		}

		err := config.Validate()
		assert.NoError(t, err)
	})
}
