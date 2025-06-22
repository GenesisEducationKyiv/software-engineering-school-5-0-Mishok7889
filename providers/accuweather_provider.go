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

type AccuWeatherResponse struct {
	Temperature struct {
		Metric struct {
			Value float64 `json:"Value"`
		} `json:"Metric"`
	} `json:"Temperature"`
	RelativeHumidity float64 `json:"RelativeHumidity"`
	WeatherText      string  `json:"WeatherText"`
	Message          string  `json:"message,omitempty"`
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
