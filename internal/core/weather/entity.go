package weather

import (
	"fmt"
	"strings"
	"time"
)

const (
	// Temperature constants
	absoluteZeroCelsius  = -273.15   // Absolute zero in Celsius
	kelvinOffset         = 273.15    // Offset to convert Celsius to Kelvin
	fahrenheitMultiplier = 9.0 / 5.0 // Multiplier for Celsius to Fahrenheit
	fahrenheitOffset     = 32.0      // Offset for Celsius to Fahrenheit

	// Comfort range constants
	comfortTempMin     = 18.0 // Minimum comfortable temperature in Celsius
	comfortTempMax     = 28.0 // Maximum comfortable temperature in Celsius
	comfortHumidityMin = 30.0 // Minimum comfortable humidity percentage
	comfortHumidityMax = 70.0 // Maximum comfortable humidity percentage

	// Humidity thresholds for descriptions
	humidityVeryDry     = 20.0 // Below this is very dry
	humidityDry         = 30.0 // Below this is dry
	humidityComfortable = 60.0 // Below this is comfortable
	humidityHumid       = 80.0 // Below this is humid, above is very humid

	// Validation constants
	minHumidity = 0.0   // Minimum valid humidity percentage
	maxHumidity = 100.0 // Maximum valid humidity percentage
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
	if w.Temperature < absoluteZeroCelsius {
		return fmt.Errorf("temperature cannot be below absolute zero")
	}
	if w.Humidity < minHumidity || w.Humidity > maxHumidity {
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
	return w.Temperature*fahrenheitMultiplier + fahrenheitOffset
}

// TemperatureInKelvin converts temperature from Celsius to Kelvin
func (w *Weather) TemperatureInKelvin() float64 {
	return w.Temperature + kelvinOffset
}

// HumidityDescription provides a human-readable description of humidity level
func (w *Weather) HumidityDescription() string {
	switch {
	case w.Humidity < humidityDry:
		if w.Humidity < humidityVeryDry {
			return "Very dry"
		}
		return "Dry"
	case w.Humidity < humidityComfortable:
		return "Comfortable"
	case w.Humidity < humidityHumid:
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
	return w.Temperature >= comfortTempMin && w.Temperature <= comfortTempMax &&
		w.Humidity >= comfortHumidityMin && w.Humidity <= comfortHumidityMax
}
