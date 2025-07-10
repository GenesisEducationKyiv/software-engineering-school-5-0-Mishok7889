package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *AppError
		expected string
	}{
		{
			name: "ErrorWithoutCause",
			setup: func() *AppError {
				return New(ValidationError, "test validation error")
			},
			expected: "VALIDATION_ERROR: test validation error",
		},
		{
			name: "ErrorWithCause",
			setup: func() *AppError {
				cause := fmt.Errorf("original error")
				return Wrap(DatabaseError, "database operation failed", cause)
			},
			expected: "DATABASE_ERROR: database operation failed (caused by: original error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*AppError, error)
		expected error
	}{
		{
			name: "ErrorWithCause",
			setup: func() (*AppError, error) {
				cause := fmt.Errorf("original error")
				err := Wrap(ExternalAPIError, "API call failed", cause)
				return err, cause
			},
		},
		{
			name: "ErrorWithoutCause",
			setup: func() (*AppError, error) {
				err := New(NotFoundError, "resource not found")
				return err, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, expectedCause := tt.setup()
			unwrapped := err.Unwrap()
			assert.Equal(t, expectedCause, unwrapped)
		})
	}
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
	tests := []struct {
		name         string
		constructor  func() *AppError
		expectedType ErrorType
		expectedMsg  string
		hasCause     bool
	}{
		{
			name: "NewValidationError",
			constructor: func() *AppError {
				return NewValidationError("field is required")
			},
			expectedType: ValidationError,
			expectedMsg:  "field is required",
			hasCause:     false,
		},
		{
			name: "NewNotFoundError",
			constructor: func() *AppError {
				return NewNotFoundError("resource not found")
			},
			expectedType: NotFoundError,
			expectedMsg:  "resource not found",
			hasCause:     false,
		},
		{
			name: "NewAlreadyExistsError",
			constructor: func() *AppError {
				return NewAlreadyExistsError("resource already exists")
			},
			expectedType: AlreadyExistsError,
			expectedMsg:  "resource already exists",
			hasCause:     false,
		},
		{
			name: "NewExternalAPIError",
			constructor: func() *AppError {
				cause := fmt.Errorf("network timeout")
				return NewExternalAPIError("API call failed", cause)
			},
			expectedType: ExternalAPIError,
			expectedMsg:  "API call failed",
			hasCause:     true,
		},
		{
			name: "NewDatabaseError",
			constructor: func() *AppError {
				cause := fmt.Errorf("connection lost")
				return NewDatabaseError("database query failed", cause)
			},
			expectedType: DatabaseError,
			expectedMsg:  "database query failed",
			hasCause:     true,
		},
		{
			name: "NewConfigurationError",
			constructor: func() *AppError {
				cause := fmt.Errorf("missing env var")
				return NewConfigurationError("config loading failed", cause)
			},
			expectedType: ConfigurationError,
			expectedMsg:  "config loading failed",
			hasCause:     true,
		},
		{
			name: "NewEmailError",
			constructor: func() *AppError {
				cause := fmt.Errorf("SMTP connection failed")
				return NewEmailError("email sending failed", cause)
			},
			expectedType: EmailError,
			expectedMsg:  "email sending failed",
			hasCause:     true,
		},
		{
			name: "NewTokenError",
			constructor: func() *AppError {
				return NewTokenError("token expired")
			},
			expectedType: TokenError,
			expectedMsg:  "token expired",
			hasCause:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()

			assert.Equal(t, tt.expectedType, err.Type)
			assert.Equal(t, tt.expectedMsg, err.Message)

			if tt.hasCause {
				assert.NotNil(t, err.Cause)
			} else {
				assert.Nil(t, err.Cause)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		expected  string
	}{
		{"ValidationError", ValidationError, "VALIDATION_ERROR"},
		{"NotFoundError", NotFoundError, "NOT_FOUND_ERROR"},
		{"AlreadyExistsError", AlreadyExistsError, "ALREADY_EXISTS_ERROR"},
		{"ExternalAPIError", ExternalAPIError, "EXTERNAL_API_ERROR"},
		{"DatabaseError", DatabaseError, "DATABASE_ERROR"},
		{"ConfigurationError", ConfigurationError, "CONFIGURATION_ERROR"},
		{"EmailError", EmailError, "EMAIL_ERROR"},
		{"TokenError", TokenError, "TOKEN_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, ErrorType(tt.expected), tt.errorType)
		})
	}
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
	tests := []struct {
		name         string
		err1Setup    func() *AppError
		err2Setup    func() *AppError
		sameType     bool
		sameMessage  bool
		shouldBeSame bool
	}{
		{
			name: "SameTypeAndMessage",
			err1Setup: func() *AppError {
				return NewValidationError("field required")
			},
			err2Setup: func() *AppError {
				return NewValidationError("field required")
			},
			sameType:     true,
			sameMessage:  true,
			shouldBeSame: false, // Different instances
		},
		{
			name: "DifferentTypes",
			err1Setup: func() *AppError {
				return NewValidationError("test error")
			},
			err2Setup: func() *AppError {
				return NewNotFoundError("test error")
			},
			sameType:     false,
			sameMessage:  true,
			shouldBeSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err1 := tt.err1Setup()
			err2 := tt.err2Setup()

			if tt.sameType {
				assert.Equal(t, err1.Type, err2.Type)
			} else {
				assert.NotEqual(t, err1.Type, err2.Type)
			}

			if tt.sameMessage {
				assert.Equal(t, err1.Message, err2.Message)
			} else {
				assert.NotEqual(t, err1.Message, err2.Message)
			}

			if tt.shouldBeSame {
				assert.Same(t, err1, err2)
			} else {
				assert.NotSame(t, err1, err2)
			}
		})
	}
}
