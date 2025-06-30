package service

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/errors"
	"weatherapi.app/models"
)

// WeatherService handles weather-related operations using provider manager
// Follows Facade pattern: simple interface to complex provider chain
type WeatherService struct {
	providerManager WeatherProviderManagerInterface
}

// NewWeatherService creates a new weather service with the specified provider manager
func NewWeatherService(providerManager WeatherProviderManagerInterface) *WeatherService {
	return &WeatherService{
		providerManager: providerManager,
	}
}

// GetWeather retrieves current weather information for a specific city
// Uses chain of responsibility with caching and logging
func (s *WeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	slog.Debug("Getting weather", "city", city)

	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	weather, err := s.providerManager.GetWeather(city)
	if err != nil {
		slog.Error("Weather provider error", "error", err, "city", city)
		return nil, err
	}

	slog.Debug("Weather data retrieved", "city", city, "temp", weather.Temperature, "description", weather.Description)
	return weather, nil
}

// GetProviderInfo returns information about configured providers
func (s *WeatherService) GetProviderInfo() map[string]interface{} {
	return s.providerManager.GetProviderInfo()
}

// GetCacheMetrics returns cache statistics from the provider manager
func (s *WeatherService) GetCacheMetrics() map[string]interface{} {
	return s.providerManager.GetCacheMetrics()
}

// SubscriptionService handles subscription-related business logic
type SubscriptionService struct {
	db               *gorm.DB
	subscriptionRepo SubscriptionRepositoryInterface
	tokenRepo        TokenRepositoryInterface
	emailService     EmailServiceInterface
	weatherService   WeatherServiceInterface
	config           *config.Config
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(
	db *gorm.DB,
	subscriptionRepo SubscriptionRepositoryInterface,
	tokenRepo TokenRepositoryInterface,
	emailService EmailServiceInterface,
	weatherService WeatherServiceInterface,
	config *config.Config,
) *SubscriptionService {
	return &SubscriptionService{
		db:               db,
		subscriptionRepo: subscriptionRepo,
		tokenRepo:        tokenRepo,
		emailService:     emailService,
		weatherService:   weatherService,
		config:           config,
	}
}

// Subscribe creates a new weather subscription or updates an existing one
func (s *SubscriptionService) Subscribe(req *models.SubscriptionRequest) error {
	slog.Debug("Processing subscription", "email", req.Email, "city", req.City, "frequency", req.Frequency)

	if err := s.validateSubscriptionRequest(req); err != nil {
		return err
	}

	existing, err := s.subscriptionRepo.FindByEmail(req.Email, req.City)
	if err != nil {
		return errors.NewDatabaseError("check existing subscription", err)
	}

	if existing != nil && existing.Confirmed {
		return errors.NewAlreadyExistsError("email already subscribed")
	}

	subscription, err := s.createOrUpdateSubscription(existing, req)
	if err != nil {
		return err
	}

	return s.sendConfirmationEmail(subscription)
}

func (s *SubscriptionService) validateSubscriptionRequest(req *models.SubscriptionRequest) error {
	if req.Email == "" {
		return errors.NewValidationError("email is required")
	}
	if req.City == "" {
		return errors.NewValidationError("city is required")
	}
	if req.Frequency != "hourly" && req.Frequency != "daily" {
		return errors.NewValidationError("frequency must be either 'hourly' or 'daily'")
	}
	return nil
}

func (s *SubscriptionService) createOrUpdateSubscription(existing *models.Subscription, req *models.SubscriptionRequest) (*models.Subscription, error) {
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, errors.NewDatabaseError("begin transaction", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var subscription *models.Subscription
	if existing != nil {
		subscription = existing
		subscription.Frequency = req.Frequency
		if err := tx.Save(subscription).Error; err != nil {
			tx.Rollback()
			return nil, errors.NewDatabaseError("update subscription", err)
		}
	} else {
		subscription = &models.Subscription{
			Email:     req.Email,
			City:      req.City,
			Frequency: req.Frequency,
			Confirmed: false,
		}
		if err := tx.Create(subscription).Error; err != nil {
			tx.Rollback()
			return nil, errors.NewDatabaseError("create subscription", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.NewDatabaseError("commit transaction", err)
	}

	return subscription, nil
}

func (s *SubscriptionService) sendConfirmationEmail(subscription *models.Subscription) error {
	token, err := s.tokenRepo.CreateToken(subscription.ID, "confirmation", 24*time.Hour)
	if err != nil {
		return errors.NewDatabaseError("create confirmation token", err)
	}

	confirmURL := fmt.Sprintf("%s/api/confirm/%s", s.config.AppBaseURL, token.Token)

	params := ConfirmationEmailParams{
		Email:      subscription.Email,
		ConfirmURL: confirmURL,
		City:       subscription.City,
	}

	if err := s.emailService.SendConfirmationEmailWithParams(params); err != nil {
		return err
	}

	return nil
}

// ConfirmSubscription validates and confirms a subscription using a token
func (s *SubscriptionService) ConfirmSubscription(tokenStr string) error {
	slog.Debug("Confirming subscription", "token", tokenStr)

	if tokenStr == "" {
		return errors.NewValidationError("token cannot be empty")
	}

	token, err := s.tokenRepo.FindByToken(tokenStr)
	if err != nil {
		return errors.NewTokenError("token not found or expired")
	}

	if token.Type != "confirmation" {
		return errors.NewTokenError("invalid token type")
	}

	subscription, err := s.subscriptionRepo.FindByID(token.SubscriptionID)
	if err != nil {
		return err
	}

	return s.processConfirmation(subscription, token)
}

func (s *SubscriptionService) processConfirmation(subscription *models.Subscription, token *models.Token) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.NewDatabaseError("begin transaction", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	subscription.Confirmed = true
	if err := tx.Save(subscription).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("update subscription", err)
	}

	if err := tx.Delete(token).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("delete token", err)
	}

	unsubscribeToken, err := s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
	if err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("create unsubscribe token", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("commit transaction", err)
	}

	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, unsubscribeToken.Token)

	// Try to send welcome email but don't fail if it doesn't work
	params := WelcomeEmailParams{
		Email:          subscription.Email,
		City:           subscription.City,
		Frequency:      subscription.Frequency,
		UnsubscribeURL: unsubscribeURL,
	}

	if err := s.emailService.SendWelcomeEmailWithParams(params); err != nil {
		slog.Warn("send welcome email", "error", err, "email", subscription.Email)
	}

	return nil
}

// Unsubscribe removes a subscription using an unsubscribe token
func (s *SubscriptionService) Unsubscribe(tokenStr string) error {
	slog.Debug("Processing unsubscribe", "token", tokenStr)

	if tokenStr == "" {
		return errors.NewValidationError("token cannot be empty")
	}

	token, err := s.tokenRepo.FindByToken(tokenStr)
	if err != nil {
		return errors.NewTokenError("token not found or expired")
	}

	if token.Type != "unsubscribe" {
		return errors.NewTokenError("invalid token type")
	}

	subscription, err := s.subscriptionRepo.FindByID(token.SubscriptionID)
	if err != nil {
		return err
	}

	return s.processUnsubscription(subscription, token)
}

func (s *SubscriptionService) processUnsubscription(subscription *models.Subscription, token *models.Token) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.NewDatabaseError("begin transaction", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Delete(subscription).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("delete subscription", err)
	}

	if err := tx.Delete(token).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("delete token", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("commit transaction", err)
	}

	// Try to send confirmation email but don't fail if it doesn't work
	params := UnsubscribeEmailParams{
		Email: subscription.Email,
		City:  subscription.City,
	}

	if err := s.emailService.SendUnsubscribeConfirmationEmailWithParams(params); err != nil {
		slog.Warn("send unsubscribe confirmation email", "error", err, "email", subscription.Email)
	}

	return nil
}

// SendWeatherUpdate sends weather updates to all subscribers of the specified frequency
func (s *SubscriptionService) SendWeatherUpdate(frequency string) error {
	slog.Debug("Sending weather updates", "frequency", frequency)

	if frequency != "hourly" && frequency != "daily" {
		return errors.NewValidationError("frequency must be either 'hourly' or 'daily'")
	}

	subscriptions, err := s.subscriptionRepo.GetSubscriptionsForUpdates(frequency)
	if err != nil {
		return errors.NewDatabaseError("get subscriptions for updates", err)
	}

	slog.Debug("Found subscriptions for updates", "count", len(subscriptions), "frequency", frequency)

	for _, subscription := range subscriptions {
		if err := s.sendWeatherUpdateToSubscriber(subscription); err != nil {
			slog.Warn("send weather update", "error", err, "email", subscription.Email, "city", subscription.City)
			continue
		}
	}

	return nil
}

func (s *SubscriptionService) sendWeatherUpdateToSubscriber(subscription models.Subscription) error {
	slog.Debug("Sending weather update to subscriber", "email", subscription.Email, "city", subscription.City)

	weather, err := s.weatherService.GetWeather(subscription.City)
	if err != nil {
		slog.Error("get weather", "error", err, "city", subscription.City)
		return fmt.Errorf("get weather for %s: %w", subscription.City, err)
	}
	slog.Debug("Retrieved weather data", "weather", weather, "city", subscription.City)

	// Try to find existing unsubscribe token
	token, err := s.tokenRepo.FindBySubscriptionIDAndType(subscription.ID, "unsubscribe")
	if err != nil {
		slog.Debug("No existing unsubscribe token found, creating new one", "subscriptionID", subscription.ID)
		// If no existing token found, create a new one
		token, err = s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
		if err != nil {
			slog.Error("create unsubscribe token", "error", err, "subscriptionID", subscription.ID)
			return fmt.Errorf("create unsubscribe token: %w", err)
		}
	} else {
		slog.Debug("Found existing unsubscribe token", "token", token.Token)
	}

	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, token.Token)
	slog.Debug("Sending weather update email", "email", subscription.Email, "unsubscribeURL", unsubscribeURL)

	params := WeatherUpdateEmailParams{
		Email:          subscription.Email,
		City:           subscription.City,
		Weather:        weather,
		UnsubscribeURL: unsubscribeURL,
	}

	return s.emailService.SendWeatherUpdateEmailWithParams(params)
}
