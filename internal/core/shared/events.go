package shared

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// buildHeaders creates common headers for events and commands
func buildHeaders(typeKey, typeValue, idKey, idValue, correlationID string) map[string]string {
	headers := make(map[string]string)
	headers[typeKey] = typeValue
	headers[idKey] = idValue
	if correlationID != "" {
		headers["correlation_id"] = correlationID
	}
	return headers
}

type EventType string

const (
	EventTypeWeatherDataFetched    EventType = "weather.data.fetched"
	EventTypeWeatherProviderFailed EventType = "weather.provider.failed"
	EventTypeSubscriptionCreated   EventType = "subscription.created"
	EventTypeSubscriptionUpdated   EventType = "subscription.updated"
	EventTypeSubscriptionCancelled EventType = "subscription.cancelled"
)

type CommandType string

const (
	CommandTypeSendWeatherUpdates CommandType = "notification.weather.send"
	CommandTypeSendEmail          CommandType = "notification.email.send"
)

type WeatherDataFetchedEvent struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	City          string    `json:"city"`
	Temperature   float64   `json:"temperature"`
	Humidity      float64   `json:"humidity"`
	Description   string    `json:"description"`
}

func NewWeatherDataFetchedEvent(city string, temperature, humidity float64, description string) *WeatherDataFetchedEvent {
	return &WeatherDataFetchedEvent{
		EventID:     uuid.New().String(),
		EventType:   EventTypeWeatherDataFetched,
		Timestamp:   time.Now().UTC(),
		City:        city,
		Temperature: temperature,
		Humidity:    humidity,
		Description: description,
	}
}

func (e *WeatherDataFetchedEvent) Topic() string {
	return string(e.EventType)
}

func (e *WeatherDataFetchedEvent) Data() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (e *WeatherDataFetchedEvent) ID() string {
	return e.EventID
}

func (e *WeatherDataFetchedEvent) Headers() map[string]string {
	return buildHeaders("event_type", string(e.EventType), "event_id", e.EventID, e.CorrelationID)
}

type SubscriptionCreatedEvent struct {
	EventID        string    `json:"event_id"`
	EventType      EventType `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CorrelationID  string    `json:"correlation_id,omitempty"`
	SubscriptionID string    `json:"subscription_id"`
	UserID         string    `json:"user_id"`
	City           string    `json:"city"`
	Frequency      string    `json:"frequency"`
	Email          string    `json:"email"`
}

func NewSubscriptionCreatedEvent(subscriptionID, userID, city, frequency, email string) *SubscriptionCreatedEvent {
	return &SubscriptionCreatedEvent{
		EventID:        uuid.New().String(),
		EventType:      EventTypeSubscriptionCreated,
		Timestamp:      time.Now().UTC(),
		SubscriptionID: subscriptionID,
		UserID:         userID,
		City:           city,
		Frequency:      frequency,
		Email:          email,
	}
}

func (e *SubscriptionCreatedEvent) Topic() string {
	return string(e.EventType)
}

func (e *SubscriptionCreatedEvent) Data() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (e *SubscriptionCreatedEvent) ID() string {
	return e.EventID
}

func (e *SubscriptionCreatedEvent) Headers() map[string]string {
	return buildHeaders("event_type", string(e.EventType), "event_id", e.EventID, e.CorrelationID)
}

type SubscriptionCancelledEvent struct {
	EventID        string    `json:"event_id"`
	EventType      EventType `json:"event_type"`
	Timestamp      time.Time `json:"timestamp"`
	CorrelationID  string    `json:"correlation_id,omitempty"`
	SubscriptionID string    `json:"subscription_id"`
	UserID         string    `json:"user_id"`
	Email          string    `json:"email"`
}

func NewSubscriptionCancelledEvent(subscriptionID, userID, email string) *SubscriptionCancelledEvent {
	return &SubscriptionCancelledEvent{
		EventID:        uuid.New().String(),
		EventType:      EventTypeSubscriptionCancelled,
		Timestamp:      time.Now().UTC(),
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Email:          email,
	}
}

func (e *SubscriptionCancelledEvent) Topic() string {
	return string(e.EventType)
}

func (e *SubscriptionCancelledEvent) Data() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (e *SubscriptionCancelledEvent) ID() string {
	return e.EventID
}

func (e *SubscriptionCancelledEvent) Headers() map[string]string {
	return buildHeaders("event_type", string(e.EventType), "event_id", e.EventID, e.CorrelationID)
}

type SendWeatherUpdatesCommand struct {
	CommandID     string      `json:"command_id"`
	CommandType   CommandType `json:"command_type"`
	Timestamp     time.Time   `json:"timestamp"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Frequency     string      `json:"frequency"`
}

func NewSendWeatherUpdatesCommand(frequency string) *SendWeatherUpdatesCommand {
	return &SendWeatherUpdatesCommand{
		CommandID:   uuid.New().String(),
		CommandType: CommandTypeSendWeatherUpdates,
		Timestamp:   time.Now().UTC(),
		Frequency:   frequency,
	}
}

func (c *SendWeatherUpdatesCommand) Topic() string {
	return string(c.CommandType)
}

func (c *SendWeatherUpdatesCommand) Data() []byte {
	data, _ := json.Marshal(c)
	return data
}

func (c *SendWeatherUpdatesCommand) ID() string {
	return c.CommandID
}

func (c *SendWeatherUpdatesCommand) Headers() map[string]string {
	return buildHeaders("command_type", string(c.CommandType), "command_id", c.CommandID, c.CorrelationID)
}
