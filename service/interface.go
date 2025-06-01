package service

import (
	"time"

	"weatherapi.app/models"
)

// WeatherServiceInterface defines the interface for the weather service
type WeatherServiceInterface interface {
	GetWeather(city string) (*models.WeatherResponse, error)
}

// Ensure WeatherService implements WeatherServiceInterface
var _ WeatherServiceInterface = (*WeatherService)(nil)

// SubscriptionServiceInterface defines the interface for the subscription service
type SubscriptionServiceInterface interface {
	Subscribe(req *models.SubscriptionRequest) error
	ConfirmSubscription(token string) error
	Unsubscribe(token string) error
	SendWeatherUpdate(frequency string) error
}

// Ensure SubscriptionService implements SubscriptionServiceInterface
var _ SubscriptionServiceInterface = (*SubscriptionService)(nil)

// EmailServiceInterface defines the interface for email service
type EmailServiceInterface interface {
	SendConfirmationEmail(email, confirmURL, city string) error
	SendWelcomeEmail(email, city, frequency, unsubscribeURL string) error
	SendUnsubscribeConfirmationEmail(email, city string) error
	SendWeatherUpdateEmail(email, city string, weather *models.WeatherResponse, unsubscribeURL string) error
}

// Ensure EmailService implements EmailServiceInterface
var _ EmailServiceInterface = (*EmailService)(nil)

// SubscriptionRepositoryInterface defines the interface for subscription repository
type SubscriptionRepositoryInterface interface {
	FindByEmail(email, city string) (*models.Subscription, error)
	FindByID(id uint) (*models.Subscription, error)
	Create(subscription *models.Subscription) error
	Update(subscription *models.Subscription) error
	Delete(subscription *models.Subscription) error
	GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error)
}

// Ensure repository.SubscriptionRepository implements SubscriptionRepositoryInterface

// TokenRepositoryInterface defines the interface for token repository
type TokenRepositoryInterface interface {
	CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error)
	FindByToken(tokenStr string) (*models.Token, error)
	DeleteToken(token *models.Token) error
	DeleteExpiredTokens() error
}
