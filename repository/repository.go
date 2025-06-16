package repository

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"weatherapi.app/errors"
	"weatherapi.app/models"
)

// SubscriptionRepository handles data access operations for subscriptions
type SubscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository creates a new repository for subscription data
func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// FindByEmail retrieves a subscription by email and city
func (r *SubscriptionRepository) FindByEmail(email, city string) (*models.Subscription, error) {
	log.Printf("[DEBUG] SubscriptionRepository.FindByEmail: email=%s, city=%s\n", email, city)

	if email == "" {
		return nil, errors.NewValidationError("email cannot be empty")
	}
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	var subscription models.Subscription
	result := r.db.Where("email = ? AND city = ?", email, city).First(&subscription)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Println("[DEBUG] No subscription found")
			return nil, nil
		}
		log.Printf("[ERROR] Database error when finding subscription: %v\n", result.Error)
		return nil, errors.NewDatabaseError("failed to find subscription", result.Error)
	}

	log.Printf("[DEBUG] Found subscription: %+v\n", subscription)
	return &subscription, nil
}

// FindByID retrieves a subscription by its ID
func (r *SubscriptionRepository) FindByID(id uint) (*models.Subscription, error) {
	log.Printf("[DEBUG] SubscriptionRepository.FindByID: id=%d\n", id)

	if id == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}

	var subscription models.Subscription
	result := r.db.First(&subscription, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("subscription not found")
		}
		log.Printf("[ERROR] Database error when finding subscription by ID: %v\n", result.Error)
		return nil, errors.NewDatabaseError("failed to find subscription by ID", result.Error)
	}

	log.Printf("[DEBUG] Found subscription: %+v\n", subscription)
	return &subscription, nil
}

// Create persists a new subscription to the database
func (r *SubscriptionRepository) Create(subscription *models.Subscription) error {
	log.Printf("[DEBUG] SubscriptionRepository.Create: %+v\n", subscription)

	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	result := r.db.Create(subscription)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when creating subscription: %v\n", result.Error)
		return errors.NewDatabaseError("failed to create subscription", result.Error)
	}

	log.Printf("[DEBUG] Created subscription with ID: %d\n", subscription.ID)
	return nil
}

// Update modifies an existing subscription
func (r *SubscriptionRepository) Update(subscription *models.Subscription) error {
	log.Printf("[DEBUG] SubscriptionRepository.Update: %+v\n", subscription)

	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	result := r.db.Save(subscription)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when updating subscription: %v\n", result.Error)
		return errors.NewDatabaseError("failed to update subscription", result.Error)
	}

	log.Println("[DEBUG] Updated subscription successfully")
	return nil
}

// Delete removes a subscription from the database
func (r *SubscriptionRepository) Delete(subscription *models.Subscription) error {
	log.Printf("[DEBUG] SubscriptionRepository.Delete: %+v\n", subscription)

	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	result := r.db.Delete(subscription)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when deleting subscription: %v\n", result.Error)
		return errors.NewDatabaseError("failed to delete subscription", result.Error)
	}

	log.Println("[DEBUG] Deleted subscription successfully")
	return nil
}

// GetSubscriptionsForUpdates retrieves all confirmed subscriptions for a specific frequency
func (r *SubscriptionRepository) GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error) {
	log.Printf("[DEBUG] SubscriptionRepository.GetSubscriptionsForUpdates: frequency=%s\n", frequency)

	if frequency == "" {
		return nil, errors.NewValidationError("frequency cannot be empty")
	}

	var subscriptions []models.Subscription
	result := r.db.Where("frequency = ? AND confirmed = ?", frequency, true).Find(&subscriptions)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when getting subscriptions for updates: %v\n", result.Error)
		return nil, errors.NewDatabaseError("failed to get subscriptions for updates", result.Error)
	}

	log.Printf("[DEBUG] Found %d subscriptions for frequency: %s\n", len(subscriptions), frequency)
	return subscriptions, nil
}

// TokenRepository handles data access operations for authentication tokens
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new repository for token operations
func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// CreateToken generates and stores a new token for a subscription
func (r *TokenRepository) CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error) {
	log.Printf("[DEBUG] TokenRepository.CreateToken: subscriptionID=%d, type=%s, expiresIn=%v\n",
		subscriptionID, tokenType, expiresIn)

	if subscriptionID == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}
	if tokenType == "" {
		return nil, errors.NewValidationError("token type cannot be empty")
	}
	if expiresIn <= 0 {
		return nil, errors.NewValidationError("expiration duration must be positive")
	}

	token := &models.Token{
		Token:          uuid.New().String(),
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
	}

	result := r.db.Create(token)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when creating token: %v\n", result.Error)
		return nil, errors.NewDatabaseError("failed to create token", result.Error)
	}

	log.Printf("[DEBUG] Created token: %s, ID: %d\n", token.Token, token.ID)
	return token, nil
}

// FindByToken retrieves a token by its string value
func (r *TokenRepository) FindByToken(tokenStr string) (*models.Token, error) {
	log.Printf("[DEBUG] TokenRepository.FindByToken: token=%s\n", tokenStr)

	if tokenStr == "" {
		return nil, errors.NewValidationError("token cannot be empty")
	}

	var token models.Token
	result := r.db.Where("token = ? AND expires_at > ?", tokenStr, time.Now()).First(&token)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("token not found or expired")
		}
		log.Printf("[ERROR] Database error when finding token: %v\n", result.Error)
		return nil, errors.NewDatabaseError("failed to find token", result.Error)
	}

	log.Printf("[DEBUG] Found token: %+v\n", token)
	return &token, nil
}

// DeleteToken removes a token from the database
func (r *TokenRepository) DeleteToken(token *models.Token) error {
	log.Printf("[DEBUG] TokenRepository.DeleteToken: %+v\n", token)

	if token == nil {
		return errors.NewValidationError("token cannot be nil")
	}

	result := r.db.Delete(token)
	if result.Error != nil {
		log.Printf("[ERROR] Database error when deleting token: %v\n", result.Error)
		return errors.NewDatabaseError("failed to delete token", result.Error)
	}

	log.Println("[DEBUG] Deleted token successfully")
	return nil
}

// DeleteExpiredTokens removes all expired tokens from the database
func (r *TokenRepository) DeleteExpiredTokens() error {
	log.Println("[DEBUG] TokenRepository.DeleteExpiredTokens called")

	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.Token{})
	if result.Error != nil {
		log.Printf("[ERROR] Database error when deleting expired tokens: %v\n", result.Error)
		return errors.NewDatabaseError("failed to delete expired tokens", result.Error)
	}

	log.Printf("[DEBUG] Deleted %d expired tokens\n", result.RowsAffected)
	return nil
}
