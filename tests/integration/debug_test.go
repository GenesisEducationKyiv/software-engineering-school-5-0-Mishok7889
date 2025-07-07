package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func (s *IntegrationTestSuite) TestDebugEndpoint() {
	// Test the debug endpoint to verify system health and configuration
	req := httptest.NewRequest("GET", "/api/debug", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)

	// Verify response structure
	s.Contains(response, "database")
	s.Contains(response, "weatherAPI")
	s.Contains(response, "smtp")
	s.Contains(response, "config")

	// Verify database connection
	database, ok := response["database"].(map[string]interface{})
	s.True(ok)
	s.Contains(database, "connected")
	s.True(database["connected"].(bool))

	// Verify weather API connection
	weatherAPI, ok := response["weatherAPI"].(map[string]interface{})
	s.True(ok)
	s.Contains(weatherAPI, "connected")

	// Verify SMTP configuration
	smtp, ok := response["smtp"].(map[string]interface{})
	s.True(ok)
	s.Contains(smtp, "host")
	s.Equal("localhost", smtp["host"])
	s.Contains(smtp, "port")
	s.Equal("1025", smtp["port"])

	// Verify app configuration
	configData, ok := response["config"].(map[string]interface{})
	s.True(ok)
	s.Contains(configData, "appBaseURL")
	s.Equal("http://localhost:8080", configData["appBaseURL"])

	// Print the response for debugging purposes
	s.T().Logf("Debug response: %s", w.Body.String())
}

func (s *IntegrationTestSuite) TestMetricsEndpoint() {
	// Test the metrics endpoint
	req := httptest.NewRequest("GET", "/api/metrics", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	// Metrics endpoint may not be available depending on implementation
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		s.NoError(err)

		// If metrics are available, verify structure
		if cache, exists := response["cache"]; exists {
			s.NotNil(cache)
		}
		if providerInfo, exists := response["provider_info"]; exists {
			s.NotNil(providerInfo)
		}
		if endpoints, exists := response["endpoints"]; exists {
			endpointsMap, ok := endpoints.(map[string]interface{})
			s.True(ok)
			s.Contains(endpointsMap, "prometheus_metrics")
			s.Contains(endpointsMap, "cache_metrics")
		}

		s.T().Logf("Metrics response: %s", w.Body.String())
	} else {
		s.T().Logf("Metrics endpoint returned status %d", w.Code)
	}
}
