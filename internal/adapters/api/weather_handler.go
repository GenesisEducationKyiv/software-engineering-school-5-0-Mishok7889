package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/shared"
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
	// Increment API call counter
	s.metricsCollector.IncrementCounter(APIRequestsTotalMetric, map[string]string{
		EndpointLabelKey: WeatherEndpoint,
		MethodLabelKey:   GetMethod,
	})

	// Get validated city from middleware
	city := c.GetString(shared.ValidatedCityKey)

	s.logger.Debug(GettingWeatherForCityMsg, ports.F(CityField, city))

	request := weather.WeatherRequest{City: city}
	weatherData, err := s.weatherUseCase.GetWeather(c.Request.Context(), request)
	if err != nil {
		s.logger.Error(WeatherUseCaseErrorMsg,
			ports.F(ErrorField, err),
			ports.F(CityField, city))
		s.metricsCollector.IncrementCounter(APIErrorsTotalMetric, map[string]string{
			EndpointLabelKey: WeatherEndpoint,
			ErrorLabelKey:    UsecaseError,
		})
		s.handleError(c, err)
		return
	}

	response := WeatherResponse{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
	}

	s.metricsCollector.IncrementCounter(APIResponsesTotalMetric, map[string]string{
		EndpointLabelKey: WeatherEndpoint,
		StatusLabelKey:   SuccessStatus,
	})

	s.logger.Debug(WeatherResultMsg,
		ports.F(TemperatureField, response.Temperature),
		ports.F(CityField, city))
	c.JSON(http.StatusOK, response)
}
