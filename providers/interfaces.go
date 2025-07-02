package providers

import (
	"weatherapi.app/models"
)

// WeatherProvider defines the interface for weather data providers
type WeatherProvider interface {
	GetCurrentWeather(city string) (*models.WeatherResponse, error)
}

// EmailProvider defines the interface for email providers
type EmailProvider interface {
	SendEmail(to, subject, body string, isHTML bool) error
}
