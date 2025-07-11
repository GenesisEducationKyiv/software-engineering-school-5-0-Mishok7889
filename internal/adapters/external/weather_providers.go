// Package external provides adapters for external services
// These adapters implement ports for weather providers, email services, etc.
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

// WeatherAPIProviderAdapter implements WeatherProvider port for WeatherAPI.com
type WeatherAPIProviderAdapter struct {
	apiKey  string
	baseURL string
	client  HTTPClient
	logger  ports.Logger
}

// WeatherAPIProviderParams holds parameters for creating WeatherAPI provider
type WeatherAPIProviderParams struct {
	APIKey  string
	BaseURL string
	Logger  ports.Logger
}

// HTTPClient interface for HTTP requests (for testing)
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// WeatherAPIResponse represents the response from WeatherAPI.com
type WeatherAPIResponse struct {
	Current struct {
		TempC     float64 `json:"temp_c"`
		Humidity  float64 `json:"humidity"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
}

// NewWeatherAPIProviderAdapter creates a new WeatherAPI provider adapter
func NewWeatherAPIProviderAdapter(params WeatherAPIProviderParams) ports.WeatherProvider {
	return &WeatherAPIProviderAdapter{
		apiKey:  params.APIKey,
		baseURL: params.BaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  params.Logger,
	}
}

// GetCurrentWeather retrieves weather data from WeatherAPI.com
func (p *WeatherAPIProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	url := fmt.Sprintf("%s/current.json?key=%s&q=%s", p.baseURL, p.apiKey, city)

	resp, err := p.client.Get(url)
	if err != nil {
		return nil, errors.NewExternalAPIError("failed to call WeatherAPI", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn("Failed to close WeatherAPI response body", ports.F("error", closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, errors.NewNotFoundError("city not found")
		}
		return nil, errors.NewExternalAPIError(fmt.Sprintf("WeatherAPI returned status %d", resp.StatusCode), nil)
	}

	var apiResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, errors.NewExternalAPIError("failed to decode WeatherAPI response", err)
	}

	return &ports.WeatherData{
		Temperature: apiResp.Current.TempC,
		Humidity:    apiResp.Current.Humidity,
		Description: apiResp.Current.Condition.Text,
		City:        city,
		Timestamp:   time.Now(),
	}, nil
}

// GetProviderName returns the name of this weather provider
func (p *WeatherAPIProviderAdapter) GetProviderName() string {
	return "weatherapi"
}
