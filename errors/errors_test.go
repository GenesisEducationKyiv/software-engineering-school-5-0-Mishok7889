package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	t.Run("ErrorWithoutCause", func(t *testing.T) {
		err := New(ValidationError, "test validation error")

		expected := "VALIDATION_ERROR: test validation error"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("ErrorWithCause", func(t *testing.T) {
		cause := fmt.Errorf("original error")
		err := Wrap(DatabaseError, "database operation failed", cause)

		expected := "DATABASE_ERROR: database operation failed (caused by: original error)"
		assert.Equal(t, expected, err.Error())
	})
}

func TestAppError_Unwrap(t *testing.T) {
	t.Run("ErrorWithCause", func(t *testing.T) {
		cause := fmt.Errorf("original error")
		err := Wrap(ExternalAPIError, "API call failed", cause)

		unwrapped := err.Unwrap()
		assert.Equal(t, cause, unwrapped)
	})

	t.Run("ErrorWithoutCause", func(t *testing.T) {
		err := New(NotFoundError, "resource not found")

		unwrapped := err.Unwrap()
		assert.Nil(t, unwrapped)
	})
}

func TestNew(t *testing.T) {
	err := New(TokenError, "invalid token")

	assert.Equal(t, TokenError, err.Type)
	assert.Equal(t, "invalid token", err.Message)
	assert.Nil(t, err.Cause)
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap(ConfigurationError, "config validation failed", cause)

	assert.Equal(t, ConfigurationError, err.Type)
	assert.Equal(t, "config validation failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestSpecificErrorConstructors(t *testing.T) {
	t.Run("NewValidationError", func(t *testing.T) {
		err := NewValidationError("field is required")

		assert.Equal(t, ValidationError, err.Type)
		assert.Equal(t, "field is required", err.Message)
		assert.Nil(t, err.Cause)
	})

	t.Run("NewNotFoundError", func(t *testing.T) {
		err := NewNotFoundError("resource not found")

		assert.Equal(t, NotFoundError, err.Type)
		assert.Equal(t, "resource not found", err.Message)
		assert.Nil(t, err.Cause)
	})

	t.Run("NewAlreadyExistsError", func(t *testing.T) {
		err := NewAlreadyExistsError("resource already exists")

		assert.Equal(t, AlreadyExistsError, err.Type)
		assert.Equal(t, "resource already exists", err.Message)
		assert.Nil(t, err.Cause)
	})

	t.Run("NewExternalAPIError", func(t *testing.T) {
		cause := fmt.Errorf("network timeout")
		err := NewExternalAPIError("API call failed", cause)

		assert.Equal(t, ExternalAPIError, err.Type)
		assert.Equal(t, "API call failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewDatabaseError", func(t *testing.T) {
		cause := fmt.Errorf("connection lost")
		err := NewDatabaseError("database query failed", cause)

		assert.Equal(t, DatabaseError, err.Type)
		assert.Equal(t, "database query failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewConfigurationError", func(t *testing.T) {
		cause := fmt.Errorf("missing env var")
		err := NewConfigurationError("config loading failed", cause)

		assert.Equal(t, ConfigurationError, err.Type)
		assert.Equal(t, "config loading failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewEmailError", func(t *testing.T) {
		cause := fmt.Errorf("SMTP connection failed")
		err := NewEmailError("email sending failed", cause)

		assert.Equal(t, EmailError, err.Type)
		assert.Equal(t, "email sending failed", err.Message)
		assert.Equal(t, cause, err.Cause)
	})

	t.Run("NewTokenError", func(t *testing.T) {
		err := NewTokenError("token expired")

		assert.Equal(t, TokenError, err.Type)
		assert.Equal(t, "token expired", err.Message)
		assert.Nil(t, err.Cause)
	})
}

func TestErrorTypes(t *testing.T) {
	// Test that error type constants are correct
	assert.Equal(t, ErrorType("VALIDATION_ERROR"), ValidationError)
	assert.Equal(t, ErrorType("NOT_FOUND_ERROR"), NotFoundError)
	assert.Equal(t, ErrorType("ALREADY_EXISTS_ERROR"), AlreadyExistsError)
	assert.Equal(t, ErrorType("EXTERNAL_API_ERROR"), ExternalAPIError)
	assert.Equal(t, ErrorType("DATABASE_ERROR"), DatabaseError)
	assert.Equal(t, ErrorType("CONFIGURATION_ERROR"), ConfigurationError)
	assert.Equal(t, ErrorType("EMAIL_ERROR"), EmailError)
	assert.Equal(t, ErrorType("TOKEN_ERROR"), TokenError)
}

func TestErrorChaining(t *testing.T) {
	t.Run("ChainedErrors", func(t *testing.T) {
		originalErr := fmt.Errorf("connection refused")
		dbErr := NewDatabaseError("query failed", originalErr)
		serviceErr := Wrap(ExternalAPIError, "service unavailable", dbErr)

		// Test error message includes full chain
		expected := "EXTERNAL_API_ERROR: service unavailable (caused by: DATABASE_ERROR: query failed (caused by: connection refused))"
		assert.Equal(t, expected, serviceErr.Error())

		// Test unwrapping
		assert.Equal(t, dbErr, serviceErr.Unwrap())
		assert.Equal(t, originalErr, dbErr.Unwrap())
	})
}

func TestErrorComparison(t *testing.T) {
	t.Run("SameTypeAndMessage", func(t *testing.T) {
		err1 := NewValidationError("field required")
		err2 := NewValidationError("field required")

		// Errors should have same type and message but be different instances
		assert.Equal(t, err1.Type, err2.Type)
		assert.Equal(t, err1.Message, err2.Message)
		assert.NotSame(t, err1, err2)
	})

	t.Run("DifferentTypes", func(t *testing.T) {
		err1 := NewValidationError("test error")
		err2 := NewNotFoundError("test error")

		assert.NotEqual(t, err1.Type, err2.Type)
		assert.Equal(t, err1.Message, err2.Message)
	})
}
