package integration

import (
	"net/http"
	"net/http/httptest"
)

func (s *IntegrationTestSuite) TestDebugWeatherConfig() {
	// Test the debug endpoint to see current configuration
	req := httptest.NewRequest("GET", "/api/debug", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	// Print the response for debugging
	s.T().Logf("Debug response: %s", w.Body.String())
}
