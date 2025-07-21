package ports

import (
	"context"
	"time"
)

// TokenStats represents token statistics
type TokenStats struct {
	TotalTokens   int64
	ExpiredTokens int64
	ActiveTokens  int64
	LastCleanupAt time.Time
}

// TokenService defines the contract for token business operations
type TokenService interface {
	// CleanupExpiredTokens removes expired tokens and returns count of deleted tokens
	CleanupExpiredTokens(ctx context.Context) (int64, error)

	// GetTokenStats returns statistics about tokens in the system
	GetTokenStats(ctx context.Context) (TokenStats, error)

	// ValidateToken checks if a token exists and is not expired
	ValidateToken(ctx context.Context, tokenValue string) (*TokenData, error)
}
