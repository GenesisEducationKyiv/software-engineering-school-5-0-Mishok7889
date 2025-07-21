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

// Validate checks if the filter has valid criteria
// Returns validation errors that should be wrapped by the adapter layer
func (f *SubscriptionFilter) Validate() error {
	// Check that at least one filter criterion is provided
	if f.ID == nil && f.Email == nil && f.City == nil && f.Frequency == nil && f.Confirmed == nil {
		return NewValidationError("at least one filter criterion must be provided")
	}

	// Validate ID if provided
	if f.ID != nil && *f.ID == 0 {
		return NewValidationError("subscription ID cannot be zero")
	}

	// Validate Email if provided
	if f.Email != nil && *f.Email == "" {
		return NewValidationError("email cannot be empty")
	}

	// Validate City if provided
	if f.City != nil && *f.City == "" {
		return NewValidationError("city cannot be empty")
	}

	// Validate Frequency if provided
	if f.Frequency != nil && *f.Frequency == "" {
		return NewValidationError("frequency cannot be empty")
	}

	return nil
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
