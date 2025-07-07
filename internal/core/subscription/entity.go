package subscription

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Subscription represents a user's weather notification subscription
type Subscription struct {
	ID        uint
	Email     string
	City      string
	Frequency Frequency
	Confirmed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Frequency represents subscription frequency options
type Frequency int

const (
	FrequencyUnknown Frequency = iota
	FrequencyHourly
	FrequencyDaily
)

// String returns the string representation of frequency
func (f Frequency) String() string {
	switch f {
	case FrequencyHourly:
		return "hourly"
	case FrequencyDaily:
		return "daily"
	default:
		return "unknown"
	}
}

// IsValid checks if the frequency value is valid
func (f Frequency) IsValid() bool {
	return f == FrequencyHourly || f == FrequencyDaily
}

// FromString converts string to Frequency enum
func FrequencyFromString(s string) Frequency {
	switch s {
	case "hourly":
		return FrequencyHourly
	case "daily":
		return FrequencyDaily
	default:
		return FrequencyUnknown
	}
}

// UnmarshalJSON implements json.Unmarshaler interface
func (f *Frequency) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*f = FrequencyFromString(s)
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (f Frequency) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// UnmarshalText implements encoding.TextUnmarshaler for form parsing
func (f *Frequency) UnmarshalText(text []byte) error {
	*f = FrequencyFromString(string(text))
	return nil
}

// MarshalText implements encoding.TextMarshaler for form parsing
func (f Frequency) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// TokenType represents the type of authentication token
type TokenType int

const (
	TokenTypeUnknown TokenType = iota
	TokenTypeConfirmation
	TokenTypeUnsubscribe
)

// String returns the string representation of token type
func (t TokenType) String() string {
	switch t {
	case TokenTypeConfirmation:
		return "confirmation"
	case TokenTypeUnsubscribe:
		return "unsubscribe"
	default:
		return "unknown"
	}
}

// IsValid checks if the token type is valid
func (t TokenType) IsValid() bool {
	return t == TokenTypeConfirmation || t == TokenTypeUnsubscribe
}

// TokenTypeFromString converts string to TokenType enum
func TokenTypeFromString(s string) TokenType {
	switch s {
	case "confirmation":
		return TokenTypeConfirmation
	case "unsubscribe":
		return TokenTypeUnsubscribe
	default:
		return TokenTypeUnknown
	}
}

// Token represents an authentication or verification token
type Token struct {
	ID             uint
	Token          string
	SubscriptionID uint
	Type           TokenType
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// SubscriptionRequest represents data required to create a subscription
type SubscriptionRequest struct {
	Email     string    `json:"email" form:"email" binding:"required,email"`
	City      string    `json:"city" form:"city" binding:"required"`
	Frequency Frequency `json:"frequency" form:"frequency" binding:"required"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// NewToken creates a new token for a subscription
func NewToken(subscriptionID uint, tokenType TokenType, expiresIn time.Duration) *Token {
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

// TimeFromUnix converts Unix timestamp to time.Time
func TimeFromUnix(timestamp int64) time.Time {
	return time.Unix(timestamp, 0).UTC()
}

// IsValid validates subscription data
func (s *Subscription) IsValid() error {
	if err := s.validateEmail(); err != nil {
		return err
	}
	if err := s.validateCity(); err != nil {
		return err
	}
	if err := s.validateFrequency(); err != nil {
		return err
	}
	return nil
}

// IsValid validates subscription request
func (sr *SubscriptionRequest) IsValid() error {
	if err := sr.validateEmail(); err != nil {
		return err
	}
	if err := sr.validateCity(); err != nil {
		return err
	}
	if err := sr.validateFrequency(); err != nil {
		return err
	}
	return nil
}

// Confirm marks subscription as confirmed
func (s *Subscription) Confirm() {
	s.Confirmed = true
	s.UpdatedAt = time.Now()
}

// NewSubscription creates a new subscription with current timestamp
func NewSubscription(email, city string, frequency Frequency) *Subscription {
	now := time.Now()
	return &Subscription{
		Email:     strings.TrimSpace(email),
		City:      strings.TrimSpace(city),
		Frequency: frequency,
		Confirmed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsConfirmed checks if subscription is confirmed
func (s *Subscription) IsConfirmed() bool {
	return s.Confirmed
}

// IsExpired checks if subscription should be considered expired (24 hours for unconfirmed)
func (s *Subscription) IsExpired() bool {
	if s.Confirmed {
		return false
	}
	return time.Since(s.CreatedAt) > 24*time.Hour
}

func (s *Subscription) validateEmail() error {
	if strings.TrimSpace(s.Email) == "" {
		return errors.New("email cannot be empty")
	}
	if !emailRegex.MatchString(s.Email) {
		return errors.New("invalid email format")
	}
	return nil
}

func (s *Subscription) validateCity() error {
	if strings.TrimSpace(s.City) == "" {
		return errors.New("city cannot be empty")
	}
	return nil
}

func (s *Subscription) validateFrequency() error {
	if !s.Frequency.IsValid() {
		return errors.New("frequency must be hourly or daily")
	}
	return nil
}

func (sr *SubscriptionRequest) validateEmail() error {
	if strings.TrimSpace(sr.Email) == "" {
		return errors.New("email cannot be empty")
	}
	if !emailRegex.MatchString(sr.Email) {
		return errors.New("invalid email format")
	}
	return nil
}

func (sr *SubscriptionRequest) validateCity() error {
	if strings.TrimSpace(sr.City) == "" {
		return errors.New("city cannot be empty")
	}
	return nil
}

func (sr *SubscriptionRequest) validateFrequency() error {
	if sr.Frequency != FrequencyHourly && sr.Frequency != FrequencyDaily {
		return errors.New("frequency must be hourly or daily")
	}
	return nil
}

// ToSubscription converts request to subscription entity
func (sr *SubscriptionRequest) ToSubscription() *Subscription {
	now := time.Now()
	return &Subscription{
		Email:     strings.TrimSpace(sr.Email),
		City:      strings.TrimSpace(sr.City),
		Frequency: sr.Frequency,
		Confirmed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
