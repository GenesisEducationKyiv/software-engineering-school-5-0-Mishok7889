package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"weatherapi.app/config"
	"weatherapi.app/errors"
	"weatherapi.app/models"
)

// WeatherAPIProvider implements WeatherProvider for WeatherAPI.com
type WeatherAPIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewWeatherAPIProvider creates a new WeatherAPI.com provider
func NewWeatherAPIProvider(config *config.WeatherConfig) *WeatherAPIProvider {
	return &WeatherAPIProvider{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// GetCurrentWeather retrieves weather data from WeatherAPI.com
func (p *WeatherAPIProvider) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	url := fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no", p.baseURL, p.apiKey, city)

	resp, err := p.client.Get(url)
	if err != nil {
		return nil, errors.NewExternalAPIError("failed to get weather data", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Ignore close error as it's not critical for the main operation
			_ = closeErr
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.NewNotFoundError("city not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewExternalAPIError(fmt.Sprintf("weather API returned status code %d", resp.StatusCode), nil)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.NewExternalAPIError("failed to decode weather data", err)
	}

	current, ok := result["current"].(map[string]interface{})
	if !ok {
		return nil, errors.NewExternalAPIError("invalid weather data format: missing current field", nil)
	}

	weatherCondition, ok := current["condition"].(map[string]interface{})
	if !ok {
		return nil, errors.NewExternalAPIError("invalid weather data format: missing condition field", nil)
	}

	temperature, ok := current["temp_c"].(float64)
	if !ok {
		return nil, errors.NewExternalAPIError("invalid weather data format: missing temperature", nil)
	}

	humidity, ok := current["humidity"].(float64)
	if !ok {
		return nil, errors.NewExternalAPIError("invalid weather data format: missing humidity", nil)
	}

	description, ok := weatherCondition["text"].(string)
	if !ok {
		return nil, errors.NewExternalAPIError("invalid weather data format: missing description", nil)
	}

	return &models.WeatherResponse{
		Temperature: temperature,
		Humidity:    humidity,
		Description: description,
	}, nil
}
