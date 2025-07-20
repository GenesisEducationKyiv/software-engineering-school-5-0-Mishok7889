package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/ports"
)

// getMetrics handles GET /api/metrics requests
func (s *HTTPServerAdapter) getMetrics(c *gin.Context) {
	s.logger.Debug("Metrics endpoint called")

	metrics, err := s.metricsCollector.GetMetrics(c.Request.Context())
	if err != nil {
		s.logger.Error("Error getting metrics", ports.F("error", err))
		s.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// getHealth handles GET /api/health requests
func (s *HTTPServerAdapter) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// getDebug handles GET /api/debug requests
func (s *HTTPServerAdapter) getDebug(c *gin.Context) {
	s.logger.Debug("Debug endpoint called")

	healthStatuses := s.systemHealthChecker.CheckAll(c.Request.Context())

	response := gin.H{}

	for component, status := range healthStatuses {
		switch component {
		case "database":
			response["database"] = gin.H{
				"connected": status.Status == "healthy",
			}
		case "weatherAPI":
			response["weatherAPI"] = gin.H{
				"connected": status.Status == "healthy",
			}
		case "smtp":
			response["smtp"] = status.Details
		case "config":
			response["config"] = status.Details
		}
	}

	c.JSON(http.StatusOK, response)
}
