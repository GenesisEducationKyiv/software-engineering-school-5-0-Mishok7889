package weather

import (
	"fmt"
	"strings"
	"time"
)

// Weather represents weather information for a specific location
type Weather struct {
	Temperature float64
	Humidity    float64
	Description string
	City        string
	Timestamp   time.Time
}

// WeatherRequest represents a request for weather information
type WeatherRequest struct {
	City string
}

// IsValid validates weather data
func (w *Weather) IsValid() error {
	if strings.TrimSpace(w.City) == "" {
		return fmt.Errorf("city cannot be empty")
	}
	if strings.TrimSpace(w.Description) == "" {
		return fmt.Errorf("description cannot be empty")
	}
	if w.Temperature < -273.15 {
		return fmt.Errorf("temperature cannot be below absolute zero")
	}
	if w.Humidity < 0 || w.Humidity > 100 {
		return fmt.Errorf("humidity must be between 0 and 100")
	}
	return nil
}

// IsValid validates weather request
func (wr *WeatherRequest) IsValid() error {
	if strings.TrimSpace(wr.City) == "" {
		return fmt.Errorf("city cannot be empty")
	}
	return nil
}

// NormalizeCity normalizes city name for consistent processing
func (wr *WeatherRequest) NormalizeCity() {
	wr.City = strings.TrimSpace(wr.City)
}

// TemperatureInFahrenheit converts temperature from Celsius to Fahrenheit
func (w *Weather) TemperatureInFahrenheit() float64 {
	return w.Temperature*9/5 + 32
}

// TemperatureInKelvin converts temperature from Celsius to Kelvin
func (w *Weather) TemperatureInKelvin() float64 {
	return w.Temperature + 273.15
}

// HumidityDescription provides a human-readable description of humidity level
func (w *Weather) HumidityDescription() string {
	switch {
	case w.Humidity < 30:
		if w.Humidity < 20 {
			return "Very dry"
		}
		return "Dry"
	case w.Humidity < 60:
		return "Comfortable"
	case w.Humidity < 80:
		return "Humid"
	default:
		return "Very humid"
	}
}

// String returns a string representation of the weather
func (w *Weather) String() string {
	return fmt.Sprintf("%s: %.1f°C, %.1f%% humidity, %s",
		w.City, w.Temperature, w.Humidity, w.Description)
}

// IsComfortable determines if the weather conditions are comfortable
func (w *Weather) IsComfortable() bool {
	// Comfortable temperature range: 18-28°C
	// Comfortable humidity range: 30-70%
	return w.Temperature >= 18 && w.Temperature <= 28 &&
		w.Humidity >= 30 && w.Humidity <= 70
}
