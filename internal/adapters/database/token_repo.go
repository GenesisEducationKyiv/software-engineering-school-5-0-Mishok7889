package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// TokenModel represents the database model for tokens
type TokenModel struct {
	ID             uint      `gorm:"primaryKey"`
	Token          string    `gorm:"uniqueIndex;not null"`
	SubscriptionID uint      `gorm:"index;not null"`
	Type           string    `gorm:"not null"`
	ExpiresAt      time.Time `gorm:"not null"`
	CreatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (TokenModel) TableName() string {
	return "tokens"
}

// TokenRepositoryAdapter implements the TokenRepository port using GORM
type TokenRepositoryAdapter struct {
	db *gorm.DB
}

// NewTokenRepositoryAdapter creates a new token repository adapter
func NewTokenRepositoryAdapter(db *gorm.DB) ports.TokenRepository {
	return &TokenRepositoryAdapter{db: db}
}

// Save persists a token to the database
func (r *TokenRepositoryAdapter) Save(ctx context.Context, token *ports.TokenData) error {
	if token == nil {
		return errors.NewValidationError("token cannot be nil")
	}

	model := r.dataToModel(token)
	var result *gorm.DB

	if token.ID == 0 {
		result = r.db.WithContext(ctx).Create(model)
		token.ID = model.ID
	} else {
		result = r.db.WithContext(ctx).Save(model)
	}

	if result.Error != nil {
		return errors.NewDatabaseError("failed to save token", result.Error)
	}

	return nil
}

// FindByToken retrieves a token by its string value
func (r *TokenRepositoryAdapter) FindByToken(ctx context.Context, tokenStr string) (*ports.TokenData, error) {
	if tokenStr == "" {
		return nil, errors.NewValidationError("token cannot be empty")
	}

	var model TokenModel
	result := r.db.WithContext(ctx).Where("token = ? AND expires_at > ?", tokenStr, time.Now()).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("token not found or expired")
		}
		return nil, errors.NewDatabaseError("failed to find token", result.Error)
	}

	return r.modelToData(&model), nil
}

// FindBySubscriptionIDAndType retrieves a token by subscription ID and type
func (r *TokenRepositoryAdapter) FindBySubscriptionIDAndType(ctx context.Context, subscriptionID uint, tokenType string) (*ports.TokenData, error) {
	if subscriptionID == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}
	if tokenType == "" {
		return nil, errors.NewValidationError("token type cannot be empty")
	}

	var model TokenModel
	result := r.db.WithContext(ctx).Where("subscription_id = ? AND type = ? AND expires_at > ?",
		subscriptionID, tokenType, time.Now()).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("token not found or expired")
		}
		return nil, errors.NewDatabaseError("failed to find token", result.Error)
	}

	return r.modelToData(&model), nil
}

// Delete removes a token from the database
func (r *TokenRepositoryAdapter) Delete(ctx context.Context, token *ports.TokenData) error {
	if token == nil {
		return errors.NewValidationError("token cannot be nil")
	}
	if token.ID == 0 {
		return errors.NewValidationError("token ID cannot be zero for delete")
	}

	result := r.db.WithContext(ctx).Delete(&TokenModel{}, token.ID)
	if result.Error != nil {
		return errors.NewDatabaseError("failed to delete token", result.Error)
	}

	return nil
}

// DeleteExpiredTokens removes all expired tokens from the database
func (r *TokenRepositoryAdapter) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&TokenModel{})
	if result.Error != nil {
		return 0, errors.NewDatabaseError("failed to delete expired tokens", result.Error)
	}

	return result.RowsAffected, nil
}

// CreateConfirmationToken creates a new confirmation token
func (r *TokenRepositoryAdapter) CreateConfirmationToken(ctx context.Context, subscriptionID uint, expiresIn time.Duration) (*ports.TokenData, error) {
	if subscriptionID == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}

	token := &ports.TokenData{
		Value:          uuid.New().String(),
		SubscriptionID: subscriptionID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(expiresIn),
		CreatedAt:      time.Now(),
	}

	if err := r.Save(ctx, token); err != nil {
		return nil, fmt.Errorf("save confirmation token: %w", err)
	}

	return token, nil
}

// CreateUnsubscribeToken creates a new unsubscribe token
func (r *TokenRepositoryAdapter) CreateUnsubscribeToken(ctx context.Context, subscriptionID uint, expiresIn time.Duration) (*ports.TokenData, error) {
	if subscriptionID == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}

	token := &ports.TokenData{
		Value:          uuid.New().String(),
		SubscriptionID: subscriptionID,
		Type:           "unsubscribe",
		ExpiresAt:      time.Now().Add(expiresIn),
		CreatedAt:      time.Now(),
	}

	if err := r.Save(ctx, token); err != nil {
		return nil, fmt.Errorf("save unsubscribe token: %w", err)
	}

	return token, nil
}

// dataToModel converts port data to database model
func (r *TokenRepositoryAdapter) dataToModel(data *ports.TokenData) *TokenModel {
	return &TokenModel{
		ID:             data.ID,
		Token:          data.Value, // Map domain Value → database Token
		SubscriptionID: data.SubscriptionID,
		Type:           data.Type,
		ExpiresAt:      data.ExpiresAt,
		CreatedAt:      data.CreatedAt,
	}
}

// modelToData converts database model to port data
func (r *TokenRepositoryAdapter) modelToData(model *TokenModel) *ports.TokenData {
	return &ports.TokenData{
		ID:             model.ID,
		Value:          model.Token, // Map database Token → domain Value
		SubscriptionID: model.SubscriptionID,
		Type:           model.Type,
		ExpiresAt:      model.ExpiresAt,
		CreatedAt:      model.CreatedAt,
	}
}
