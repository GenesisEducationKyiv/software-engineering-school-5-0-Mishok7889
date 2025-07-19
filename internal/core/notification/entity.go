package notification

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"weatherapi.app/internal/core/subscription"
)

// Token type constants
const (
	TokenTypeUnsubscribe  = "unsubscribe"
	TokenTypeConfirmation = "confirmation"
)

// NotificationStats represents statistics about notification operations
type NotificationStats struct {
	TotalSubscriptions  int
	HourlySubscriptions int
	DailySubscriptions  int
	EmailsSent          int
	EmailsFailed        int
	LastUpdated         time.Time
	LastSentAt          time.Time
	ProcessingTime      time.Duration
}

// NotificationRequest represents a request to send notifications
type NotificationRequest struct {
	Frequency subscription.Frequency
	City      string
	Email     string
}

// Token represents a notification token for unsubscribe functionality
type Token struct {
	ID        uint
	Value     string
	Type      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewNotificationToken creates a new notification token
func NewNotificationToken(tokenType string, expiresIn time.Duration) *Token {
	return &Token{
		Value:     uuid.New().String(),
		Type:      tokenType,
		ExpiresAt: time.Now().Add(expiresIn),
		CreatedAt: time.Now(),
	}
}

// IsExpired checks if the token has expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid validates notification request
func (n *NotificationRequest) IsValid() error {
	if !n.Frequency.IsValid() {
		return errors.New("frequency must be hourly or daily")
	}
	return nil
}
