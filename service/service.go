// Package service implements business logic and external API interactions
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/models"
)

// WeatherService handles interactions with the weather API
type WeatherService struct {
	config *config.Config
	client *http.Client
}

// NewWeatherService creates a new weather service instance
func NewWeatherService(config *config.Config) *WeatherService {
	return &WeatherService{
		config: config,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetWeather retrieves current weather information for a specific city
func (s *WeatherService) GetWeather(city string) (*models.WeatherResponse, error) {
	log.Printf("[DEBUG] WeatherService.GetWeather called for city: %s\n", city)

	url := fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no",
		s.config.Weather.BaseURL, s.config.Weather.APIKey, city)

	log.Printf("[DEBUG] Making request to Weather API: %s\n", url)

	resp, err := s.client.Get(url)
	if err != nil {
		log.Printf("[ERROR] Failed to get weather data: %v\n", err)
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}
	// Fix for unchecked error from resp.Body.Close()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	log.Printf("[DEBUG] Weather API response status: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("city not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status code %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode weather data: %v\n", err)
		return nil, fmt.Errorf("failed to decode weather data: %w", err)
	}

	log.Printf("[DEBUG] Weather API raw response: %+v\n", result)

	current, ok := result["current"].(map[string]interface{})
	if !ok {
		log.Printf("[ERROR] Invalid weather data format, 'current' field not found or wrong type\n")
		return nil, fmt.Errorf("invalid weather data format")
	}

	weatherCondition, ok := current["condition"].(map[string]interface{})
	if !ok {
		log.Printf("[ERROR] Invalid weather data format, 'condition' field not found or wrong type\n")
		return nil, fmt.Errorf("invalid weather data format")
	}

	weather := &models.WeatherResponse{
		Temperature: current["temp_c"].(float64),
		Humidity:    current["humidity"].(float64),
		Description: weatherCondition["text"].(string),
	}

	log.Printf("[DEBUG] Parsed weather data: %+v\n", weather)
	return weather, nil
}

// SubscriptionService handles business logic for subscription management
type SubscriptionService struct {
	db               *gorm.DB
	subscriptionRepo SubscriptionRepositoryInterface
	tokenRepo        TokenRepositoryInterface
	emailService     EmailServiceInterface
	weatherService   WeatherServiceInterface
	config           *config.Config
}

// NewSubscriptionService creates a new subscription service instance
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

	existing, err := s.subscriptionRepo.FindByEmail(req.Email, req.City)
	if err != nil {
		log.Printf("[ERROR] Error checking existing subscription: %v\n", err)
		return err
	}

	if existing != nil {
		log.Printf("[DEBUG] Found existing subscription: %+v, confirmed: %v\n",
			existing, existing.Confirmed)

		if existing.Confirmed {
			return fmt.Errorf("email already subscribed")
		}
	}

	// Fix: Split into two separate transactions
	var subscription *models.Subscription

	// First transaction: Create or update subscription
	tx1 := s.db.Begin()
	if tx1.Error != nil {
		log.Printf("[ERROR] Error beginning transaction 1: %v\n", tx1.Error)
		return tx1.Error
	}

	log.Println("[DEBUG] Started DB transaction 1 for subscription")

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Recovered from panic in Subscribe tx1: %v\n", r)
			tx1.Rollback()
		}
	}()

	if existing != nil {
		subscription = existing
		subscription.Frequency = req.Frequency
		log.Printf("[DEBUG] Updating existing subscription to frequency: %s\n", req.Frequency)

		if err := tx1.Save(subscription).Error; err != nil {
			log.Printf("[ERROR] Error saving updated subscription: %v\n", err)
			tx1.Rollback()
			return err
		}
	} else {
		subscription = &models.Subscription{
			Email:     req.Email,
			City:      req.City,
			Frequency: req.Frequency,
			Confirmed: false,
		}
		log.Printf("[DEBUG] Creating new subscription: %+v\n", subscription)

		if err := tx1.Create(subscription).Error; err != nil {
			log.Printf("[ERROR] Error creating new subscription: %v\n", err)
			tx1.Rollback()
			return err
		}
	}

	// Important: Commit first transaction to ensure subscription is saved
	log.Println("[DEBUG] Committing transaction 1")
	if err := tx1.Commit().Error; err != nil {
		log.Printf("[ERROR] Error committing transaction 1: %v\n", err)
		return err
	}

	log.Printf("[DEBUG] Subscription created/updated with ID: %d\n", subscription.ID)

	// Second transaction: Create token for the saved subscription
	tx2 := s.db.Begin()
	if tx2.Error != nil {
		log.Printf("[ERROR] Error beginning transaction 2: %v\n", tx2.Error)
		return tx2.Error
	}

	log.Println("[DEBUG] Started DB transaction 2 for token")

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Recovered from panic in Subscribe tx2: %v\n", r)
			tx2.Rollback()
		}
	}()

	// Fetch the fresh subscription to ensure we have correct ID
	refreshedSubscription := &models.Subscription{}
	if err := s.db.First(refreshedSubscription, subscription.ID).Error; err != nil {
		log.Printf("[ERROR] Error fetching refreshed subscription: %v\n", err)
		tx2.Rollback()
		return err
	}

	log.Printf("[DEBUG] Refreshed subscription: %+v\n", refreshedSubscription)
	log.Printf("[DEBUG] Creating confirmation token for subscription ID: %d\n", refreshedSubscription.ID)

	token, err := s.tokenRepo.CreateToken(refreshedSubscription.ID, "confirmation", 24*time.Hour)
	if err != nil {
		log.Printf("[ERROR] Error creating token: %v\n", err)
		tx2.Rollback()
		return err
	}

	log.Printf("[DEBUG] Created token: %s, expires: %v\n", token.Token, token.ExpiresAt)

	log.Println("[DEBUG] Committing transaction 2")
	if err := tx2.Commit().Error; err != nil {
		log.Printf("[ERROR] Error committing transaction 2: %v\n", err)
		return err
	}

	confirmURL := fmt.Sprintf("%s/api/confirm/%s", s.config.AppBaseURL, token.Token)
	log.Printf("[DEBUG] Would send confirmation email to: %s with URL: %s\n", refreshedSubscription.Email, confirmURL)

	// Attempt to send confirmation email and return error if it fails
	err = s.emailService.SendConfirmationEmail(refreshedSubscription.Email, confirmURL, refreshedSubscription.City)
	if err != nil {
		log.Printf("[ERROR] Failed to send confirmation email: %v\n", err)
		return fmt.Errorf("failed to send confirmation email: %w", err)
	}

	log.Println("[DEBUG] Subscription process completed successfully")
	return nil
}

// ConfirmSubscription validates and confirms a subscription using a token
func (s *SubscriptionService) ConfirmSubscription(tokenStr string) error {
	log.Printf("[DEBUG] ConfirmSubscription called with token: %s\n", tokenStr)

	token, err := s.tokenRepo.FindByToken(tokenStr)
	if err != nil {
		log.Printf("[ERROR] Error finding token: %v\n", err)
		return err
	}

	log.Printf("[DEBUG] Found token: %+v\n", token)

	if token.Type != "confirmation" {
		log.Printf("[ERROR] Invalid token type: %s\n", token.Type)
		return fmt.Errorf("invalid token type")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		log.Printf("[ERROR] Error beginning transaction: %v\n", tx.Error)
		return tx.Error
	}

	log.Println("[DEBUG] Started DB transaction")

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Recovered from panic in ConfirmSubscription: %v\n", r)
			tx.Rollback()
		}
	}()

	subscription, err := s.subscriptionRepo.FindByID(token.SubscriptionID)
	if err != nil {
		log.Printf("[ERROR] Error finding subscription: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Printf("[DEBUG] Found subscription: %+v\n", subscription)

	subscription.Confirmed = true
	log.Println("[DEBUG] Setting subscription to confirmed")

	if err := tx.Save(subscription).Error; err != nil {
		log.Printf("[ERROR] Error saving subscription: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Println("[DEBUG] Deleting confirmation token")
	if err := tx.Delete(token).Error; err != nil {
		log.Printf("[ERROR] Error deleting token: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Println("[DEBUG] Creating unsubscribe token")
	unsubscribeToken, err := s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
	if err != nil {
		log.Printf("[ERROR] Error creating unsubscribe token: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Printf("[DEBUG] Created unsubscribe token: %s\n", unsubscribeToken.Token)

	log.Println("[DEBUG] Committing transaction")
	if err := tx.Commit().Error; err != nil {
		log.Printf("[ERROR] Error committing transaction: %v\n", err)
		return err
	}

	unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, unsubscribeToken.Token)
	log.Printf("[DEBUG] Would send welcome email to: %s with unsubscribe URL: %s\n",
		subscription.Email, unsubscribeURL)

	// Try to send email but don't fail if it doesn't work
	err = s.emailService.SendWelcomeEmail(subscription.Email, subscription.City, subscription.Frequency, unsubscribeURL)
	if err != nil {
		log.Printf("[WARNING] Error sending welcome email, but continuing anyway: %v\n", err)
		// Don't return the error
	}

	log.Println("[DEBUG] Confirmation process completed successfully")
	return nil
}

// Unsubscribe removes a subscription using an unsubscribe token
func (s *SubscriptionService) Unsubscribe(tokenStr string) error {
	log.Printf("[DEBUG] Unsubscribe called with token: %s\n", tokenStr)

	token, err := s.tokenRepo.FindByToken(tokenStr)
	if err != nil {
		log.Printf("[ERROR] Error finding token: %v\n", err)
		return err
	}

	log.Printf("[DEBUG] Found token: %+v\n", token)

	if token.Type != "unsubscribe" {
		log.Printf("[ERROR] Invalid token type: %s\n", token.Type)
		return fmt.Errorf("invalid token type")
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		log.Printf("[ERROR] Error beginning transaction: %v\n", tx.Error)
		return tx.Error
	}

	log.Println("[DEBUG] Started DB transaction")

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Recovered from panic in Unsubscribe: %v\n", r)
			tx.Rollback()
		}
	}()

	subscription, err := s.subscriptionRepo.FindByID(token.SubscriptionID)
	if err != nil {
		log.Printf("[ERROR] Error finding subscription: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Printf("[DEBUG] Found subscription: %+v\n", subscription)

	log.Println("[DEBUG] Deleting subscription")
	if err := tx.Delete(subscription).Error; err != nil {
		log.Printf("[ERROR] Error deleting subscription: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Println("[DEBUG] Deleting token")
	if err := tx.Delete(token).Error; err != nil {
		log.Printf("[ERROR] Error deleting token: %v\n", err)
		tx.Rollback()
		return err
	}

	log.Println("[DEBUG] Committing transaction")
	if err := tx.Commit().Error; err != nil {
		log.Printf("[ERROR] Error committing transaction: %v\n", err)
		return err
	}

	log.Printf("[DEBUG] Would send unsubscribe confirmation email to: %s\n", subscription.Email)
	// Try to send email but don't fail if it doesn't work
	err = s.emailService.SendUnsubscribeConfirmationEmail(subscription.Email, subscription.City)
	if err != nil {
		log.Printf("[WARNING] Error sending unsubscribe confirmation email, but continuing anyway: %v\n", err)
		// Don't return the error
	}

	log.Println("[DEBUG] Unsubscribe process completed successfully")
	return nil
}

// SendWeatherUpdate sends weather updates to all subscribers of the specified frequency
func (s *SubscriptionService) SendWeatherUpdate(frequency string) error {
	log.Printf("[DEBUG] SendWeatherUpdate called for frequency: %s\n", frequency)

	subscriptions, err := s.subscriptionRepo.GetSubscriptionsForUpdates(frequency)
	if err != nil {
		log.Printf("[ERROR] Error getting subscriptions for updates: %v\n", err)
		return err
	}

	log.Printf("[DEBUG] Found %d subscriptions for frequency: %s\n", len(subscriptions), frequency)

	for _, subscription := range subscriptions {
		log.Printf("[DEBUG] Processing subscription: %+v\n", subscription)

		weather, err := s.weatherService.GetWeather(subscription.City)
		if err != nil {
			log.Printf("[ERROR] Error getting weather for %s: %v\n", subscription.City, err)
			continue
		}

		log.Printf("[DEBUG] Got weather data: %+v\n", weather)

		token, err := s.tokenRepo.FindByToken(fmt.Sprintf("%d", subscription.ID))
		if err != nil {
			log.Printf("[DEBUG] No existing token found, creating new one: %v\n", err)
			token, err = s.tokenRepo.CreateToken(subscription.ID, "unsubscribe", 365*24*time.Hour)
			if err != nil {
				log.Printf("[ERROR] Error creating unsubscribe token for subscription %d: %v\n", subscription.ID, err)
				continue
			}
			log.Printf("[DEBUG] Created new token: %s\n", token.Token)
		}

		unsubscribeURL := fmt.Sprintf("%s/api/unsubscribe/%s", s.config.AppBaseURL, token.Token)
		log.Printf("[DEBUG] Would send weather update to: %s with unsubscribe URL: %s\n",
			subscription.Email, unsubscribeURL)

		// Try to send email but don't fail if it doesn't work
		err = s.emailService.SendWeatherUpdateEmail(
			subscription.Email,
			subscription.City,
			weather,
			unsubscribeURL,
		)
		if err != nil {
			log.Printf("[WARNING] Error sending weather update email, but continuing anyway: %v\n", err)
			continue
		}

		log.Printf("[DEBUG] Successfully sent weather update to: %s\n", subscription.Email)
	}

	log.Println("[DEBUG] SendWeatherUpdate completed")
	return nil
}
