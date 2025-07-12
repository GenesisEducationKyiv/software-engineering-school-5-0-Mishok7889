package api

import (
	"errors"
	"net/http"

	"log/slog"

	"github.com/gin-gonic/gin"
	errorspkg "weatherapi.app/pkg/errors"
)

// ErrorResponse represents an error message structure for API responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleError handles different types of application errors
func (s *HTTPServerAdapter) handleError(c *gin.Context, err error) {
	var appErr *errorspkg.AppError
	var statusCode int
	var message string

	if !errors.As(err, &appErr) {
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
		c.JSON(statusCode, ErrorResponse{Error: message})
		return
	}

	switch appErr.Type {
	case errorspkg.ValidationError:
		statusCode = http.StatusBadRequest
		message = appErr.Message
	case errorspkg.NotFoundError:
		statusCode = http.StatusNotFound
		message = appErr.Message
	case errorspkg.AlreadyExistsError:
		statusCode = http.StatusConflict
		message = appErr.Message
	case errorspkg.ExternalAPIError:
		statusCode = http.StatusServiceUnavailable
		message = "External service unavailable"
	case errorspkg.DatabaseError:
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	case errorspkg.EmailError:
		statusCode = http.StatusServiceUnavailable
		message = "Unable to send email"
	case errorspkg.TokenError:
		statusCode = http.StatusBadRequest
		message = appErr.Message
	default:
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	}

	c.JSON(statusCode, ErrorResponse{Error: message})
}

// getMetrics handles GET /api/metrics requests
func (s *HTTPServerAdapter) getMetrics(c *gin.Context) {
	slog.Debug("Metrics endpoint called")

	metrics, err := s.metricsCollector.GetMetrics(c.Request.Context())
	if err != nil {
		slog.Error("Error getting metrics", "error", err)
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
	slog.Debug("Debug endpoint called")

	// Perform health checks on all system components
	healthStatuses := s.systemHealthChecker.CheckAll(c.Request.Context())

	// Transform health statuses to the expected format
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
