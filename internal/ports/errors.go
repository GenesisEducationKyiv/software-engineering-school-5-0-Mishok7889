package ports

import "fmt"

// Common error types that can be used across layers

// NotFoundError represents an error when a resource is not found
type NotFoundError interface {
	error
	IsNotFound() bool
}

// AlreadyExistsError represents an error when a resource already exists
type AlreadyExistsError interface {
	error
	IsAlreadyExists() bool
}

// NotFoundErrorImpl is a simple implementation of NotFoundError
type NotFoundErrorImpl struct {
	Message string
}

func (e *NotFoundErrorImpl) Error() string {
	return e.Message
}

func (e *NotFoundErrorImpl) IsNotFound() bool {
	return true
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(message string) error {
	return &NotFoundErrorImpl{Message: message}
}

// IsNotFoundError checks if an error is a NotFoundError
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	nfe, ok := err.(NotFoundError)
	return ok && nfe.IsNotFound()
}

// AlreadyExistsErrorImpl is a simple implementation of AlreadyExistsError
type AlreadyExistsErrorImpl struct {
	Message string
}

func (e *AlreadyExistsErrorImpl) Error() string {
	return e.Message
}

func (e *AlreadyExistsErrorImpl) IsAlreadyExists() bool {
	return true
}

// NewAlreadyExistsError creates a new AlreadyExistsError
func NewAlreadyExistsError(message string) error {
	return &AlreadyExistsErrorImpl{Message: message}
}

// IsAlreadyExistsError checks if an error is an AlreadyExistsError
func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	aee, ok := err.(AlreadyExistsError)
	return ok && aee.IsAlreadyExists()
}

// AdapterError represents errors from adapter layer
type AdapterError struct {
	Type    string
	Message string
	Cause   error
}

func (e *AdapterError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AdapterError) Unwrap() error {
	return e.Cause
}

// DatabaseError represents database-related errors
func NewDatabaseError(message string, cause error) *AdapterError {
	return &AdapterError{
		Type:    "DATABASE_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// ExternalAPIError represents external API errors
func NewExternalAPIError(message string, cause error) *AdapterError {
	return &AdapterError{
		Type:    "EXTERNAL_API_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// EmailError represents email service errors
func NewEmailError(message string, cause error) *AdapterError {
	return &AdapterError{
		Type:    "EMAIL_ERROR",
		Message: message,
		Cause:   cause,
	}
}

// ConfigurationError represents configuration errors
func NewConfigurationError(message string) *AdapterError {
	return &AdapterError{
		Type:    "CONFIGURATION_ERROR",
		Message: message,
	}
}

// ValidationError represents validation errors in adapters
func NewValidationError(message string) *AdapterError {
	return &AdapterError{
		Type:    "VALIDATION_ERROR",
		Message: message,
	}
}

// CacheError represents cache-related errors
func NewCacheError(message string, cause error) *AdapterError {
	return &AdapterError{
		Type:    "CACHE_ERROR",
		Message: message,
		Cause:   cause,
	}
}
