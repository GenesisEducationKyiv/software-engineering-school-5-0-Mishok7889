package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func (s *IntegrationTestSuite) TestDebugEndpoint() {
	req := httptest.NewRequest("GET", "/api/debug", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)

	s.Contains(response, "database")
	s.Contains(response, "weatherAPI")
	s.Contains(response, "smtp")
	s.Contains(response, "config")

	database := response["database"].(map[string]interface{})
	s.Equal(true, database["connected"])
	s.Equal(float64(0), database["subscriptionCount"])

	weatherAPI := response["weatherAPI"].(map[string]interface{})
	s.Equal(true, weatherAPI["connected"])

	config := response["config"].(map[string]interface{})
	s.Equal("http://localhost:8080", config["appBaseURL"])
}

func (s *IntegrationTestSuite) TestDebugEndpoint_WithSubscriptions() {
	s.CreateTestSubscription("debug1@example.com", "London", "daily", true)
	s.CreateTestSubscription("debug2@example.com", "Paris", "hourly", false)

	req := httptest.NewRequest("GET", "/api/debug", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)

	database := response["database"].(map[string]interface{})
	s.Equal(true, database["connected"])
	s.Equal(float64(2), database["subscriptionCount"])
}
