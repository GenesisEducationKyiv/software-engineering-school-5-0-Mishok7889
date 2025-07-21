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

// SubscriptionFilter defines criteria for filtering subscriptions
type SubscriptionFilter struct {
	ID        *uint
	Email     *string
	City      *string
	Frequency *string
	Confirmed *bool
}

// SubscriptionRepository defines the contract for subscription data persistence
type SubscriptionRepository interface {
	Save(ctx context.Context, sub *SubscriptionData) error
	FindByID(ctx context.Context, id uint) (*SubscriptionData, error)
	FindByEmail(ctx context.Context, email, city string) (*SubscriptionData, error)
	Find(ctx context.Context, filter SubscriptionFilter) ([]*SubscriptionData, error)
	Update(ctx context.Context, sub *SubscriptionData) error
	Delete(ctx context.Context, sub *SubscriptionData) error
	CountByFrequency(ctx context.Context, frequency string) (int64, error)
	CountConfirmed(ctx context.Context) (int64, error)
}
