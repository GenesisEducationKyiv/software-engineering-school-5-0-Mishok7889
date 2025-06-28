package providers

import (
	"fmt"
	"net/http"
	"time"

	"weatherapi.app/models"
)

type AccuWeatherProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// MetricTemperature represents temperature value in metric units
type MetricTemperature struct {
	Value float64 `json:"Value"`
}

// Temperature represents temperature data with metric information
type Temperature struct {
	Metric MetricTemperature `json:"Metric"`
}

// AccuWeatherResponse represents the response structure from AccuWeather API
type AccuWeatherResponse struct {
	Temperature      Temperature `json:"Temperature"`
	RelativeHumidity float64     `json:"RelativeHumidity"`
	WeatherText      string      `json:"WeatherText"`
	Message          string      `json:"message,omitempty"`
}

func NewAccuWeatherProvider(apiKey string) WeatherProvider {
	return &AccuWeatherProvider{
		apiKey:  apiKey,
		baseURL: "http://dataservice.accuweather.com/currentconditions/v1",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *AccuWeatherProvider) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	if city == "" {
		return nil, fmt.Errorf("accuweather: city cannot be empty")
	}

	mockResponse := &AccuWeatherResponse{
		RelativeHumidity: 65,
		WeatherText:      "Partly cloudy",
	}
	mockResponse.Temperature.Metric.Value = 22.5

	return p.convertToWeatherResponse(mockResponse), nil
}

func (p *AccuWeatherProvider) convertToWeatherResponse(apiResp *AccuWeatherResponse) *models.WeatherResponse {
	return &models.WeatherResponse{
		Temperature: apiResp.Temperature.Metric.Value,
		Humidity:    apiResp.RelativeHumidity,
		Description: apiResp.WeatherText,
	}
}
