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
	statusCode, message := s.mapError(err)
	c.JSON(statusCode, ErrorResponse{Error: message})
}

// mapError determines the appropriate HTTP status code and message for an error
func (s *HTTPServerAdapter) mapError(err error) (int, string) {
	if statusCode, message := s.mapAPIError(err); statusCode != 0 {
		return statusCode, message
	}

	if statusCode, message := s.mapDomainError(err); statusCode != 0 {
		return statusCode, message
	}

	if statusCode, message := s.mapPortError(err); statusCode != 0 {
		return statusCode, message
	}

	return http.StatusInternalServerError, InternalServerErrorMsg
}

// mapAPIError maps API-specific errors to HTTP responses
func (s *HTTPServerAdapter) mapAPIError(err error) (int, string) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return 0, ""
	}

	switch apiErr.Type {
	case APIValidationErrorType:
		return http.StatusBadRequest, apiErr.Message
	default:
		return http.StatusInternalServerError, InternalServerErrorMsg
	}
}

// mapDomainError maps domain errors to HTTP responses
func (s *HTTPServerAdapter) mapDomainError(err error) (int, string) {
	var domainErr *shared.DomainError
	if !errors.As(err, &domainErr) {
		return 0, ""
	}

	switch domainErr.Code {
	case shared.ErrCodeValidation:
		return http.StatusBadRequest, domainErr.Message
	case shared.ErrCodeNotFound:
		return http.StatusNotFound, domainErr.Message
	case shared.ErrCodeAlreadyExists:
		return http.StatusConflict, domainErr.Message
	case shared.ErrCodeUnauthorized:
		return http.StatusUnauthorized, domainErr.Message
	case shared.ErrCodeExternalService:
		return http.StatusServiceUnavailable, ExternalServiceErrorMsg
	case shared.ErrCodeInternal:
		return http.StatusInternalServerError, InternalServerErrorMsg
	default:
		return http.StatusInternalServerError, InternalServerErrorMsg
	}
}

// mapPortError maps port-level errors to HTTP responses
func (s *HTTPServerAdapter) mapPortError(err error) (int, string) {
	if ports.IsNotFoundError(err) {
		return http.StatusNotFound, err.Error()
	}

	if ports.IsAlreadyExistsError(err) {
		return http.StatusConflict, err.Error()
	}

	return 0, ""
}
