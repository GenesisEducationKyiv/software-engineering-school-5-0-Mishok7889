package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Error constants for validation
const (
	ErrTokenRequired = "token parameter is required"
	ErrCityRequired  = "city parameter is required"
)

// ErrorResponse represents an error message structure for API responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidationMiddleware provides parameter validation for API endpoints
type ValidationMiddleware struct{}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{}
}

// ValidateTokenParam validates that a token URL parameter exists and is not empty
func (v *ValidationMiddleware) ValidateTokenParam() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrTokenRequired,
			})
			c.Abort()
			return
		}

		// Store validated token in context for handler use
		c.Set("validated_token", token)
		c.Next()
	}
}

// ValidateCityQuery validates that a city query parameter exists and is not empty
func (v *ValidationMiddleware) ValidateCityQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		city := strings.TrimSpace(c.Query("city"))
		if city == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrCityRequired,
			})
			c.Abort()
			return
		}

		// Store validated city in context for handler use
		c.Set("validated_city", city)
		c.Next()
	}
}

// ValidateRequiredParam validates that a specific parameter exists and is not empty
func (v *ValidationMiddleware) ValidateRequiredParam(paramName, paramType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var value string

		switch paramType {
		case "param":
			value = strings.TrimSpace(c.Param(paramName))
		case "query":
			value = strings.TrimSpace(c.Query(paramName))
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid parameter type",
			})
			c.Abort()
			return
		}

		if value == "" {
			errorMsg := paramName + " parameter is required"
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: errorMsg,
			})
			c.Abort()
			return
		}

		// Store validated parameter in context
		contextKey := "validated_" + paramName
		c.Set(contextKey, value)
		c.Next()
	}
}
