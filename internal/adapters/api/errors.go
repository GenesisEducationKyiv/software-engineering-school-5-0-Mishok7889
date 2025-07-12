package api

import "fmt"

// APIError represents errors specific to the API layer
type APIError struct {
	Type    string
	Message string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error for the API layer
func NewValidationError(message string) *APIError {
	return &APIError{
		Type:    "API_VALIDATION_ERROR",
		Message: message,
	}
}
