package weather

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWeather_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		weather Weather
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidWeather",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    60.0,
				Description: "Sunny",
				City:        "London",
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "EmptyCity",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    60.0,
				Description: "Sunny",
				City:        "",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name: "WhitespaceOnlyCity",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    60.0,
				Description: "Sunny",
				City:        "   ",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name: "EmptyDescription",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    60.0,
				Description: "",
				City:        "London",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "description cannot be empty",
		},
		{
			name: "TemperatureBelowAbsoluteZero",
			weather: Weather{
				Temperature: -274.0,
				Humidity:    60.0,
				Description: "Impossible",
				City:        "Mars",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "temperature cannot be below absolute zero",
		},
		{
			name: "TemperatureAtAbsoluteZero",
			weather: Weather{
				Temperature: -273.15,
				Humidity:    60.0,
				Description: "Frozen",
				City:        "Antarctica",
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "NegativeHumidity",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    -10.0,
				Description: "Impossible",
				City:        "Desert",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "humidity must be between 0 and 100",
		},
		{
			name: "HumidityOverOneHundred",
			weather: Weather{
				Temperature: 20.0,
				Humidity:    110.0,
				Description: "Super saturated",
				City:        "Ocean",
				Timestamp:   time.Now(),
			},
			wantErr: true,
			errMsg:  "humidity must be between 0 and 100",
		},
		{
			name: "ZeroHumidity",
			weather: Weather{
				Temperature: 40.0,
				Humidity:    0.0,
				Description: "Bone dry",
				City:        "Desert",
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "OneHundredPercentHumidity",
			weather: Weather{
				Temperature: 25.0,
				Humidity:    100.0,
				Description: "Saturated",
				City:        "Rainforest",
				Timestamp:   time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.weather.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWeatherRequest_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		request WeatherRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "ValidRequest",
			request: WeatherRequest{City: "London"},
			wantErr: false,
		},
		{
			name:    "EmptyCity",
			request: WeatherRequest{City: ""},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name:    "WhitespaceOnlyCity",
			request: WeatherRequest{City: "   "},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWeatherRequest_NormalizeCity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "NoTrimming",
			input:    "London",
			expected: "London",
		},
		{
			name:     "LeadingSpaces",
			input:    "   London",
			expected: "London",
		},
		{
			name:     "TrailingSpaces",
			input:    "London   ",
			expected: "London",
		},
		{
			name:     "BothSideSpaces",
			input:    "   London   ",
			expected: "London",
		},
		{
			name:     "TabsAndSpaces",
			input:    "\t  London  \t",
			expected: "London",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := WeatherRequest{City: tt.input}
			request.NormalizeCity()
			assert.Equal(t, tt.expected, request.City)
		})
	}
}

func TestWeather_TemperatureInFahrenheit(t *testing.T) {
	tests := []struct {
		name        string
		tempCelsius float64
		expected    float64
	}{
		{
			name:        "FreezingPoint",
			tempCelsius: 0.0,
			expected:    32.0,
		},
		{
			name:        "BoilingPoint",
			tempCelsius: 100.0,
			expected:    212.0,
		},
		{
			name:        "RoomTemperature",
			tempCelsius: 20.0,
			expected:    68.0,
		},
		{
			name:        "NegativeTemperature",
			tempCelsius: -10.0,
			expected:    14.0,
		},
		{
			name:        "AbsoluteZero",
			tempCelsius: -273.15,
			expected:    -459.67,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather := Weather{Temperature: tt.tempCelsius}
			result := weather.TemperatureInFahrenheit()
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestWeather_TemperatureInKelvin(t *testing.T) {
	tests := []struct {
		name        string
		tempCelsius float64
		expected    float64
	}{
		{
			name:        "FreezingPoint",
			tempCelsius: 0.0,
			expected:    273.15,
		},
		{
			name:        "BoilingPoint",
			tempCelsius: 100.0,
			expected:    373.15,
		},
		{
			name:        "RoomTemperature",
			tempCelsius: 20.0,
			expected:    293.15,
		},
		{
			name:        "AbsoluteZero",
			tempCelsius: -273.15,
			expected:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather := Weather{Temperature: tt.tempCelsius}
			result := weather.TemperatureInKelvin()
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestWeather_HumidityDescription(t *testing.T) {
	tests := []struct {
		name     string
		humidity float64
		expected string
	}{
		{
			name:     "VeryDryLow",
			humidity: 10.0,
			expected: "Very dry",
		},
		{
			name:     "VeryDryHigh",
			humidity: 19.9,
			expected: "Very dry",
		},
		{
			name:     "DryLow",
			humidity: 20.0,
			expected: "Dry",
		},
		{
			name:     "DryHigh",
			humidity: 29.9,
			expected: "Dry",
		},
		{
			name:     "ComfortableLow",
			humidity: 30.0,
			expected: "Comfortable",
		},
		{
			name:     "ComfortableHigh",
			humidity: 59.9,
			expected: "Comfortable",
		},
		{
			name:     "HumidLow",
			humidity: 60.0,
			expected: "Humid",
		},
		{
			name:     "HumidHigh",
			humidity: 79.9,
			expected: "Humid",
		},
		{
			name:     "VeryHumidLow",
			humidity: 80.0,
			expected: "Very humid",
		},
		{
			name:     "VeryHumidHigh",
			humidity: 100.0,
			expected: "Very humid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather := Weather{Humidity: tt.humidity}
			result := weather.HumidityDescription()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWeather_IsComfortable(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		humidity    float64
		expected    bool
	}{
		{
			name:        "ComfortableConditions",
			temperature: 22.0,
			humidity:    50.0,
			expected:    true,
		},
		{
			name:        "TooHot",
			temperature: 35.0,
			humidity:    50.0,
			expected:    false,
		},
		{
			name:        "TooCold",
			temperature: 10.0,
			humidity:    50.0,
			expected:    false,
		},
		{
			name:        "TooHumid",
			temperature: 22.0,
			humidity:    85.0,
			expected:    false,
		},
		{
			name:        "TooDry",
			temperature: 22.0,
			humidity:    15.0,
			expected:    false,
		},
		{
			name:        "BoundaryTemperatureLow",
			temperature: 18.0,
			humidity:    50.0,
			expected:    true,
		},
		{
			name:        "BoundaryTemperatureHigh",
			temperature: 28.0,
			humidity:    50.0,
			expected:    true,
		},
		{
			name:        "BoundaryHumidityLow",
			temperature: 22.0,
			humidity:    30.0,
			expected:    true,
		},
		{
			name:        "BoundaryHumidityHigh",
			temperature: 22.0,
			humidity:    70.0,
			expected:    true,
		},
		{
			name:        "JustOutsideTemperatureRange",
			temperature: 17.9,
			humidity:    50.0,
			expected:    false,
		},
		{
			name:        "JustOutsideHumidityRange",
			temperature: 22.0,
			humidity:    70.1,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weather := Weather{
				Temperature: tt.temperature,
				Humidity:    tt.humidity,
			}
			result := weather.IsComfortable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWeather_String(t *testing.T) {
	tests := []struct {
		name     string
		weather  Weather
		expected string
	}{
		{
			name: "StandardWeather",
			weather: Weather{
				City:        "London",
				Temperature: 20.5,
				Humidity:    65.7,
				Description: "Partly cloudy",
			},
			expected: "London: 20.5°C, 65.7% humidity, Partly cloudy",
		},
		{
			name: "NegativeTemperature",
			weather: Weather{
				City:        "Moscow",
				Temperature: -15.2,
				Humidity:    30.0,
				Description: "Snow",
			},
			expected: "Moscow: -15.2°C, 30.0% humidity, Snow",
		},
		{
			name: "HighTemperature",
			weather: Weather{
				City:        "Dubai",
				Temperature: 45.8,
				Humidity:    90.0,
				Description: "Very hot and humid",
			},
			expected: "Dubai: 45.8°C, 90.0% humidity, Very hot and humid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.weather.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
