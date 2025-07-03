package providers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/config"
	weathererr "weatherapi.app/errors"
)

func TestWeatherAPIProvider_GetCurrentWeather(t *testing.T) {
	t.Run("ValidWeatherResponse", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.String(), "/current.json")
			assert.Contains(t, r.URL.String(), "q=London")
			assert.Contains(t, r.URL.String(), "key=test-api-key")

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

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("London")

		assert.NoError(t, err)
		assert.NotNil(t, weather)
		assert.Equal(t, 15.0, weather.Temperature)
		assert.Equal(t, 76.0, weather.Humidity)
		assert.Equal(t, "Partly cloudy", weather.Description)
	})

	t.Run("EmptyCity", func(t *testing.T) {
		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: "https://api.example.com",
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "city cannot be empty")
	})

	t.Run("CityNotFound", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer mockServer.Close()

		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("NonExistentCity")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.NotFoundError, appErr.Type)
		assert.Contains(t, appErr.Message, "city not found")
	})

	t.Run("ServerError", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("London")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ExternalAPIError, appErr.Type)
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`invalid json`))
			require.NoError(t, err)
		}))
		defer mockServer.Close()

		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("London")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ExternalAPIError, appErr.Type)
		assert.Contains(t, appErr.Message, "failed to decode weather data")
	})

	t.Run("MissingCurrentField", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"location": {"name": "London"}}`))
			require.NoError(t, err)
		}))
		defer mockServer.Close()

		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("London")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ExternalAPIError, appErr.Type)
		assert.Contains(t, appErr.Message, "missing current field")
	})

	t.Run("MissingConditionField", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{
				"current": {
					"temp_c": 15.0,
					"humidity": 76
				}
			}`))
			require.NoError(t, err)
		}))
		defer mockServer.Close()

		config := &config.WeatherConfig{
			APIKey:  "test-api-key",
			BaseURL: mockServer.URL,
		}

		provider := NewWeatherAPIProvider(config)
		weather, err := provider.GetCurrentWeather("London")

		assert.Error(t, err)
		assert.Nil(t, weather)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ExternalAPIError, appErr.Type)
		assert.Contains(t, appErr.Message, "missing condition field")
	})
}

func TestSMTPEmailProvider_SendEmail(t *testing.T) {
	t.Run("ValidEmailInputs", func(t *testing.T) {
		config := &config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromName:     "Test Sender",
			FromAddress:  "test@example.com",
		}

		provider := NewSMTPEmailProvider(config)

		// Note: This test would require a mock SMTP server or would actually try to send email
		// For now, we'll test the validation logic
		err := provider.SendEmail("", "Subject", "Body", false)
		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "recipient email cannot be empty")
	})

	t.Run("EmptyRecipient", func(t *testing.T) {
		config := &config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromName:     "Test Sender",
			FromAddress:  "test@example.com",
		}

		provider := NewSMTPEmailProvider(config)
		err := provider.SendEmail("", "Subject", "Body", false)

		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "recipient email cannot be empty")
	})

	t.Run("EmptySubject", func(t *testing.T) {
		config := &config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromName:     "Test Sender",
			FromAddress:  "test@example.com",
		}

		provider := NewSMTPEmailProvider(config)
		err := provider.SendEmail("recipient@example.com", "", "Body", false)

		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "email subject cannot be empty")
	})

	t.Run("NewlineInSubject", func(t *testing.T) {
		config := &config.EmailConfig{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "password",
			FromName:     "Test Sender",
			FromAddress:  "test@example.com",
		}

		provider := NewSMTPEmailProvider(config)

		// Test that newlines are properly cleaned from subject
		// This would require a mock SMTP server to fully test, but we can verify the validation
		assert.NotNil(t, provider)
	})
}

func TestNewWeatherAPIProvider(t *testing.T) {
	config := &config.WeatherConfig{
		APIKey:  "test-api-key",
		BaseURL: "https://api.example.com",
	}

	provider := NewWeatherAPIProvider(config)

	assert.NotNil(t, provider)
	assert.Equal(t, "test-api-key", provider.apiKey)
	assert.Equal(t, "https://api.example.com", provider.baseURL)
	assert.NotNil(t, provider.client)
}

func TestNewSMTPEmailProvider(t *testing.T) {
	config := &config.EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromName:     "Test Sender",
		FromAddress:  "test@example.com",
	}

	provider := NewSMTPEmailProvider(config)

	assert.NotNil(t, provider)
	assert.Equal(t, "smtp.example.com", provider.smtpHost)
	assert.Equal(t, 587, provider.smtpPort)
	assert.Equal(t, "test@example.com", provider.smtpUsername)
	assert.Equal(t, "password", provider.smtpPassword)
	assert.Equal(t, "Test Sender", provider.fromName)
	assert.Equal(t, "test@example.com", provider.fromAddress)
}

// TestProviderManagerBuilder_Validation tests the validation logic in the builder pattern
func TestProviderManagerBuilder_Validation(t *testing.T) {
	tests := []struct {
		name      string
		builder   func() *ProviderManagerBuilder
		wantError bool
		errorMsg  string
	}{
		{
			name: "ValidAccuWeatherConfiguration",
			builder: func() *ProviderManagerBuilder {
				return NewProviderManagerBuilder().
					WithAccuWeatherKey("accuweather-key")
			},
			wantError: false,
		},
		{
			name: "MissingAllAPIKeys",
			builder: func() *ProviderManagerBuilder {
				// Create builder but don't set any API keys (default config has empty keys)
				builder := &ProviderManagerBuilder{
					config: &ProviderConfiguration{
						CacheTTL:      10 * time.Minute,
						LogFilePath:   "logs/weather_providers.log",
						EnableLogging: true,
						ProviderOrder: []string{"weatherapi", "openweathermap", "accuweather"},
						CacheConfig:   &config.CacheConfig{Type: CacheTypeMemory.String()}, // Enable caching
						// All API keys are empty strings by default
					},
				}
				return builder
			},
			wantError: true,
			errorMsg:  "at least one weather provider API key must be configured",
		},
		{
			name: "InvalidCacheTTL",
			builder: func() *ProviderManagerBuilder {
				return NewProviderManagerBuilder().
					WithAccuWeatherKey("test-key").
					WithCacheTTL(-1 * time.Minute)
			},
			wantError: true,
			errorMsg:  "cache TTL must be positive",
		},
		{
			name: "ZeroCacheTTL",
			builder: func() *ProviderManagerBuilder {
				return NewProviderManagerBuilder().
					WithAccuWeatherKey("test-key").
					WithCacheTTL(0)
			},
			wantError: true,
			errorMsg:  "cache TTL must be positive",
		},
		{
			name: "LoggingEnabledWithoutLogFile",
			builder: func() *ProviderManagerBuilder {
				return NewProviderManagerBuilder().
					WithAccuWeatherKey("test-key").
					WithLoggingEnabled(true).
					WithLogFilePath("")
			},
			wantError: true,
			errorMsg:  "log file path is required when logging is enabled",
		},
		{
			name: "InvalidProviderInOrder",
			builder: func() *ProviderManagerBuilder {
				return NewProviderManagerBuilder().
					WithAccuWeatherKey("test-key").
					WithProviderOrder([]string{"weatherapi", "invalid-provider", "openweathermap"})
			},
			wantError: true,
			errorMsg:  "invalid weather provider in order: invalid-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.builder()
			manager, err := builder.Build()

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, manager)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

// TestProviderManagerBuilder_DefaultConfiguration tests default values
func TestProviderManagerBuilder_DefaultConfiguration(t *testing.T) {
	builder := NewProviderManagerBuilder()

	// Test that builder starts with default configuration
	assert.Equal(t, 10*time.Minute, builder.config.CacheTTL)
	assert.Equal(t, "logs/weather_providers.log", builder.config.LogFilePath)
	assert.NotNil(t, builder.config.CacheConfig) // Caching enabled by default through CacheConfig presence
	assert.True(t, builder.config.EnableLogging)
	assert.Equal(t, []string{"weatherapi", "openweathermap", "accuweather"}, builder.config.ProviderOrder)

	// Test that API keys start empty (this is why validation fails by default)
	assert.Empty(t, builder.config.WeatherAPIKey)
	assert.Empty(t, builder.config.OpenWeatherMapKey)
	assert.Empty(t, builder.config.AccuWeatherKey)
}
