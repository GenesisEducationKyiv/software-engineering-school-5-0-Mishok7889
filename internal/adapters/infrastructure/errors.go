package infrastructure

import "fmt"

// InfrastructureError represents errors from infrastructure layer
type InfrastructureError struct {
	Type    string
	Message string
	Cause   error
}

func (e *InfrastructureError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *InfrastructureError) Unwrap() error {
	return e.Cause
}

// DatabaseError represents database-related errors
func NewDatabaseError(message string, cause error) *InfrastructureError {
	return &InfrastructureError{
		Type:    "DATABASE_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// ExternalAPIError represents external API errors
func NewExternalAPIError(message string, cause error) *InfrastructureError {
	return &InfrastructureError{
		Type:    "EXTERNAL_API_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// EmailError represents email service errors
func NewEmailError(message string, cause error) *InfrastructureError {
	return &InfrastructureError{
		Type:    "EMAIL_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// ConfigurationError represents configuration errors
func NewConfigurationError(message string) *InfrastructureError {
	return &InfrastructureError{
		Type:    "CONFIGURATION_ERROR",
		Message: message,
	}
}

// ValidationError represents validation errors in adapters
func NewValidationError(message string) *InfrastructureError {
	return &InfrastructureError{
		Type:    "VALIDATION_ERROR",
		Message: message,
	}
}

// CacheError represents cache-related errors
func NewCacheError(message string, cause error) *InfrastructureError {
	return &InfrastructureError{
		Type:    "CACHE_ERROR",
		Message: message,
		Cause:   cause,
	}
}
