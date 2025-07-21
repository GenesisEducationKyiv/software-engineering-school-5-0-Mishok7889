package ports

import (
	"context"
	"time"
)

// TokenData represents token data for persistence
type TokenData struct {
	ID             uint
	Value          string
	SubscriptionID uint
	Type           string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// IsExpired checks if the token has expired
func (td *TokenData) IsExpired() bool {
	return time.Now().After(td.ExpiresAt)
}

// TokenRepository defines the contract for token data persistence
type TokenRepository interface {
	Save(ctx context.Context, token *TokenData) error
	FindByToken(ctx context.Context, tokenStr string) (*TokenData, error)
	FindBySubscriptionID(ctx context.Context, subscriptionID uint, tokenType string) (*TokenData, error)
	FindBySubscriptionIDAndType(ctx context.Context, subscriptionID uint, tokenType string) (*TokenData, error)
	Delete(ctx context.Context, token *TokenData) error
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}
