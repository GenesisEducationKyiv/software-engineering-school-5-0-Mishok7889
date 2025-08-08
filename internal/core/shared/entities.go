package shared

import (
	"time"
)

// WeatherData represents weather information used across services
type WeatherData struct {
	City        string
	Temperature float64
	Humidity    float64
	Description string
	LastUpdated time.Time
}

// Subscription represents subscription data used across services
type Subscription struct {
	ID        string
	UserID    string
	Email     string
	City      string
	Frequency Frequency
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Frequency represents subscription frequency used across services
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

// FrequencyFromString converts string to Frequency enum
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

// EmailRequest represents an email sending request
type EmailRequest struct {
	To      string
	Subject string
	Body    string
}
