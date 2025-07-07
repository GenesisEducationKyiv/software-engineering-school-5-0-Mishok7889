package database

import (
	"context"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
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
		return errors.NewValidationError("subscription cannot be nil")
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
		return errors.NewDatabaseError("failed to save subscription", result.Error)
	}

	return nil
}

// FindByID retrieves a subscription by its ID
func (r *SubscriptionRepositoryAdapter) FindByID(ctx context.Context, id uint) (*ports.SubscriptionData, error) {
	if id == 0 {
		return nil, errors.NewValidationError("subscription ID cannot be zero")
	}

	var model SubscriptionModel
	result := r.db.WithContext(ctx).First(&model, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("subscription not found")
		}
		return nil, errors.NewDatabaseError("failed to find subscription by ID", result.Error)
	}

	return r.modelToData(&model), nil
}

// FindByEmail retrieves a subscription by email and city
func (r *SubscriptionRepositoryAdapter) FindByEmail(ctx context.Context, email, city string) (*ports.SubscriptionData, error) {
	if email == "" {
		return nil, errors.NewValidationError("email cannot be empty")
	}
	if city == "" {
		return nil, errors.NewValidationError("city cannot be empty")
	}

	var model SubscriptionModel
	result := r.db.WithContext(ctx).Where("email = ? AND city = ?", email, city).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("subscription not found")
		}
		return nil, errors.NewDatabaseError("failed to find subscription", result.Error)
	}

	return r.modelToData(&model), nil
}

// Update modifies an existing subscription
func (r *SubscriptionRepositoryAdapter) Update(ctx context.Context, sub *ports.SubscriptionData) error {
	if sub == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}
	if sub.ID == 0 {
		return errors.NewValidationError("subscription ID cannot be zero for update")
	}

	model := r.dataToModel(sub)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return errors.NewDatabaseError("failed to update subscription", result.Error)
	}

	return nil
}

// Delete removes a subscription from the database
func (r *SubscriptionRepositoryAdapter) Delete(ctx context.Context, sub *ports.SubscriptionData) error {
	if sub == nil {
		return errors.NewValidationError("subscription cannot be nil")
	}
	if sub.ID == 0 {
		return errors.NewValidationError("subscription ID cannot be zero for delete")
	}

	result := r.db.WithContext(ctx).Delete(&SubscriptionModel{}, sub.ID)
	if result.Error != nil {
		return errors.NewDatabaseError("failed to delete subscription", result.Error)
	}

	return nil
}

// GetConfirmedByFrequency retrieves all confirmed subscriptions for a specific frequency
func (r *SubscriptionRepositoryAdapter) GetConfirmedByFrequency(ctx context.Context, frequency string) ([]*ports.SubscriptionData, error) {
	if frequency == "" {
		return nil, errors.NewValidationError("frequency cannot be empty")
	}

	var models []SubscriptionModel
	result := r.db.WithContext(ctx).Where("frequency = ? AND confirmed = ?", frequency, true).Find(&models)
	if result.Error != nil {
		return nil, errors.NewDatabaseError("failed to get confirmed subscriptions", result.Error)
	}

	subscriptions := make([]*ports.SubscriptionData, len(models))
	for i, model := range models {
		subscriptions[i] = r.modelToData(&model)
	}

	return subscriptions, nil
}

// CountByFrequency counts subscriptions by frequency
func (r *SubscriptionRepositoryAdapter) CountByFrequency(ctx context.Context, frequency string) (int64, error) {
	if frequency == "" {
		return 0, errors.NewValidationError("frequency cannot be empty")
	}

	var count int64
	result := r.db.WithContext(ctx).Model(&SubscriptionModel{}).Where("frequency = ? AND confirmed = ?", frequency, true).Count(&count)
	if result.Error != nil {
		return 0, errors.NewDatabaseError("failed to count subscriptions by frequency", result.Error)
	}

	return count, nil
}

// CountConfirmed counts all confirmed subscriptions
func (r *SubscriptionRepositoryAdapter) CountConfirmed(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&SubscriptionModel{}).Where("confirmed = ?", true).Count(&count)
	if result.Error != nil {
		return 0, errors.NewDatabaseError("failed to count confirmed subscriptions", result.Error)
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
