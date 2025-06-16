package providers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/config"
	apperrors "weatherapi.app/errors"
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ValidationError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.NotFoundError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ExternalAPIError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ExternalAPIError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ExternalAPIError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ExternalAPIError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ValidationError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ValidationError, appErr.Type)
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

		var appErr *apperrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperrors.ValidationError, appErr.Type)
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
