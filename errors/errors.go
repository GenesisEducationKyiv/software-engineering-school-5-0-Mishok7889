package errors

import "fmt"

// Application error types organized by category for better error handling

type ErrorType string

// Domain/Business Logic Errors - errors related to business rules and validation
const (
	ValidationError    ErrorType = "VALIDATION_ERROR"
	NotFoundError      ErrorType = "NOT_FOUND_ERROR"
	AlreadyExistsError ErrorType = "ALREADY_EXISTS_ERROR"
	TokenError         ErrorType = "TOKEN_ERROR"
)

// Infrastructure Errors - errors related to external systems and services
const (
	DatabaseError    ErrorType = "DATABASE_ERROR"
	ExternalAPIError ErrorType = "EXTERNAL_API_ERROR"
	EmailError       ErrorType = "EMAIL_ERROR"
)

// System/Configuration Errors - errors related to system setup and configuration
const (
	ConfigurationError ErrorType = "CONFIGURATION_ERROR"
)

type AppError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Message: message,
	}
}

func Wrap(errorType ErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// Domain/Business Logic Error Constructors
func NewValidationError(message string) *AppError {
	return New(ValidationError, message)
}

func NewNotFoundError(message string) *AppError {
	return New(NotFoundError, message)
}

func NewAlreadyExistsError(message string) *AppError {
	return New(AlreadyExistsError, message)
}

func NewTokenError(message string) *AppError {
	return New(TokenError, message)
}

// Infrastructure Error Constructors
func NewDatabaseError(message string, cause error) *AppError {
	return Wrap(DatabaseError, message, cause)
}

func NewExternalAPIError(message string, cause error) *AppError {
	return Wrap(ExternalAPIError, message, cause)
}

func NewEmailError(message string, cause error) *AppError {
	return Wrap(EmailError, message, cause)
}

// System/Configuration Error Constructors
func NewConfigurationError(message string, cause error) *AppError {
	return Wrap(ConfigurationError, message, cause)
}
