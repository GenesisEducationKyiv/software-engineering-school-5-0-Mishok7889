package subscription

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrequency_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		frequency Frequency
		expected  bool
	}{
		{
			name:      "ValidHourly",
			frequency: FrequencyHourly,
			expected:  true,
		},
		{
			name:      "ValidDaily",
			frequency: FrequencyDaily,
			expected:  true,
		},
		{
			name:      "InvalidFrequency",
			frequency: FrequencyUnknown,
			expected:  false,
		},
		{
			name:      "EmptyFrequency",
			frequency: FrequencyUnknown,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.frequency.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFrequency_String(t *testing.T) {
	tests := []struct {
		name      string
		frequency Frequency
		expected  string
	}{
		{
			name:      "HourlyString",
			frequency: FrequencyHourly,
			expected:  "hourly",
		},
		{
			name:      "DailyString",
			frequency: FrequencyDaily,
			expected:  "daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.frequency.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFrequency_JSON(t *testing.T) {
	tests := []struct {
		name      string
		frequency Frequency
		expected  string
	}{
		{
			name:      "MarshalHourly",
			frequency: FrequencyHourly,
			expected:  `"hourly"`,
		},
		{
			name:      "MarshalDaily",
			frequency: FrequencyDaily,
			expected:  `"daily"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			result, err := json.Marshal(tt.frequency)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))

			// Test unmarshaling
			var unmarshaled Frequency
			err = json.Unmarshal(result, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.frequency, unmarshaled)
		})
	}
}

func TestTokenType_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		tokenType TokenType
		expected  bool
	}{
		{
			name:      "ValidConfirmation",
			tokenType: TokenTypeConfirmation,
			expected:  true,
		},
		{
			name:      "ValidUnsubscribe",
			tokenType: TokenTypeUnsubscribe,
			expected:  true,
		},
		{
			name:      "InvalidTokenType",
			tokenType: TokenTypeUnknown,
			expected:  false,
		},
		{
			name:      "EmptyTokenType",
			tokenType: TokenTypeUnknown,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tokenType.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewToken(t *testing.T) {
	subscriptionID := uint(123)
	tokenType := TokenTypeConfirmation
	expiresIn := 24 * time.Hour

	token := NewToken(subscriptionID, tokenType, expiresIn)

	assert.NotNil(t, token)
	assert.NotEmpty(t, token.Token)
	assert.Equal(t, subscriptionID, token.SubscriptionID)
	assert.Equal(t, tokenType, token.Type)
	assert.True(t, token.ExpiresAt.After(time.Now()))
	assert.True(t, token.ExpiresAt.Before(time.Now().Add(25*time.Hour)))
	assert.True(t, token.CreatedAt.Before(time.Now().Add(time.Minute)))
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *Token
		expected  bool
	}{
		{
			name: "NotExpired",
			setupFunc: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(time.Hour),
				}
			},
			expected: false,
		},
		{
			name: "Expired",
			setupFunc: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(-time.Hour),
				}
			},
			expected: true,
		},
		{
			name: "JustExpired",
			setupFunc: func() *Token {
				return &Token{
					ExpiresAt: time.Now().Add(-time.Millisecond),
				}
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupFunc()
			result := token.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeFromUnix(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		expected  time.Time
	}{
		{
			name:      "ZeroTimestamp",
			timestamp: 0,
			expected:  time.Unix(0, 0).UTC(),
		},
		{
			name:      "PositiveTimestamp",
			timestamp: 1609459200, // 2021-01-01 00:00:00 UTC
			expected:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "NegativeTimestamp",
			timestamp: -86400, // 1969-12-31 00:00:00 UTC
			expected:  time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeFromUnix(tt.timestamp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscription_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		subscription Subscription
		wantErr      bool
		errMsg       string
	}{
		{
			name: "ValidSubscription",
			subscription: Subscription{
				Email:     "user@example.com",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: false,
		},
		{
			name: "EmptyEmail",
			subscription: Subscription{
				Email:     "",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "email cannot be empty",
		},
		{
			name: "WhitespaceOnlyEmail",
			subscription: Subscription{
				Email:     "   ",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "email cannot be empty",
		},
		{
			name: "InvalidEmailFormat",
			subscription: Subscription{
				Email:     "invalid-email",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "EmptyCity",
			subscription: Subscription{
				Email:     "user@example.com",
				City:      "",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name: "WhitespaceOnlyCity",
			subscription: Subscription{
				Email:     "user@example.com",
				City:      "   ",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name: "InvalidFrequency",
			subscription: Subscription{
				Email:     "user@example.com",
				City:      "London",
				Frequency: FrequencyUnknown,
			},
			wantErr: true,
			errMsg:  "frequency must be hourly or daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.subscription.IsValid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionRequest_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		request SubscriptionRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "ValidRequest",
			request: SubscriptionRequest{
				Email:     "user@example.com",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: false,
		},
		{
			name: "InvalidEmail",
			request: SubscriptionRequest{
				Email:     "invalid-email",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "EmptyCity",
			request: SubscriptionRequest{
				Email:     "user@example.com",
				City:      "",
				Frequency: FrequencyDaily,
			},
			wantErr: true,
			errMsg:  "city cannot be empty",
		},
		{
			name: "InvalidFrequency",
			request: SubscriptionRequest{
				Email:     "user@example.com",
				City:      "London",
				Frequency: FrequencyUnknown,
			},
			wantErr: true,
			errMsg:  "frequency must be hourly or daily",
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

func TestSubscription_Confirm(t *testing.T) {
	subscription := &Subscription{
		Email:     "user@example.com",
		City:      "London",
		Frequency: FrequencyDaily,
		Confirmed: false,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	oldUpdateTime := subscription.UpdatedAt

	subscription.Confirm()

	// Verify confirmation and timestamp update
	assert.True(t, subscription.Confirmed)
	require.Eventually(t, func() bool {
		return subscription.UpdatedAt.After(oldUpdateTime)
	}, time.Second, time.Microsecond)
}

func TestNewSubscription(t *testing.T) {
	email := "  user@example.com  "
	city := "  London  "
	frequency := FrequencyDaily

	subscription := NewSubscription(email, city, frequency)

	assert.NotNil(t, subscription)
	assert.Equal(t, "user@example.com", subscription.Email) // Trimmed
	assert.Equal(t, "London", subscription.City)            // Trimmed
	assert.Equal(t, frequency, subscription.Frequency)
	assert.False(t, subscription.Confirmed)
	assert.True(t, subscription.CreatedAt.Before(time.Now().Add(time.Minute)))
	assert.True(t, subscription.UpdatedAt.Before(time.Now().Add(time.Minute)))
	assert.Equal(t, subscription.CreatedAt, subscription.UpdatedAt)
}

func TestSubscription_IsConfirmed(t *testing.T) {
	tests := []struct {
		name         string
		subscription Subscription
		expected     bool
	}{
		{
			name: "ConfirmedSubscription",
			subscription: Subscription{
				Confirmed: true,
			},
			expected: true,
		},
		{
			name: "UnconfirmedSubscription",
			subscription: Subscription{
				Confirmed: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.subscription.IsConfirmed()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscription_IsExpired(t *testing.T) {
	tests := []struct {
		name         string
		subscription Subscription
		expected     bool
	}{
		{
			name: "ConfirmedSubscription_NeverExpires",
			subscription: Subscription{
				Confirmed: true,
				CreatedAt: time.Now().Add(-48 * time.Hour), // 2 days old
			},
			expected: false,
		},
		{
			name: "UnconfirmedSubscription_NotExpired",
			subscription: Subscription{
				Confirmed: false,
				CreatedAt: time.Now().Add(-12 * time.Hour), // 12 hours old
			},
			expected: false,
		},
		{
			name: "UnconfirmedSubscription_Expired",
			subscription: Subscription{
				Confirmed: false,
				CreatedAt: time.Now().Add(-25 * time.Hour), // 25 hours old
			},
			expected: true,
		},
		{
			name: "UnconfirmedSubscription_JustExpired",
			subscription: Subscription{
				Confirmed: false,
				CreatedAt: time.Now().Add(-24*time.Hour - time.Minute), // Just over 24 hours
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.subscription.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscriptionRequest_ToSubscription(t *testing.T) {
	request := SubscriptionRequest{
		Email:     "  user@example.com  ",
		City:      "  London  ",
		Frequency: FrequencyHourly,
	}

	subscription := request.ToSubscription()

	assert.NotNil(t, subscription)
	assert.Equal(t, "user@example.com", subscription.Email) // Trimmed
	assert.Equal(t, "London", subscription.City)            // Trimmed
	assert.Equal(t, FrequencyHourly, subscription.Frequency)
	assert.False(t, subscription.Confirmed)
	assert.True(t, subscription.CreatedAt.Before(time.Now().Add(time.Minute)))
	assert.True(t, subscription.UpdatedAt.Before(time.Now().Add(time.Minute)))
}

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{
			name:  "ValidSimpleEmail",
			email: "test@example.com",
			valid: true,
		},
		{
			name:  "ValidEmailWithNumbers",
			email: "user123@domain123.com",
			valid: true,
		},
		{
			name:  "ValidEmailWithDots",
			email: "first.last@example.com",
			valid: true,
		},
		{
			name:  "ValidEmailWithPlus",
			email: "user+tag@example.com",
			valid: true,
		},
		{
			name:  "ValidEmailWithUnderscore",
			email: "user_name@example.com",
			valid: true,
		},
		{
			name:  "ValidEmailWithSubdomain",
			email: "user@mail.example.com",
			valid: true,
		},
		{
			name:  "InvalidEmailNoAt",
			email: "userexample.com",
			valid: false,
		},
		{
			name:  "InvalidEmailNoDomain",
			email: "user@",
			valid: false,
		},
		{
			name:  "InvalidEmailNoUser",
			email: "@example.com",
			valid: false,
		},
		{
			name:  "InvalidEmailNoTLD",
			email: "user@domain",
			valid: false,
		},
		{
			name:  "InvalidEmailSpaces",
			email: "user name@example.com",
			valid: false,
		},
		{
			name:  "InvalidEmailMultipleAt",
			email: "user@@example.com",
			valid: false,
		},
		{
			name:  "EmptyEmail",
			email: "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := Subscription{
				Email:     tt.email,
				City:      "London",
				Frequency: FrequencyDaily,
			}

			err := subscription.validateEmail()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			// Also test with SubscriptionRequest
			request := SubscriptionRequest{
				Email:     tt.email,
				City:      "London",
				Frequency: FrequencyDaily,
			}

			err = request.validateEmail()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
