package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
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
		s.handleError(c, NewValidationError("city parameter is required"))
		return
	}

	s.logger.Debug("Getting weather for city", ports.F("city", city))

	request := weather.WeatherRequest{City: city}
	weatherData, err := s.weatherUseCase.GetWeather(c.Request.Context(), request)
	if err != nil {
		s.logger.Error("Weather use case error",
			ports.F("error", err),
			ports.F("city", city))
		s.handleError(c, err)
		return
	}

	response := WeatherResponse{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
	}

	s.logger.Debug("Weather result",
		ports.F("temperature", response.Temperature),
		ports.F("city", city))
	c.JSON(http.StatusOK, response)
}
