package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockMetricsCollector for testing
type mockMetricsCollector struct {
	mock.Mock
}

func (m *mockMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	m.Called(name, labels)
}

func TestMetricsMiddleware_CollectRequestMetrics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockCollector := &mockMetricsCollector{}
	middleware := NewMetricsMiddleware(mockCollector)

	// Expect request counter increment
	mockCollector.On("IncrementCounter", APIRequestsTotalMetric, map[string]string{
		EndpointLabelKey: "weather",
		MethodLabelKey:   "GET",
	}).Once()

	// Expect response counter increment for successful response
	mockCollector.On("IncrementCounter", APIResponsesTotalMetric, map[string]string{
		EndpointLabelKey: "weather",
		StatusLabelKey:   SuccessStatus,
	}).Once()

	router := gin.New()
	router.Use(middleware.CollectRequestMetrics())
	router.GET("/api/weather", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockCollector.AssertExpectations(t)
}

func TestMetricsMiddleware_CollectRequestMetrics_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockCollector := &mockMetricsCollector{}
	middleware := NewMetricsMiddleware(mockCollector)

	// Expect only request counter increment for error response
	mockCollector.On("IncrementCounter", APIRequestsTotalMetric, map[string]string{
		EndpointLabelKey: "weather",
		MethodLabelKey:   "GET",
	}).Once()

	// No response counter increment expected for error responses

	router := gin.New()
	router.Use(middleware.CollectRequestMetrics())
	router.GET("/api/weather", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	req := httptest.NewRequest("GET", "/api/weather", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockCollector.AssertExpectations(t)
}

func TestExtractEndpointFromPath(t *testing.T) {
	tests := []struct {
		fullPath string
		expected string
	}{
		{"/api/weather", "weather"},
		{"/api/subscribe", "subscribe"},
		{"/api/confirm/:token", "confirm"},
		{"/api/unsubscribe/:token", "unsubscribe"},
		{"/api/health", "health"},
		{"/api/debug", "debug"},
		{"/api/metrics", "metrics"},
		{"/unknown/path", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.fullPath, func(t *testing.T) {
			result := extractEndpointFromPath(tt.fullPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
