package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"weatherapi.app/models"
)

func (s *IntegrationTestSuite) TestGetWeather_Success() {
	req := httptest.NewRequest("GET", "/api/weather?city=London", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response models.WeatherResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)

	s.Equal(15.0, response.Temperature)
	s.Equal(76.0, response.Humidity)
	s.Equal("Partly cloudy", response.Description)
}

func (s *IntegrationTestSuite) TestGetWeather_MissingCity() {
	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("city parameter is required", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestGetWeather_EmptyCity() {
	req := httptest.NewRequest("GET", "/api/weather?city=", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("city parameter is required", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestGetWeather_CityNotFound() {
	req := httptest.NewRequest("GET", "/api/weather?city=NonExistentCity", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("city not found", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestGetWeather_ServerError() {
	req := httptest.NewRequest("GET", "/api/weather?city=servererror", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusServiceUnavailable, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("External service unavailable", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestGetWeather_DifferentCities() {
	cities := []struct {
		name        string
		temperature float64
		humidity    float64
		description string
	}{
		{"London", 15.0, 76.0, "Partly cloudy"},
		{"Paris", 18.0, 68.0, "Clear"},
		{"Berlin", 12.0, 82.0, "Overcast"},
	}

	for _, city := range cities {
		req := httptest.NewRequest("GET", "/api/weather?city="+city.name, nil)
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)

		var response models.WeatherResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		s.NoError(err)

		s.Equal(city.temperature, response.Temperature)
		s.Equal(city.humidity, response.Humidity)
		s.Equal(city.description, response.Description)
	}
}
