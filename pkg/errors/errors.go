package errors

import "fmt"

// Application error types organized by category for better error handling

type ErrorType int

// Domain/Business Logic Errors - errors related to business rules and validation
const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeValidation
	ErrorTypeNotFound
	ErrorTypeAlreadyExists
	ErrorTypeToken

	// Infrastructure Errors - errors related to external systems and services
	ErrorTypeDatabase
	ErrorTypeExternalAPI
	ErrorTypeEmail

	// System/Configuration Errors - errors related to system setup and configuration
	ErrorTypeConfiguration
)

// String returns the string representation of error type
func (e ErrorType) String() string {
	switch e {
	case ErrorTypeValidation:
		return "VALIDATION_ERROR"
	case ErrorTypeNotFound:
		return "NOT_FOUND_ERROR"
	case ErrorTypeAlreadyExists:
		return "ALREADY_EXISTS_ERROR"
	case ErrorTypeToken:
		return "TOKEN_ERROR"
	case ErrorTypeDatabase:
		return "DATABASE_ERROR"
	case ErrorTypeExternalAPI:
		return "EXTERNAL_API_ERROR"
	case ErrorTypeEmail:
		return "EMAIL_ERROR"
	case ErrorTypeConfiguration:
		return "CONFIGURATION_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}

// Legacy constants for backward compatibility
const (
	ValidationError    = ErrorTypeValidation
	NotFoundError      = ErrorTypeNotFound
	AlreadyExistsError = ErrorTypeAlreadyExists
	TokenError         = ErrorTypeToken
	DatabaseError      = ErrorTypeDatabase
	ExternalAPIError   = ErrorTypeExternalAPI
	EmailError         = ErrorTypeEmail
	ConfigurationError = ErrorTypeConfiguration
)

type AppError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type.String(), e.Message)
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

// Helper functions for error type checking
func IsNotFoundError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == NotFoundError
	}
	return false
}

func IsTokenError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == TokenError
	}
	return false
}

func IsAlreadyExistsError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == AlreadyExistsError
	}
	return false
}

func IsValidationError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ValidationError
	}
	return false
}

func IsDatabaseError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == DatabaseError
	}
	return false
}

func IsEmailError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == EmailError
	}
	return false
}

func IsConfigurationError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == ConfigurationError
	}
	return false
}
