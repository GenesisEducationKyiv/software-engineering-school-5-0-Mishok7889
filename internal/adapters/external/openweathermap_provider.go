package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// OpenWeatherMapProviderAdapter implements WeatherProvider port for OpenWeatherMap
type OpenWeatherMapProviderAdapter struct {
	apiKey  string
	baseURL string
	client  HTTPClient
	logger  ports.Logger
}

// OpenWeatherMapProviderParams holds parameters for creating OpenWeatherMap provider
type OpenWeatherMapProviderParams struct {
	APIKey  string
	BaseURL string
	Logger  ports.Logger
}

// OpenWeatherMapResponse represents the response from OpenWeatherMap API
type OpenWeatherMapResponse struct {
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity float64 `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

// NewOpenWeatherMapProviderAdapter creates a new OpenWeatherMap provider adapter
func NewOpenWeatherMapProviderAdapter(params OpenWeatherMapProviderParams) ports.WeatherProvider {
	baseURL := params.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openweathermap.org/data/2.5"
	}

	return &OpenWeatherMapProviderAdapter{
		apiKey:  params.APIKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  params.Logger,
	}
}

// GetCurrentWeather retrieves weather data from OpenWeatherMap
func (p *OpenWeatherMapProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	url := fmt.Sprintf("%s/weather?q=%s&appid=%s&units=metric", p.baseURL, city, p.apiKey)

	resp, err := p.client.Get(url)
	if err != nil {
		return nil, errors.NewExternalAPIError("failed to call OpenWeatherMap", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn("Failed to close OpenWeatherMap response body", ports.F("error", closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewExternalAPIError(fmt.Sprintf("OpenWeatherMap returned status %d", resp.StatusCode), nil)
	}

	var apiResp OpenWeatherMapResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.NewExternalAPIError("failed to decode OpenWeatherMap response", err)
	}

	description := "Clear"
	if len(apiResp.Weather) > 0 {
		description = apiResp.Weather[0].Description
	}

	return &ports.WeatherData{
		Temperature: apiResp.Main.Temp,
		Humidity:    apiResp.Main.Humidity,
		Description: description,
		City:        city,
		Timestamp:   time.Now(),
	}, nil
}

// GetProviderName returns the name of this weather provider
func (p *OpenWeatherMapProviderAdapter) GetProviderName() string {
	return "openweathermap"
}
