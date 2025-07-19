package notification

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/core/subscription"
)

func TestNewNotificationToken(t *testing.T) {
	tests := []struct {
		name       string
		tokenType  string
		expiresIn  time.Duration
		wantErrMsg string
	}{
		{
			name:      "ValidConfirmationToken",
			tokenType: TokenTypeConfirmation,
			expiresIn: 24 * time.Hour,
		},
		{
			name:      "ValidUnsubscribeToken",
			tokenType: TokenTypeUnsubscribe,
			expiresIn: 365 * 24 * time.Hour,
		},
		{
			name:      "ShortExpiryToken",
			tokenType: TokenTypeConfirmation,
			expiresIn: 5 * time.Minute,
		},
		{
			name:      "LongExpiryToken",
			tokenType: TokenTypeUnsubscribe,
			expiresIn: 10 * 365 * 24 * time.Hour, // 10 years
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			token := NewNotificationToken(tt.tokenType, tt.expiresIn)
			after := time.Now()

			assert.NotNil(t, token)
			assert.NotEmpty(t, token.Value)
			assert.Equal(t, tt.tokenType, token.Type)

			// Verify UUID format (36 characters with dashes)
			assert.Len(t, token.Value, 36)
			assert.Contains(t, token.Value, "-")

			// Verify timestamps are reasonable
			assert.True(t, token.CreatedAt.After(before.Add(-time.Second)))
			assert.True(t, token.CreatedAt.Before(after.Add(time.Second)))

			// Verify expiry calculation
			expectedExpiry := token.CreatedAt.Add(tt.expiresIn)
			assert.True(t, token.ExpiresAt.Sub(expectedExpiry) < time.Second)
			assert.True(t, token.ExpiresAt.Sub(expectedExpiry) > -time.Second)
		})
	}
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Token
		expected bool
	}{
		{
			name: "NotExpired",
			setup: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(time.Hour),
				}
			},
			expected: false,
		},
		{
			name: "Expired",
			setup: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(-time.Hour),
				}
			},
			expected: true,
		},
		{
			name: "JustExpired",
			setup: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(-time.Millisecond),
				}
			},
			expected: true,
		},
		{
			name: "ExpiresNow",
			setup: func() *Token {
				// Set expiration slightly in the future to avoid race condition
				return &Token{
					ExpiresAt: time.Now().Add(10 * time.Millisecond),
				}
			},
			expected: false, // Should be false since it expires slightly in the future
		},
		{
			name: "ExpiresInFuture",
			setup: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}
			},
			expected: false,
		},
		{
			name: "ExpiredLongAgo",
			setup: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(-24 * time.Hour),
				}
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setup()
			result := token.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToken_FieldAssignment(t *testing.T) {
	// Test that Token struct fields can be assigned and retrieved correctly
	token := &Token{
		ID:        123,
		Value:     "custom-token-value",
		Type:      TokenTypeConfirmation,
		ExpiresAt: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	assert.Equal(t, uint(123), token.ID)
	assert.Equal(t, "custom-token-value", token.Value)
	assert.Equal(t, TokenTypeConfirmation, token.Type)
	assert.Equal(t, 2024, token.ExpiresAt.Year())
	assert.Equal(t, 2024, token.CreatedAt.Year())

	// Test zero values
	zeroToken := &Token{}
	assert.Equal(t, uint(0), zeroToken.ID)
	assert.Equal(t, "", zeroToken.Value)
	assert.Equal(t, "", zeroToken.Type)
	assert.True(t, zeroToken.ExpiresAt.IsZero())
	assert.True(t, zeroToken.CreatedAt.IsZero())
}

func TestNotificationRequest_FieldAssignment(t *testing.T) {
	// Test that NotificationRequest struct fields can be assigned and retrieved correctly
	request := NotificationRequest{
		Frequency: subscription.FrequencyHourly,
		City:      "New York",
		Email:     "test@example.com",
	}

	assert.Equal(t, subscription.FrequencyHourly, request.Frequency)
	assert.Equal(t, "New York", request.City)
	assert.Equal(t, "test@example.com", request.Email)

	// Test field modification
	request.Frequency = subscription.FrequencyDaily
	request.City = "Los Angeles"
	request.Email = "updated@example.com"

	assert.Equal(t, subscription.FrequencyDaily, request.Frequency)
	assert.Equal(t, "Los Angeles", request.City)
	assert.Equal(t, "updated@example.com", request.Email)
}

func TestNewNotificationToken_UniqueValues(t *testing.T) {
	// Test that multiple tokens created in succession have unique values
	tokens := make([]*Token, 10)
	for i := 0; i < 10; i++ {
		tokens[i] = NewNotificationToken(TokenTypeConfirmation, 24*time.Hour)
	}

	// Verify all tokens have unique values
	seenValues := make(map[string]bool)
	for i, token := range tokens {
		assert.NotEmpty(t, token.Value, "Token %d should have a non-empty value", i)
		assert.False(t, seenValues[token.Value], "Token %d value should be unique", i)
		seenValues[token.Value] = true
	}
}
