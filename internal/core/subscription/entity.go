package subscription

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
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

// SubscriptionRequest represents data required to create a subscription
type SubscriptionRequest struct {
	Email     string    `json:"email" form:"email" binding:"required,email"`
	City      string    `json:"city" form:"city" binding:"required"`
	Frequency Frequency `json:"frequency" form:"frequency" binding:"required"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

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
	return time.Since(s.CreatedAt) > SubscriptionTTL
}

func (s *Subscription) validateEmail() error {
	if strings.TrimSpace(s.Email) == "" {
		return errors.New(ErrEmailEmpty)
	}
	if !emailRegex.MatchString(s.Email) {
		return errors.New(ErrEmailInvalid)
	}
	return nil
}

func (s *Subscription) validateCity() error {
	if strings.TrimSpace(s.City) == "" {
		return errors.New(ErrCityEmpty)
	}
	return nil
}

func (s *Subscription) validateFrequency() error {
	if !s.Frequency.IsValid() {
		return errors.New(ErrFrequencyRequired)
	}
	return nil
}

func (sr *SubscriptionRequest) validateEmail() error {
	if strings.TrimSpace(sr.Email) == "" {
		return errors.New(ErrEmailEmpty)
	}
	if !emailRegex.MatchString(sr.Email) {
		return errors.New(ErrEmailInvalid)
	}
	return nil
}

func (sr *SubscriptionRequest) validateCity() error {
	if strings.TrimSpace(sr.City) == "" {
		return errors.New(ErrCityEmpty)
	}
	return nil
}

func (sr *SubscriptionRequest) validateFrequency() error {
	if sr.Frequency != FrequencyHourly && sr.Frequency != FrequencyDaily {
		return errors.New(ErrFrequencyRequired)
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
