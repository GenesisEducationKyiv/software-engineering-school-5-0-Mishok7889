package external

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// OpenWeatherMapProviderAdapter implements WeatherProvider port for OpenWeatherMap
type OpenWeatherMapProviderAdapter struct {
	apiKey     string
	baseURL    string
	httpClient *HTTPWeatherClient
	logger     ports.Logger
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
func NewOpenWeatherMapProviderAdapter(params OpenWeatherMapProviderParams) (ports.WeatherProvider, error) {
	if err := ValidateConstructorParams(params.APIKey); err != nil {
		return nil, fmt.Errorf("OpenWeatherMap provider validation failed: %w", err)
	}

	baseURL := params.BaseURL
	if baseURL == "" {
		baseURL = DefaultOpenWeatherMapURL
	}

	client := &http.Client{Timeout: DefaultHTTPTimeout}
	httpClient := NewHTTPWeatherClient(client, params.Logger)

	return &OpenWeatherMapProviderAdapter{
		apiKey:     params.APIKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     params.Logger,
	}, nil
}

// GetCurrentWeather retrieves weather data from OpenWeatherMap
func (p *OpenWeatherMapProviderAdapter) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if city == "" {
		return nil, infrastructure.NewValidationError(EmptyCityValidationMsg)
	}

	url := fmt.Sprintf("%s/weather?q=%s&appid=%s&units=metric", p.baseURL, city, p.apiKey)

	req := HTTPWeatherRequest{
		URL:          url,
		ProviderName: string(OpenWeatherMap),
	}

	resp, err := p.httpClient.ExecuteRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var apiResp OpenWeatherMapResponse
	if err := p.httpClient.DecodeJSONResponse(resp, &apiResp, string(OpenWeatherMap)); err != nil {
		return nil, err
	}

	description := DefaultWeatherDescription
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
	return string(OpenWeatherMap)
}
