package shared

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrCodeValidation      ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeExternalService ErrorCode = "EXTERNAL_SERVICE"
	ErrCodeInternal        ErrorCode = "INTERNAL"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Details map[string]interface{}
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// IsNotFound implements the NotFoundError interface when appropriate
func (e *DomainError) IsNotFound() bool {
	return e.Code == ErrCodeNotFound
}

// IsAlreadyExists implements the AlreadyExistsError interface when appropriate
func (e *DomainError) IsAlreadyExists() bool {
	return e.Code == ErrCodeAlreadyExists
}

func NewDomainError(code ErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

func NewDomainErrorWithCause(code ErrorCode, message string, cause error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func NewValidationError(message string) *DomainError {
	return NewDomainError(ErrCodeValidation, message)
}

func NewNotFoundError(message string) *DomainError {
	return NewDomainError(ErrCodeNotFound, message)
}

func NewAlreadyExistsError(message string) *DomainError {
	return NewDomainError(ErrCodeAlreadyExists, message)
}

func NewExternalServiceError(message string) *DomainError {
	return NewDomainError(ErrCodeExternalService, message)
}

func NewExternalServiceErrorWithCause(message string, cause error) *DomainError {
	return NewDomainErrorWithCause(ErrCodeExternalService, message, cause)
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrCodeValidation
	}

	return false
}

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it implements the NotFoundError interface
	if nfe, ok := err.(interface{ IsNotFound() bool }); ok {
		return nfe.IsNotFound()
	}

	// Check if it's a DomainError with NotFound code
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrCodeNotFound
	}

	return false
}

func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it implements the AlreadyExistsError interface
	if aee, ok := err.(interface{ IsAlreadyExists() bool }); ok {
		return aee.IsAlreadyExists()
	}

	// Check if it's a DomainError with AlreadyExists code
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code == ErrCodeAlreadyExists
	}

	return false
}
