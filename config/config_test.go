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
	// Test case 1: Required fields - should return error when no weather provider API keys are configured
	t.Run("RequiredFieldsMissing", func(t *testing.T) {
		os.Clearenv()

		// Set required email fields but no weather API keys
		require.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", "test-username"))
		require.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", "test-password"))

		config, err := LoadConfig()

		assert.Error(t, err)
		assert.Nil(t, config)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ConfigurationError, appErr.Type)
		assert.Contains(t, appErr.Message, "at least one weather provider API key must be configured")
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
		assert.Equal(t, []string{"weatherapi", "openweathermap", "accuweather"}, config.Weather.ProviderOrder)
		assert.True(t, config.Weather.EnableCache)
		assert.True(t, config.Weather.EnableLogging)
		assert.Equal(t, 10, config.Weather.CacheTTLMinutes)
		assert.Equal(t, "logs/weather_providers.log", config.Weather.LogFilePath)
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
		require.NoError(t, os.Setenv("OPENWEATHERMAP_API_KEY", "custom-openweather-key"))
		require.NoError(t, os.Setenv("ACCUWEATHER_API_KEY", "custom-accuweather-key"))
		require.NoError(t, os.Setenv("WEATHER_ENABLE_CACHE", "false"))
		require.NoError(t, os.Setenv("WEATHER_ENABLE_LOGGING", "false"))
		require.NoError(t, os.Setenv("WEATHER_CACHE_TTL_MINUTES", "30"))
		require.NoError(t, os.Setenv("WEATHER_LOG_FILE_PATH", "/custom/weather.log"))
		require.NoError(t, os.Setenv("WEATHER_PROVIDER_ORDER", "accuweather,openweathermap,weatherapi"))
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
		assert.Equal(t, "custom-openweather-key", config.Weather.OpenWeatherMapKey)
		assert.Equal(t, "custom-accuweather-key", config.Weather.AccuWeatherKey)
		assert.False(t, config.Weather.EnableCache)
		assert.False(t, config.Weather.EnableLogging)
		assert.Equal(t, 30, config.Weather.CacheTTLMinutes)
		assert.Equal(t, "/custom/weather.log", config.Weather.LogFilePath)
		assert.Equal(t, []string{"accuweather", "openweathermap", "weatherapi"}, config.Weather.ProviderOrder)
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
				name: "ValidConfigWithWeatherAPI",
				config: WeatherConfig{
					APIKey:          "test-key",
					BaseURL:         "https://api.example.com",
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"weatherapi"},
				},
				wantErr: false,
			},
			{
				name: "ValidConfigWithOpenWeatherMap",
				config: WeatherConfig{
					OpenWeatherMapKey: "openweather-key",
					CacheTTLMinutes:   10,
					ProviderOrder:     []string{"openweathermap"},
				},
				wantErr: false,
			},
			{
				name: "ValidConfigWithAccuWeather",
				config: WeatherConfig{
					AccuWeatherKey:  "accuweather-key",
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"accuweather"},
				},
				wantErr: false,
			},
			{
				name: "NoAPIKeysConfigured",
				config: WeatherConfig{
					BaseURL:         "https://api.example.com",
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"weatherapi"},
					// All API keys are empty
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "at least one weather provider API key must be configured",
			},
			{
				name: "WeatherAPIKeyWithoutBaseURL",
				config: WeatherConfig{
					APIKey:          "test-key",
					BaseURL:         "", // Empty base URL
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"weatherapi"},
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "WEATHER_API_BASE_URL cannot be empty when WEATHER_API_KEY is set",
			},
			{
				name: "InvalidBaseURL",
				config: WeatherConfig{
					APIKey:          "key",
					BaseURL:         "invalid-url",
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"weatherapi"},
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "WEATHER_API_BASE_URL must start with http:// or https://",
			},
			{
				name: "InvalidCacheTTL",
				config: WeatherConfig{
					APIKey:          "test-key",
					BaseURL:         "https://api.example.com",
					CacheTTLMinutes: 0, // Invalid cache TTL
					ProviderOrder:   []string{"weatherapi"},
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "WEATHER_CACHE_TTL_MINUTES must be between 1 and 1440 minutes",
			},
			{
				name: "InvalidProviderOrder",
				config: WeatherConfig{
					APIKey:          "test-key",
					BaseURL:         "https://api.example.com",
					CacheTTLMinutes: 10,
					ProviderOrder:   []string{"invalid-provider"},
				},
				wantErr:   true,
				errorType: weathererr.ConfigurationError,
				errorMsg:  "invalid weather provider in order: invalid-provider",
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
				APIKey:          "test-key",
				BaseURL:         "https://api.example.com",
				CacheTTLMinutes: 10,
				ProviderOrder:   []string{"weatherapi", "openweathermap", "accuweather"},
				EnableCache:     true,
				EnableLogging:   true,
				LogFilePath:     "logs/weather.log",
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

func TestEmailConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      EmailConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config with credentials",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "user@gmail.com",
				SMTPPassword: "password",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: false,
		},
		{
			name: "Valid config without credentials (MailHog)",
			config: EmailConfig{
				SMTPHost:     "mailhog",
				SMTPPort:     1025,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: false,
		},
		{
			name: "Invalid - only username provided",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "user@gmail.com",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty",
		},
		{
			name: "Invalid - only password provided",
			config: EmailConfig{
				SMTPHost:     "smtp.gmail.com",
				SMTPPort:     587,
				SMTPUsername: "",
				SMTPPassword: "password",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty",
		},
		{
			name: "Invalid - empty host",
			config: EmailConfig{
				SMTPHost:     "",
				SMTPPort:     587,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "test@example.com",
			},
			expectError: true,
			errorMsg:    "EMAIL_SMTP_HOST cannot be empty",
		},
		{
			name: "Invalid - invalid from address",
			config: EmailConfig{
				SMTPHost:     "mailhog",
				SMTPPort:     1025,
				SMTPUsername: "",
				SMTPPassword: "",
				FromName:     "Test App",
				FromAddress:  "invalid-email",
			},
			expectError: true,
			errorMsg:    "EMAIL_FROM_ADDRESS must be a valid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				var appErr *weathererr.AppError
				assert.True(t, errors.As(err, &appErr))
				assert.Equal(t, weathererr.ConfigurationError, appErr.Type)
				assert.Contains(t, appErr.Message, tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_LoadWithMailHogCredentials(t *testing.T) {
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			if len(env) <= 0 {
				continue
			}
			parts := []string{"", ""}
			for i, part := range []rune(env) {
				if part == '=' {
					parts = []string{string([]rune(env)[:i]), string([]rune(env)[i+1:])}
					break
				}
			}
			if len(parts) != 2 || parts[0] == "" {
				continue
			}
			_ = os.Setenv(parts[0], parts[1])
		}
	}()

	// Clear environment and set minimal required values for MailHog testing
	os.Clearenv()
	assert.NoError(t, os.Setenv("WEATHER_API_KEY", "test-api-key"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_HOST", "mailhog"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_PORT", "1025"))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_USERNAME", ""))
	assert.NoError(t, os.Setenv("EMAIL_SMTP_PASSWORD", ""))
	assert.NoError(t, os.Setenv("EMAIL_FROM_ADDRESS", "test@example.com"))
	assert.NoError(t, os.Setenv("DB_HOST", "localhost"))
	assert.NoError(t, os.Setenv("DB_USER", "testuser"))
	assert.NoError(t, os.Setenv("DB_NAME", "testdb"))

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "mailhog", config.Email.SMTPHost)
	assert.Equal(t, 1025, config.Email.SMTPPort)
	assert.Equal(t, "", config.Email.SMTPUsername)
	assert.Equal(t, "", config.Email.SMTPPassword)
}
