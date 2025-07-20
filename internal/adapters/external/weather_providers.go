// Package external provides adapters for external services
// These adapters implement ports for weather providers, email services, etc.
package external

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// WeatherAPIProviderAdapter implements WeatherProvider port for WeatherAPI.com
type WeatherAPIProviderAdapter struct {
	apiKey     string
	baseURL    string
	httpClient *HTTPWeatherClient
	logger     ports.Logger
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
func NewWeatherAPIProviderAdapter(params WeatherAPIProviderParams) (ports.WeatherProvider, error) {
	if err := ValidateConstructorParams(params.APIKey); err != nil {
		return nil, fmt.Errorf("WeatherAPI provider validation failed: %w", err)
	}

	baseURL := params.BaseURL
	if baseURL == "" {
		baseURL = DefaultWeatherAPIURL
	}

	client := &http.Client{Timeout: DefaultHTTPTimeout}
	httpClient := NewHTTPWeatherClient(client, params.Logger)

	return &WeatherAPIProviderAdapter{
		apiKey:     params.APIKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     params.Logger,
	}, nil
}

// GetCurrentWeather retrieves weather data from WeatherAPI.com
func (p *WeatherAPIProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, infrastructure.NewValidationError(EmptyCityValidationMsg)
	}

	url := fmt.Sprintf("%s/current.json?key=%s&q=%s", p.baseURL, p.apiKey, city)

	req := HTTPWeatherRequest{
		URL:          url,
		ProviderName: string(WeatherAPI),
	}

	resp, err := p.httpClient.ExecuteRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var apiResp WeatherAPIResponse
	if err := p.httpClient.DecodeJSONResponse(resp, &apiResp, string(WeatherAPI)); err != nil {
		return nil, err
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
	return string(WeatherAPI)
}
