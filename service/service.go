package service

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/errors"
	"weatherapi.app/models"
	"weatherapi.app/providers"
)

// WeatherService handles weather-related operations
type WeatherService struct {
	provider providers.WeatherProvider
}

// NewWeatherService creates a new weather service with the specified provider
func NewWeatherService(provider providers.WeatherProvider) *WeatherService {
	return &WeatherService{
		provider: provider,
	}
}

// GetWeather retrieves current weather information for a specific city
func (s *WeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	log.Printf("[DEBUG] WeatherService.GetWeather called for city: %s\n", city)

	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	weather, err := s.provider.GetCurrentWeather(city)
	if err != nil {
		log.Printf("[ERROR] Weather provider error: %v\n", err)
		return nil, err
	}

	log.Printf("[DEBUG] Weather data retrieved: %+v\n", weather)
	return weather, nil
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
	log.Printf("[DEBUG] SubscriptionService.Subscribe called with: %+v\n", req)

	if err := s.validateSubscriptionRequest(req); err != nil {
		return err
	}

	existing, err := s.subscriptionRepo.FindByEmail(req.Email, req.City)
	if err != nil {
		return errors.NewDatabaseError("failed to check existing subscription", err)
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
		return nil, errors.NewDatabaseError("failed to begin transaction", tx.Error)
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
			return nil, errors.NewDatabaseError("failed to update subscription", err)
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
			return nil, errors.NewDatabaseError("failed to create subscription", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.NewDatabaseError("failed to commit transaction", err)
	}

	return subscription, nil
}

func (s *SubscriptionService) sendConfirmationEmail(subscription *models.Subscription) error {
	token, err := s.tokenRepo.CreateToken(subscription.ID, "confirmation", 24*time.Hour)
	if err != nil {
		return errors.NewDatabaseError("failed to create confirmation token", err)
	}

	confirmURL := fmt.Sprintf("%s/api/confirm/%s", s.config.AppBaseURL, token.Token)

	if err := s.emailService.SendConfirmationEmail(subscription.Email, confirmURL, subscription.City); err != nil {
		return err
	}

	return nil
}

// ConfirmSubscription validates and confirms a subscription using a token
func (s *SubscriptionService) ConfirmSubscription(tokenStr string) error {
	log.Printf("[DEBUG] ConfirmSubscription called with token: %s\n", tokenStr)

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
		return errors.NewDatabaseError("failed to find subscription", err)
	}

	return s.processConfirmation(subscription, token)
}

func (s *SubscriptionService) processConfirmation(subscription *models.Subscription, token *models.Token) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.NewDatabaseError("failed to begin transaction", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	subscription.Confirmed = true
	if err := tx.Save(subscription).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("failed to update subscription", err)
	}

	if err := tx.Delete(token).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("failed to delete token", err)
	}

	unsubscribeToken, err := s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
	if err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("failed to create unsubscribe token", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("failed to commit transaction", err)
	}

	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, unsubscribeToken.Token)

	// Try to send welcome email but don't fail if it doesn't work
	if err := s.emailService.SendWelcomeEmail(subscription.Email, subscription.City, subscription.Frequency, unsubscribeURL); err != nil {
		log.Printf("[WARNING] Failed to send welcome email: %v\n", err)
	}

	return nil
}

// Unsubscribe removes a subscription using an unsubscribe token
func (s *SubscriptionService) Unsubscribe(tokenStr string) error {
	log.Printf("[DEBUG] Unsubscribe called with token: %s\n", tokenStr)

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
		return errors.NewDatabaseError("failed to find subscription", err)
	}

	return s.processUnsubscription(subscription, token)
}

func (s *SubscriptionService) processUnsubscription(subscription *models.Subscription, token *models.Token) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return errors.NewDatabaseError("failed to begin transaction", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Delete(subscription).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("failed to delete subscription", err)
	}

	if err := tx.Delete(token).Error; err != nil {
		tx.Rollback()
		return errors.NewDatabaseError("failed to delete token", err)
	}

	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("failed to commit transaction", err)
	}

	// Try to send confirmation email but don't fail if it doesn't work
	if err := s.emailService.SendUnsubscribeConfirmationEmail(subscription.Email, subscription.City); err != nil {
		log.Printf("[WARNING] Failed to send unsubscribe confirmation email: %v\n", err)
	}

	return nil
}

// SendWeatherUpdate sends weather updates to all subscribers of the specified frequency
func (s *SubscriptionService) SendWeatherUpdate(frequency string) error {
	log.Printf("[DEBUG] SendWeatherUpdate called for frequency: %s\n", frequency)

	if frequency != "hourly" && frequency != "daily" {
		return errors.NewValidationError("frequency must be either 'hourly' or 'daily'")
	}

	subscriptions, err := s.subscriptionRepo.GetSubscriptionsForUpdates(frequency)
	if err != nil {
		return errors.NewDatabaseError("failed to get subscriptions for updates", err)
	}

	log.Printf("[DEBUG] Found %d subscriptions for frequency: %s\n", len(subscriptions), frequency)

	for _, subscription := range subscriptions {
		if err := s.sendWeatherUpdateToSubscriber(subscription); err != nil {
			log.Printf("[WARNING] Failed to send weather update to %s: %v\n", subscription.Email, err)
			continue
		}
	}

	return nil
}

func (s *SubscriptionService) sendWeatherUpdateToSubscriber(subscription models.Subscription) error {
	weather, err := s.weatherService.GetWeather(subscription.City)
	if err != nil {
		return fmt.Errorf("failed to get weather for %s: %w", subscription.City, err)
	}

	token, err := s.tokenRepo.FindByToken(fmt.Sprintf("%d", subscription.ID))
	if err != nil {
		token, err = s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
		if err != nil {
			return fmt.Errorf("failed to create unsubscribe token: %w", err)
		}
	}

	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, token.Token)

	return s.emailService.SendWeatherUpdateEmail(subscription.Email, subscription.City, weather, unsubscribeURL)
}
