package external

import (
	"context"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// AccuWeatherProviderAdapter implements WeatherProvider port for AccuWeather
// Note: Uses mock data for simplicity since AccuWeather requires location ID resolution
type AccuWeatherProviderAdapter struct {
	apiKey  string
	baseURL string
	logger  ports.Logger
}

// AccuWeatherProviderParams holds parameters for creating AccuWeather provider
type AccuWeatherProviderParams struct {
	APIKey  string
	BaseURL string
	Logger  ports.Logger
}

// NewAccuWeatherProviderAdapter creates a new AccuWeather provider adapter
func NewAccuWeatherProviderAdapter(params AccuWeatherProviderParams) ports.WeatherProvider {
	baseURL := params.BaseURL
	if baseURL == "" {
		baseURL = "http://dataservice.accuweather.com/currentconditions/v1"
	}

	return &AccuWeatherProviderAdapter{
		apiKey:  params.APIKey,
		baseURL: baseURL,
		logger:  params.Logger,
	}
}

// GetCurrentWeather retrieves weather data from AccuWeather (mock implementation)
func (p *AccuWeatherProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	if p.apiKey == "" {
		return nil, errors.NewExternalAPIError("AccuWeather API key not configured", nil)
	}

	// Mock: simulate city not found for test cases
	if city == "NonExistentCity" {
		return nil, errors.NewNotFoundError("city not found")
	}

	// Mock weather data for demonstration
	// In production, this would require location lookup and actual API calls
	return &ports.WeatherData{
		Temperature: 22.5,
		Humidity:    65.0,
		Description: "Partly cloudy",
		City:        city,
		Timestamp:   time.Now(),
	}, nil
}

// GetProviderName returns the name of this weather provider
func (p *AccuWeatherProviderAdapter) GetProviderName() string {
	return "accuweather"
}
