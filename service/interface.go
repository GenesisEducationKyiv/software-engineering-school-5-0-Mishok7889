package service

import (
	"time"

	"weatherapi.app/models"
	"weatherapi.app/providers"
)

// WeatherProviderManagerInterface is an alias to the providers interface
type WeatherProviderManagerInterface = providers.ProviderManagerInterface

// WeatherServiceInterface defines the interface for weather operations
type WeatherServiceInterface interface {
	GetWeather(city string) (*models.WeatherResponse, error)
	GetProviderInfo() map[string]interface{}
	GetCacheMetrics() map[string]interface{}
}

// SubscriptionManagerInterface handles subscription creation and removal
type SubscriptionManagerInterface interface {
	Subscribe(req *models.SubscriptionRequest) error
	Unsubscribe(token string) error
}

// ConfirmationServiceInterface handles subscription confirmations
type ConfirmationServiceInterface interface {
	ConfirmSubscription(token string) error
}

// NotificationServiceInterface handles sending notifications
type NotificationServiceInterface interface {
	SendWeatherUpdate(frequency string) error
}

// Combined interface for backward compatibility
type SubscriptionServiceInterface interface {
	SubscriptionManagerInterface
	ConfirmationServiceInterface
	NotificationServiceInterface
}

// EmailServiceInterface defines the interface for email operations
type EmailServiceInterface interface {
	SendConfirmationEmailWithParams(params ConfirmationEmailParams) error
	SendWelcomeEmailWithParams(params WelcomeEmailParams) error
	SendUnsubscribeConfirmationEmailWithParams(params UnsubscribeEmailParams) error
	SendWeatherUpdateEmailWithParams(params WeatherUpdateEmailParams) error
}

// SubscriptionRepositoryInterface defines the interface for subscription data operations
type SubscriptionRepositoryInterface interface {
	FindByEmail(email, city string) (*models.Subscription, error)
	FindByID(id uint) (*models.Subscription, error)
	Create(subscription *models.Subscription) error
	Update(subscription *models.Subscription) error
	Delete(subscription *models.Subscription) error
	GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error)
}

// TokenRepositoryInterface defines the interface for token operations
type TokenRepositoryInterface interface {
	CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error)
	FindByToken(tokenStr string) (*models.Token, error)
	FindBySubscriptionIDAndType(subscriptionID uint, tokenType string) (*models.Token, error)
	DeleteToken(token *models.Token) error
	DeleteExpiredTokens() error
}

// Ensure implementations satisfy interfaces
var _ WeatherServiceInterface = (*WeatherService)(nil)
var _ SubscriptionServiceInterface = (*SubscriptionService)(nil)
var _ EmailServiceInterface = (*EmailService)(nil)
