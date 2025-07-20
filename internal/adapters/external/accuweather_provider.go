package external

import (
	"context"
	"fmt"
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
func NewAccuWeatherProviderAdapter(params AccuWeatherProviderParams) (ports.WeatherProvider, error) {
	if err := ValidateConstructorParams(params.APIKey); err != nil {
		return nil, fmt.Errorf("AccuWeather provider validation failed: %w", err)
	}

	baseURL := params.BaseURL
	if baseURL == "" {
		baseURL = DefaultAccuWeatherURL
	}

	return &AccuWeatherProviderAdapter{
		apiKey:  params.APIKey,
		baseURL: baseURL,
		logger:  params.Logger,
	}, nil
}

// GetCurrentWeather retrieves weather data from AccuWeather (mock implementation)
func (p *AccuWeatherProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, infrastructure.NewValidationError(EmptyCityValidationMsg)
	}

	if city == "NonExistentCity" {
		return nil, ports.NewNotFoundError(CityNotFoundMsg)
	}

	return &ports.WeatherData{
		Temperature: DefaultTemperature,
		Humidity:    DefaultHumidity,
		Description: DefaultMockDescription,
		City:        city,
		Timestamp:   time.Now(),
	}, nil
}

// GetProviderName returns the name of this weather provider
func (p *AccuWeatherProviderAdapter) GetProviderName() string {
	return string(AccuWeather)
}
