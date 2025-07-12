package api

import (
	"net/http"

	"log/slog"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/pkg/errors"
)

// WeatherResponse represents the HTTP response for weather data
type WeatherResponse struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Description string  `json:"description"`
	City        string  `json:"city"`
}

// getWeather handles GET /api/weather requests
func (s *HTTPServerAdapter) getWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		s.handleError(c, errors.NewValidationError("city parameter is required"))
		return
	}

	slog.Debug("Getting weather for city", "city", city)

	request := weather.WeatherRequest{City: city}
	weatherData, err := s.weatherUseCase.GetWeather(c.Request.Context(), request)
	if err != nil {
		slog.Error("Weather use case error", "error", err, "city", city)
		s.handleError(c, err)
		return
	}

	response := WeatherResponse{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
	}

	slog.Debug("Weather result", "weather", response, "city", city)
	c.JSON(http.StatusOK, response)
}
