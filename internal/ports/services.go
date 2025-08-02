package ports

import (
	"context"
	"time"

	"weatherapi.app/internal/core/shared"
)

// Service ports for cross-context communication

// WeatherService defines the contract for weather operations used by other bounded contexts
type WeatherService interface {
	GetWeather(ctx context.Context, city string) (*WeatherServiceData, error)
	GetWeatherByCity(ctx context.Context, city string) (*shared.WeatherData, error)
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
	GetActiveSubscriptionsByCity(ctx context.Context, city string) ([]shared.Subscription, error)
	GetActiveSubscriptionsByFrequency(ctx context.Context, frequency shared.Frequency) ([]shared.Subscription, error)
	FindByID(ctx context.Context, id uint) (*SubscriptionServiceData, error)
	// HTTP operations for API Gateway
	Subscribe(ctx context.Context, email, city, frequency string) error
	ConfirmSubscription(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
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

// UserService defines the contract for user/auth operations used by other bounded contexts
type UserService interface {
	ValidateToken(ctx context.Context, token string) (bool, error)
	GenerateToken(ctx context.Context, userID, email string, ttlSeconds int64) (string, error)
}
