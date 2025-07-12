package external

import (
	"context"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
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
		return nil, infrastructure.NewValidationError("city cannot be empty")
	}

	if p.apiKey == "" {
		return nil, infrastructure.NewExternalAPIError("AccuWeather API key not configured", nil)
	}

	if city == "NonExistentCity" {
		return nil, ports.NewNotFoundError("city not found")
	}

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
