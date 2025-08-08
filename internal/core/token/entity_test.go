package token

import (
	"testing"
	"time"
)

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tokenType Type
		expected  string
	}{
		{TypeConfirmation, "confirmation"},
		{TypeUnsubscribe, "unsubscribe"},
		{TypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := string(tt.tokenType); got != tt.expected {
			t.Errorf("Type.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Type
	}{
		{"confirmation", TypeConfirmation},
		{"unsubscribe", TypeUnsubscribe},
		{"invalid", TypeUnknown},
		{"", TypeUnknown},
	}

	for _, tt := range tests {
		if got := FromString(tt.input); got != tt.expected {
			t.Errorf("FromString(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNewToken(t *testing.T) {
	subscriptionID := uint(123)
	tokenType := Type(TypeConfirmation)
	expiresIn := 24 * time.Hour

	token := NewToken(subscriptionID, tokenType, expiresIn)

	if token.SubscriptionID != subscriptionID {
		t.Errorf("NewToken() SubscriptionID = %v, want %v", token.SubscriptionID, subscriptionID)
	}

	if token.Type != tokenType {
		t.Errorf("NewToken() Type = %v, want %v", token.Type, tokenType)
	}

	if token.Token == "" {
		t.Error("NewToken() Token should not be empty")
	}

	if token.IsExpired() {
		t.Error("NewToken() should not be expired immediately")
	}
}

func TestTokenIsExpired(t *testing.T) {
	token := &Token{
		ExpiresAt: time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	if !token.IsExpired() {
		t.Error("Token should be expired")
	}

	token.ExpiresAt = time.Now().Add(1 * time.Hour) // 1 hour from now
	if token.IsExpired() {
		t.Error("Token should not be expired")
	}
}
