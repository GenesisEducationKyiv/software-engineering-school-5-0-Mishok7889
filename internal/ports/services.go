package ports

import (
	"context"
	"time"
)

// Service ports for cross-context communication

// WeatherService defines the contract for weather operations used by other bounded contexts
type WeatherService interface {
	GetWeather(ctx context.Context, city string) (*WeatherServiceData, error)
	GetProviderInfo(ctx context.Context) ProviderInfo
}

// WeatherServiceData represents weather data for cross-context communication
type WeatherServiceData struct {
	Temperature float64
	Humidity    float64
	Description string
	City        string
	Timestamp   time.Time
}

// SubscriptionService defines the contract for subscription operations used by other bounded contexts
type SubscriptionService interface {
	GetConfirmedSubscriptions(ctx context.Context, frequency string) ([]*SubscriptionServiceData, error)
	FindByID(ctx context.Context, id uint) (*SubscriptionServiceData, error)
}

// SubscriptionServiceData represents subscription data for cross-context communication
type SubscriptionServiceData struct {
	ID               uint
	Email            string
	City             string
	Frequency        string
	Confirmed        bool
	UnsubscribeToken string
}
