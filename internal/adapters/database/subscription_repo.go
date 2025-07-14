package database

import (
	"context"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// Error message constants for subscription repository
const (
	// Validation error messages
	ErrSubscriptionNil             = "subscription cannot be nil"
	ErrSubscriptionIDZero          = "subscription ID cannot be zero"
	ErrSubscriptionIDZeroForUpdate = "subscription ID cannot be zero for update"
	ErrSubscriptionIDZeroForDelete = "subscription ID cannot be zero for delete"
	ErrEmailEmpty                  = "email cannot be empty"
	ErrCityEmpty                   = "city cannot be empty"
	ErrFrequencyEmpty              = "frequency cannot be empty"

	// Operation error messages
	ErrFailedToSave             = "failed to save subscription"
	ErrFailedToFindByID         = "failed to find subscription by ID"
	ErrFailedToFind             = "failed to find subscription"
	ErrFailedToUpdate           = "failed to update subscription"
	ErrFailedToDelete           = "failed to delete subscription"
	ErrFailedToGetConfirmed     = "failed to get confirmed subscriptions"
	ErrFailedToCountByFrequency = "failed to count subscriptions by frequency"
	ErrFailedToCountConfirmed   = "failed to count confirmed subscriptions"

	// Not found error messages
	ErrSubscriptionNotFound = "subscription not found"
)

// SubscriptionModel represents the database model for subscriptions
type SubscriptionModel struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"index;not null"`
	City      string `gorm:"not null"`
	Frequency string `gorm:"not null"`
	Confirmed bool   `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (SubscriptionModel) TableName() string {
	return "subscriptions"
}

// SubscriptionRepositoryAdapter implements the SubscriptionRepository port using GORM
type SubscriptionRepositoryAdapter struct {
	db *gorm.DB
}

// NewSubscriptionRepositoryAdapter creates a new subscription repository adapter
func NewSubscriptionRepositoryAdapter(db *gorm.DB) ports.SubscriptionRepository {
	return &SubscriptionRepositoryAdapter{db: db}
}

// Save persists a subscription to the database
func (r *SubscriptionRepositoryAdapter) Save(ctx context.Context, sub *ports.SubscriptionData) error {
	if sub == nil {
		return infrastructure.NewDatabaseError(ErrSubscriptionNil, nil)
	}

	model := r.dataToModel(sub)
	var result *gorm.DB

	if sub.ID == 0 {
		result = r.db.WithContext(ctx).Create(model)
		sub.ID = model.ID
	} else {
		result = r.db.WithContext(ctx).Save(model)
	}

	if result.Error != nil {
		return infrastructure.NewDatabaseError(ErrFailedToSave, result.Error)
	}

	return nil
}

// FindByID retrieves a subscription by its ID
func (r *SubscriptionRepositoryAdapter) FindByID(ctx context.Context, id uint) (*ports.SubscriptionData, error) {
	if id == 0 {
		return nil, infrastructure.NewDatabaseError(ErrSubscriptionIDZero, nil)
	}

	var model SubscriptionModel
	result := r.db.WithContext(ctx).First(&model, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ports.NewNotFoundError(ErrSubscriptionNotFound)
		}
		return nil, infrastructure.NewDatabaseError(ErrFailedToFindByID, result.Error)
	}

	return r.modelToData(&model), nil
}

// FindByEmail retrieves a subscription by email and city
func (r *SubscriptionRepositoryAdapter) FindByEmail(ctx context.Context, email, city string) (*ports.SubscriptionData, error) {
	if email == "" {
		return nil, infrastructure.NewDatabaseError(ErrEmailEmpty, nil)
	}
	if city == "" {
		return nil, infrastructure.NewDatabaseError(ErrCityEmpty, nil)
	}

	var model SubscriptionModel
	result := r.db.WithContext(ctx).Where("email = ? AND city = ?", email, city).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ports.NewNotFoundError(ErrSubscriptionNotFound)
		}
		return nil, infrastructure.NewDatabaseError(ErrFailedToFind, result.Error)
	}

	return r.modelToData(&model), nil
}

// Update modifies an existing subscription
func (r *SubscriptionRepositoryAdapter) Update(ctx context.Context, sub *ports.SubscriptionData) error {
	if sub == nil {
		return infrastructure.NewDatabaseError(ErrSubscriptionNil, nil)
	}
	if sub.ID == 0 {
		return infrastructure.NewDatabaseError(ErrSubscriptionIDZeroForUpdate, nil)
	}

	model := r.dataToModel(sub)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return infrastructure.NewDatabaseError(ErrFailedToUpdate, result.Error)
	}

	return nil
}

// Delete removes a subscription from the database
func (r *SubscriptionRepositoryAdapter) Delete(ctx context.Context, sub *ports.SubscriptionData) error {
	if sub == nil {
		return infrastructure.NewDatabaseError(ErrSubscriptionNil, nil)
	}
	if sub.ID == 0 {
		return infrastructure.NewDatabaseError(ErrSubscriptionIDZeroForDelete, nil)
	}

	result := r.db.WithContext(ctx).Delete(&SubscriptionModel{}, sub.ID)
	if result.Error != nil {
		return infrastructure.NewDatabaseError(ErrFailedToDelete, result.Error)
	}

	return nil
}

// GetConfirmedByFrequency retrieves all confirmed subscriptions for a specific frequency
func (r *SubscriptionRepositoryAdapter) GetConfirmedByFrequency(ctx context.Context, frequency string) ([]*ports.SubscriptionData, error) {
	if frequency == "" {
		return nil, infrastructure.NewDatabaseError(ErrFrequencyEmpty, nil)
	}

	var models []SubscriptionModel
	result := r.db.WithContext(ctx).Where("frequency = ? AND confirmed = ?", frequency, true).Find(&models)
	if result.Error != nil {
		return nil, infrastructure.NewDatabaseError(ErrFailedToGetConfirmed, result.Error)
	}

	subscriptions := make([]*ports.SubscriptionData, len(models))
	for i, model := range models {
		subscriptions[i] = r.modelToData(&model)
	}

	return subscriptions, nil
}

// FindActiveByFrequency retrieves all active (confirmed) subscriptions for a specific frequency
func (r *SubscriptionRepositoryAdapter) FindActiveByFrequency(ctx context.Context, frequency string) ([]*ports.SubscriptionData, error) {
	return r.GetConfirmedByFrequency(ctx, frequency)
}

// FindConfirmedByFrequency retrieves all confirmed subscriptions for a specific frequency
func (r *SubscriptionRepositoryAdapter) FindConfirmedByFrequency(ctx context.Context, frequency string) ([]*ports.SubscriptionData, error) {
	return r.GetConfirmedByFrequency(ctx, frequency)
}

// CountByFrequency counts subscriptions by frequency
func (r *SubscriptionRepositoryAdapter) CountByFrequency(ctx context.Context, frequency string) (int64, error) {
	if frequency == "" {
		return 0, infrastructure.NewDatabaseError(ErrFrequencyEmpty, nil)
	}

	var count int64
	result := r.db.WithContext(ctx).Model(&SubscriptionModel{}).Where("frequency = ? AND confirmed = ?", frequency, true).Count(&count)
	if result.Error != nil {
		return 0, infrastructure.NewDatabaseError(ErrFailedToCountByFrequency, result.Error)
	}

	return count, nil
}

// CountConfirmed counts all confirmed subscriptions
func (r *SubscriptionRepositoryAdapter) CountConfirmed(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&SubscriptionModel{}).Where("confirmed = ?", true).Count(&count)
	if result.Error != nil {
		return 0, infrastructure.NewDatabaseError(ErrFailedToCountConfirmed, result.Error)
	}

	return count, nil
}

// dataToModel converts port data to database model
func (r *SubscriptionRepositoryAdapter) dataToModel(data *ports.SubscriptionData) *SubscriptionModel {
	return &SubscriptionModel{
		ID:        data.ID,
		Email:     data.Email,
		City:      data.City,
		Frequency: data.Frequency,
		Confirmed: data.Confirmed,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}
}

// modelToData converts database model to port data
func (r *SubscriptionRepositoryAdapter) modelToData(model *SubscriptionModel) *ports.SubscriptionData {
	return &ports.SubscriptionData{
		ID:        model.ID,
		Email:     model.Email,
		City:      model.City,
		Frequency: model.Frequency,
		Confirmed: model.Confirmed,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
