package token

import (
	"time"

	"github.com/google/uuid"
)

// Token type string constants
const (
	TypeConfirmation = "confirmation"
	TypeUnsubscribe  = "unsubscribe"
	TypeUnknown      = "unknown"
)

// Type represents the type of authentication token
type Type string

// IsValid checks if the token type is valid
func (t Type) IsValid() bool {
	return t == TypeConfirmation || t == TypeUnsubscribe
}

// FromString converts string to Type, validating the input
func FromString(s string) Type {
	switch s {
	case TypeConfirmation:
		return TypeConfirmation
	case TypeUnsubscribe:
		return TypeUnsubscribe
	default:
		return TypeUnknown
	}
}

// Token represents an authentication or verification token
type Token struct {
	ID             uint
	Token          string
	SubscriptionID uint
	Type           Type
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// NewToken creates a new token for a subscription
func NewToken(subscriptionID uint, tokenType Type, expiresIn time.Duration) *Token {
	return &Token{
		Token:          uuid.New().String(),
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
		CreatedAt:      time.Now(),
	}
}

// IsExpired checks if the token has expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
