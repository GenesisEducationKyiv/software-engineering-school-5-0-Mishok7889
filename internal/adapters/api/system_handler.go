package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/ports"
)

// getMetrics handles GET /api/metrics requests
func (s *HTTPServerAdapter) getMetrics(c *gin.Context) {
	s.logger.Debug(MetricsEndpointCalledMsg)

	metrics, err := s.metricsCollector.GetMetrics(c.Request.Context())
	if err != nil {
		s.logger.Error(ErrorGettingMetricsMsg, ports.F(ErrorField, err))
		s.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// getHealth handles GET /api/health requests
func (s *HTTPServerAdapter) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{StatusField: HealthStatusOK})
}

// getDebug handles GET /api/debug requests
func (s *HTTPServerAdapter) getDebug(c *gin.Context) {
	s.logger.Debug(DebugEndpointCalledMsg)

	healthStatuses := s.systemHealthChecker.CheckAll(c.Request.Context())

	response := gin.H{}

	for component, status := range healthStatuses {
		switch component {
		case DatabaseComponent:
			response[DatabaseComponent] = gin.H{
				ConnectedField: status.Status == HealthyStatus,
			}
		case WeatherAPIComponent:
			response[WeatherAPIComponent] = gin.H{
				ConnectedField: status.Status == HealthyStatus,
			}
		case SMTPComponent:
			response[SMTPComponent] = status.Details
		case ConfigComponent:
			response[ConfigComponent] = status.Details
		}
	}

	c.JSON(http.StatusOK, response)
}
