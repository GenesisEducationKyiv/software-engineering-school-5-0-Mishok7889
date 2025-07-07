package ports

import (
	"context"
	"time"
)

// SubscriptionData represents subscription data for persistence
type SubscriptionData struct {
	ID        uint
	Email     string
	City      string
	Frequency string
	Confirmed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TokenData represents token data for persistence
type TokenData struct {
	ID             uint
	Value          string
	SubscriptionID uint
	Type           string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// SubscriptionRepository defines the contract for subscription data persistence
type SubscriptionRepository interface {
	Save(ctx context.Context, sub *SubscriptionData) error
	FindByID(ctx context.Context, id uint) (*SubscriptionData, error)
	FindByEmail(ctx context.Context, email, city string) (*SubscriptionData, error)
	Update(ctx context.Context, sub *SubscriptionData) error
	Delete(ctx context.Context, sub *SubscriptionData) error
	GetConfirmedByFrequency(ctx context.Context, frequency string) ([]*SubscriptionData, error)
	CountByFrequency(ctx context.Context, frequency string) (int64, error)
	CountConfirmed(ctx context.Context) (int64, error)
}

// TokenRepository defines the contract for token data persistence
type TokenRepository interface {
	Save(ctx context.Context, token *TokenData) error
	FindByToken(ctx context.Context, tokenStr string) (*TokenData, error)
	FindBySubscriptionIDAndType(ctx context.Context, subscriptionID uint, tokenType string) (*TokenData, error)
	Delete(ctx context.Context, token *TokenData) error
	DeleteExpiredTokens(ctx context.Context) (int64, error)
	CreateConfirmationToken(ctx context.Context, subscriptionID uint, expiresIn time.Duration) (*TokenData, error)
	CreateUnsubscribeToken(ctx context.Context, subscriptionID uint, expiresIn time.Duration) (*TokenData, error)
}
