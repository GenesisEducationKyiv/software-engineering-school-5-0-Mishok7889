package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&SubscriptionModel{})
	require.NoError(t, err)

	return db
}

func TestSubscriptionRepository_Save_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	sub := &ports.SubscriptionData{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Save(ctx, sub)
	assert.NoError(t, err)
	assert.NotZero(t, sub.ID)
}

func TestSubscriptionRepository_Save_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	sub := &ports.SubscriptionData{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Save(ctx, sub)
	require.NoError(t, err)

	sub.Confirmed = true
	err = repo.Save(ctx, sub)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, sub.ID)
	require.NoError(t, err)
	assert.True(t, found.Confirmed)
}

func TestSubscriptionRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	sub := &ports.SubscriptionData{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Save(ctx, sub)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, sub.ID)
	assert.NoError(t, err)
	assert.Equal(t, sub.Email, found.Email)
	assert.Equal(t, sub.City, found.City)
	assert.Equal(t, sub.Frequency, found.Frequency)
	assert.Equal(t, sub.Confirmed, found.Confirmed)
}

func TestSubscriptionRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, found)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.NotFoundError, appErr.Type)
}

func TestSubscriptionRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	sub := &ports.SubscriptionData{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Save(ctx, sub)
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, "test@example.com", "London")
	assert.NoError(t, err)
	assert.Equal(t, sub.Email, found.Email)
	assert.Equal(t, sub.City, found.City)
}

func TestSubscriptionRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	found, err := repo.FindByEmail(ctx, "nonexistent@example.com", "London")
	assert.Error(t, err)
	assert.Nil(t, found)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.NotFoundError, appErr.Type)
}

func TestSubscriptionRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	sub := &ports.SubscriptionData{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Save(ctx, sub)
	require.NoError(t, err)

	err = repo.Delete(ctx, sub)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, sub.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestSubscriptionRepository_GetConfirmedByFrequency(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	tests := []struct {
		email     string
		frequency string
		confirmed bool
	}{
		{"daily1@example.com", "daily", true},
		{"daily2@example.com", "daily", true},
		{"hourly1@example.com", "hourly", true},
		{"daily3@example.com", "daily", false},
	}

	for _, tt := range tests {
		sub := &ports.SubscriptionData{
			Email:     tt.email,
			City:      "London",
			Frequency: tt.frequency,
			Confirmed: tt.confirmed,
		}
		err := repo.Save(ctx, sub)
		require.NoError(t, err)
	}

	dailySubscriptions, err := repo.GetConfirmedByFrequency(ctx, "daily")
	assert.NoError(t, err)
	assert.Len(t, dailySubscriptions, 2)

	hourlySubscriptions, err := repo.GetConfirmedByFrequency(ctx, "hourly")
	assert.NoError(t, err)
	assert.Len(t, hourlySubscriptions, 1)
}

func TestSubscriptionRepository_CountByFrequency(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	tests := []struct {
		frequency string
		confirmed bool
	}{
		{"daily", true},
		{"daily", true},
		{"daily", false},
		{"hourly", true},
	}

	for i, tt := range tests {
		sub := &ports.SubscriptionData{
			Email:     "test" + string(rune(i)) + "@example.com",
			City:      "London",
			Frequency: tt.frequency,
			Confirmed: tt.confirmed,
		}
		err := repo.Save(ctx, sub)
		require.NoError(t, err)
	}

	count, err := repo.CountByFrequency(ctx, "daily")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = repo.CountByFrequency(ctx, "hourly")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSubscriptionRepository_ValidationErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepositoryAdapter(db)
	ctx := context.Background()

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "Save nil subscription",
			test: func() error {
				return repo.Save(ctx, nil)
			},
		},
		{
			name: "FindByID zero ID",
			test: func() error {
				_, err := repo.FindByID(ctx, 0)
				return err
			},
		},
		{
			name: "FindByEmail empty email",
			test: func() error {
				_, err := repo.FindByEmail(ctx, "", "London")
				return err
			},
		},
		{
			name: "FindByEmail empty city",
			test: func() error {
				_, err := repo.FindByEmail(ctx, "test@example.com", "")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			assert.Error(t, err)

			var appErr *errors.AppError
			assert.ErrorAs(t, err, &appErr)
			assert.Equal(t, errors.ValidationError, appErr.Type)
		})
	}
}
