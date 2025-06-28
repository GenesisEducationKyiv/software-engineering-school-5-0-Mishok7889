package providers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"weatherapi.app/models"
)

type OpenWeatherMapProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type OpenWeatherMapResponse struct {
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity float64 `json:"humidity"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
	Message string `json:"message,omitempty"`
}

func NewOpenWeatherMapProvider(apiKey string) WeatherProvider {
	return &OpenWeatherMapProvider{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org/data/2.5/weather",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *OpenWeatherMapProvider) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	url := fmt.Sprintf("%s?q=%s&appid=%s&units=metric", p.baseURL, city, p.apiKey)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("openweathermap API request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleHTTPError(resp.StatusCode)
	}

	var apiResponse OpenWeatherMapResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("decode openweathermap response: %w", err)
	}

	return p.convertToWeatherResponse(&apiResponse), nil
}

func (p *OpenWeatherMapProvider) handleHTTPError(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("openweathermap: invalid API key")
	case http.StatusNotFound:
		return fmt.Errorf("openweathermap: city not found")
	case http.StatusTooManyRequests:
		return fmt.Errorf("openweathermap: rate limit exceeded")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("openweathermap: service unavailable")
	default:
		return fmt.Errorf("openweathermap: HTTP %d error", statusCode)
	}
}

func (p *OpenWeatherMapProvider) convertToWeatherResponse(apiResp *OpenWeatherMapResponse) *models.WeatherResponse {
	description := "No description"
	if len(apiResp.Weather) > 0 {
		description = apiResp.Weather[0].Description
	}

	return &models.WeatherResponse{
		Temperature: apiResp.Main.Temp,
		Humidity:    apiResp.Main.Humidity,
		Description: description,
	}
}
