package services

import (
	"context"

	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
)

// WeatherServiceAdapter adapts the weather use case to the WeatherService port
type WeatherServiceAdapter struct {
	weatherUseCase weather.Service
}

// NewWeatherServiceAdapter creates a new weather service adapter
func NewWeatherServiceAdapter(weatherUseCase weather.Service) ports.WeatherService {
	return &WeatherServiceAdapter{
		weatherUseCase: weatherUseCase,
	}
}

// GetWeather adapts the use case weather data to service data
func (a *WeatherServiceAdapter) GetWeather(ctx context.Context, city string) (*ports.WeatherServiceData, error) {
	request := weather.WeatherRequest{City: city}
	weatherData, err := a.weatherUseCase.GetWeather(ctx, request)
	if err != nil {
		return nil, err
	}

	return &ports.WeatherServiceData{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
		Timestamp:   weatherData.Timestamp,
	}, nil
}

// GetProviderInfo delegates to the use case
func (a *WeatherServiceAdapter) GetProviderInfo(ctx context.Context) ports.ProviderInfo {
	return a.weatherUseCase.GetProviderInfo(ctx)
}
