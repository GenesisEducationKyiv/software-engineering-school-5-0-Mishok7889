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

func TestEmailParams_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		params  EmailParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidEmailParams",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "Test Subject",
				Body:    "Test email body content",
				IsHTML:  true,
			},
			wantErr: false,
		},
		{
			name: "ValidPlainTextEmail",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "Plain Text Subject",
				Body:    "Plain text email body",
				IsHTML:  false,
			},
			wantErr: false,
		},
		{
			name: "EmptyRecipient",
			params: EmailParams{
				To:      "",
				Subject: "Test Subject",
				Body:    "Test body",
				IsHTML:  true,
			},
			wantErr: true,
			errMsg:  "recipient email is required",
		},
		{
			name: "EmptySubject",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "",
				Body:    "Test body",
				IsHTML:  true,
			},
			wantErr: true,
			errMsg:  "email subject is required",
		},
		{
			name: "EmptyBody",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "Test Subject",
				Body:    "",
				IsHTML:  true,
			},
			wantErr: true,
			errMsg:  "email body is required",
		},
		{
			name: "AllFieldsEmpty",
			params: EmailParams{
				To:      "",
				Subject: "",
				Body:    "",
				IsHTML:  false,
			},
			wantErr: true,
			errMsg:  "recipient email is required",
		},
		{
			name: "WhitespaceOnlyFields",
			params: EmailParams{
				To:      "   ",
				Subject: "   ",
				Body:    "   ",
				IsHTML:  true,
			},
			wantErr: true,
			errMsg:  "recipient email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotificationRequest_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		request NotificationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidHourlyRequest",
			request: NotificationRequest{
				Frequency: subscription.FrequencyHourly,
				City:      "London",
				Email:     "user@example.com",
			},
			wantErr: false,
		},
		{
			name: "ValidDailyRequest",
			request: NotificationRequest{
				Frequency: subscription.FrequencyDaily,
				City:      "Paris",
				Email:     "user@example.com",
			},
			wantErr: false,
		},
		{
			name: "InvalidFrequency",
			request: NotificationRequest{
				Frequency: subscription.FrequencyUnknown,
				City:      "London",
				Email:     "user@example.com",
			},
			wantErr: true,
			errMsg:  "frequency must be hourly or daily",
		},
		{
			name: "ValidRequestWithOptionalFields",
			request: NotificationRequest{
				Frequency: subscription.FrequencyDaily,
				City:      "", // Optional field
				Email:     "", // Optional field
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotificationStats_Fields(t *testing.T) {
	stats := NotificationStats{
		TotalSubscriptions:  100,
		HourlySubscriptions: 30,
		DailySubscriptions:  70,
		EmailsSent:          150,
		EmailsFailed:        5,
		LastUpdated:         time.Now(),
		LastSentAt:          time.Now().Add(-time.Hour),
		ProcessingTime:      2 * time.Second,
	}

	// Test that all fields can be set and retrieved
	assert.Equal(t, 100, stats.TotalSubscriptions)
	assert.Equal(t, 30, stats.HourlySubscriptions)
	assert.Equal(t, 70, stats.DailySubscriptions)
	assert.Equal(t, 150, stats.EmailsSent)
	assert.Equal(t, 5, stats.EmailsFailed)
	assert.NotZero(t, stats.LastUpdated)
	assert.NotZero(t, stats.LastSentAt)
	assert.Equal(t, 2*time.Second, stats.ProcessingTime)

	// Test zero values
	zeroStats := NotificationStats{}
	assert.Equal(t, 0, zeroStats.TotalSubscriptions)
	assert.Equal(t, 0, zeroStats.HourlySubscriptions)
	assert.Equal(t, 0, zeroStats.DailySubscriptions)
	assert.Equal(t, 0, zeroStats.EmailsSent)
	assert.Equal(t, 0, zeroStats.EmailsFailed)
	assert.True(t, zeroStats.LastUpdated.IsZero())
	assert.True(t, zeroStats.LastSentAt.IsZero())
	assert.Equal(t, time.Duration(0), zeroStats.ProcessingTime)
}

func TestTokenConstants(t *testing.T) {
	// Test that token type constants have expected values
	assert.Equal(t, "unsubscribe", TokenTypeUnsubscribe)
	assert.Equal(t, "confirmation", TokenTypeConfirmation)

	// Test that constants are strings (not custom types)
	assert.IsType(t, "", TokenTypeUnsubscribe)
	assert.IsType(t, "", TokenTypeConfirmation)
}

func TestEmailParams_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		params  EmailParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "VeryLongEmailAddress",
			params: EmailParams{
				To:      "verylongemailaddress123456789012345678901234567890@verylongdomainname123456789012345678901234567890.com",
				Subject: "Test",
				Body:    "Test",
				IsHTML:  false,
			},
			wantErr: false,
		},
		{
			name: "VeryLongSubject",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "This is a very long subject line that might be used in some email systems to provide detailed information about the content of the email message being sent to the recipient",
				Body:    "Test",
				IsHTML:  false,
			},
			wantErr: false,
		},
		{
			name: "VeryLongBody",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "Test",
				Body:    "This is a very long email body that contains a lot of text content. " + "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " + "Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " + "Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris. " + "Nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit. " + "In voluptate velit esse cillum dolore eu fugiat nulla pariatur. " + "Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia.",
				IsHTML:  true,
			},
			wantErr: false,
		},
		{
			name: "HTMLEmailBody",
			params: EmailParams{
				To:      "user@example.com",
				Subject: "HTML Email",
				Body:    "<html><body><h1>Hello</h1><p>This is an <b>HTML</b> email.</p></body></html>",
				IsHTML:  true,
			},
			wantErr: false,
		},
		{
			name: "SpecialCharactersInEmail",
			params: EmailParams{
				To:      "user+tag@example-domain.co.uk",
				Subject: "Subject with émojis 🎉 and special chars: àáâãäåæçèéêë",
				Body:    "Body with special characters: ñóôõöøùúûüý and symbols: €£¥$",
				IsHTML:  false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
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
