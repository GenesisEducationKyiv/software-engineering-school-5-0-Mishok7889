package repository

import (
	"log/slog"
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
	slog.Debug("Finding subscription by email and city", "email", email, "city", city)

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
			slog.Debug("No subscription found", "email", email, "city", city)
			return nil, nil
		}
		slog.Error("Database error when finding subscription", "error", result.Error, "email", email, "city", city)
		return nil, errors.NewDatabaseError("failed to find subscription", result.Error)
	}

	slog.Debug("Found subscription", "id", subscription.ID, "email", email, "city", city)
	return &subscription, nil
}

// FindByID retrieves a subscription by its ID
func (r *SubscriptionRepository) FindByID(id uint) (*models.Subscription, error) {
	slog.Debug("Finding subscription by ID", "id", id)

	if id == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}

	var subscription models.Subscription
	result := r.db.First(&subscription, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("subscription not found")
		}
		slog.Error("Database error when finding subscription by ID", "error", result.Error, "id", id)
		return nil, errors.NewDatabaseError("failed to find subscription by ID", result.Error)
	}

	slog.Debug("Found subscription by ID", "id", subscription.ID, "email", subscription.Email, "city", subscription.City)
	return &subscription, nil
}

// Create persists a new subscription to the database
func (r *SubscriptionRepository) Create(subscription *models.Subscription) error {
	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	slog.Debug("Creating subscription", "email", subscription.Email, "city", subscription.City)

	result := r.db.Create(subscription)
	if result.Error != nil {
		slog.Error("Database error when creating subscription", "error", result.Error, "email", subscription.Email)
		return errors.NewDatabaseError("failed to create subscription", result.Error)
	}

	slog.Debug("Created subscription", "id", subscription.ID, "email", subscription.Email, "city", subscription.City)
	return nil
}

// Update modifies an existing subscription
func (r *SubscriptionRepository) Update(subscription *models.Subscription) error {
	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	slog.Debug("Updating subscription", "id", subscription.ID, "email", subscription.Email, "city", subscription.City)

	result := r.db.Save(subscription)
	if result.Error != nil {
		slog.Error("Database error when updating subscription", "error", result.Error, "id", subscription.ID)
		return errors.NewDatabaseError("failed to update subscription", result.Error)
	}

	slog.Debug("Updated subscription successfully", "id", subscription.ID)
	return nil
}

// Delete removes a subscription from the database
func (r *SubscriptionRepository) Delete(subscription *models.Subscription) error {
	if subscription == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}

	slog.Debug("Deleting subscription", "id", subscription.ID, "email", subscription.Email, "city", subscription.City)

	result := r.db.Delete(subscription)
	if result.Error != nil {
		slog.Error("Database error when deleting subscription", "error", result.Error, "id", subscription.ID)
		return errors.NewDatabaseError("failed to delete subscription", result.Error)
	}

	slog.Debug("Deleted subscription successfully", "id", subscription.ID)
	return nil
}

// GetSubscriptionsForUpdates retrieves all confirmed subscriptions for a specific frequency
func (r *SubscriptionRepository) GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error) {
	slog.Debug("Getting subscriptions for updates", "frequency", frequency)

	if frequency == "" {
		return nil, errors.NewValidationError("frequency cannot be empty")
	}

	var subscriptions []models.Subscription
	result := r.db.Where("frequency = ? AND confirmed = ?", frequency, true).Find(&subscriptions)
	if result.Error != nil {
		slog.Error("Database error when getting subscriptions for updates", "error", result.Error, "frequency", frequency)
		return nil, errors.NewDatabaseError("failed to get subscriptions for updates", result.Error)
	}

	slog.Debug("Found subscriptions for updates", "count", len(subscriptions), "frequency", frequency)
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
	slog.Debug("Creating token", "subscriptionID", subscriptionID, "type", tokenType, "expiresIn", expiresIn)

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
		slog.Error("Database error when creating token", "error", result.Error, "subscriptionID", subscriptionID)
		return nil, errors.NewDatabaseError("failed to create token", result.Error)
	}

	slog.Debug("Created token", "tokenID", token.ID, "type", token.Type, "subscriptionID", subscriptionID)
	return token, nil
}

// FindByToken retrieves a token by its string value
func (r *TokenRepository) FindByToken(tokenStr string) (*models.Token, error) {
	slog.Debug("Finding token by value", "token", tokenStr)

	if tokenStr == "" {
		return nil, errors.NewValidationError("token cannot be empty")
	}

	var token models.Token
	result := r.db.Where("token = ? AND expires_at > ?", tokenStr, time.Now()).First(&token)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("token not found or expired")
		}
		slog.Error("Database error when finding token", "error", result.Error, "token", tokenStr)
		return nil, errors.NewDatabaseError("failed to find token", result.Error)
	}

	slog.Debug("Found token", "tokenID", token.ID, "type", token.Type, "subscriptionID", token.SubscriptionID)
	return &token, nil
}

// DeleteToken removes a token from the database
func (r *TokenRepository) DeleteToken(token *models.Token) error {
	if token == nil {
		return errors.NewValidationError("token cannot be nil")
	}

	slog.Debug("Deleting token", "tokenID", token.ID, "type", token.Type, "subscriptionID", token.SubscriptionID)

	result := r.db.Delete(token)
	if result.Error != nil {
		slog.Error("Database error when deleting token", "error", result.Error, "tokenID", token.ID)
		return errors.NewDatabaseError("failed to delete token", result.Error)
	}

	slog.Debug("Deleted token successfully", "tokenID", token.ID)
	return nil
}

// DeleteExpiredTokens removes all expired tokens from the database
func (r *TokenRepository) DeleteExpiredTokens() error {
	slog.Debug("Deleting expired tokens")

	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.Token{})
	if result.Error != nil {
		slog.Error("Database error when deleting expired tokens", "error", result.Error)
		return errors.NewDatabaseError("failed to delete expired tokens", result.Error)
	}

	slog.Debug("Deleted expired tokens", "count", result.RowsAffected)
	return nil
}
