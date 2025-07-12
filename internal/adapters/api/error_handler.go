package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

// ErrorResponse represents an error message structure for API responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleError handles different types of application errors
func (s *HTTPServerAdapter) handleError(c *gin.Context, err error) {
	var statusCode int
	var message string

	// Handle API-specific errors
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Type {
		case "API_VALIDATION_ERROR":
			statusCode = http.StatusBadRequest
			message = apiErr.Message
		default:
			statusCode = http.StatusInternalServerError
			message = "Internal server error"
		}
		c.JSON(statusCode, ErrorResponse{Error: message})
		return
	}

	// Handle domain errors
	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case shared.ErrCodeValidation:
			statusCode = http.StatusBadRequest
			message = domainErr.Message
		case shared.ErrCodeNotFound:
			statusCode = http.StatusNotFound
			message = domainErr.Message
		case shared.ErrCodeAlreadyExists:
			statusCode = http.StatusConflict
			message = domainErr.Message
		case shared.ErrCodeUnauthorized:
			statusCode = http.StatusUnauthorized
			message = domainErr.Message
		case shared.ErrCodeExternalService:
			statusCode = http.StatusServiceUnavailable
			message = "External service unavailable"
		case shared.ErrCodeInternal:
			statusCode = http.StatusInternalServerError
			message = "Internal server error"
		default:
			statusCode = http.StatusInternalServerError
			message = "Internal server error"
		}
		c.JSON(statusCode, ErrorResponse{Error: message})
		return
	}

	// Handle port-level errors
	if ports.IsNotFoundError(err) {
		statusCode = http.StatusNotFound
		message = err.Error()
		c.JSON(statusCode, ErrorResponse{Error: message})
		return
	}

	if ports.IsAlreadyExistsError(err) {
		statusCode = http.StatusConflict
		message = err.Error()
		c.JSON(statusCode, ErrorResponse{Error: message})
		return
	}

	// Default error handling
	statusCode = http.StatusInternalServerError
	message = "Internal server error"
	c.JSON(statusCode, ErrorResponse{Error: message})
}

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
