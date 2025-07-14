package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
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
	assert.True(t, ports.IsNotFoundError(err))
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
	assert.True(t, ports.IsNotFoundError(err))
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

func TestSubscriptionRepository_Find(t *testing.T) {
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

	// Test finding confirmed daily subscriptions
	dailyFreq := "daily"
	confirmed := true
	filter := ports.SubscriptionFilter{
		Frequency: &dailyFreq,
		Confirmed: &confirmed,
	}
	dailySubscriptions, err := repo.Find(ctx, filter)
	assert.NoError(t, err)
	assert.Len(t, dailySubscriptions, 2)

	// Test finding confirmed hourly subscriptions
	hourlyFreq := "hourly"
	hourlyFilter := ports.SubscriptionFilter{
		Frequency: &hourlyFreq,
		Confirmed: &confirmed,
	}
	hourlySubscriptions, err := repo.Find(ctx, hourlyFilter)
	assert.NoError(t, err)
	assert.Len(t, hourlySubscriptions, 1)

	// Test finding all daily subscriptions (including unconfirmed)
	dailyAllFilter := ports.SubscriptionFilter{
		Frequency: &dailyFreq,
	}
	allDailySubscriptions, err := repo.Find(ctx, dailyAllFilter)
	assert.NoError(t, err)
	assert.Len(t, allDailySubscriptions, 3)

	// Test finding by email
	email := "daily1@example.com"
	city := "London"
	emailFilter := ports.SubscriptionFilter{
		Email: &email,
		City:  &city,
	}
	emailSubscriptions, err := repo.Find(ctx, emailFilter)
	assert.NoError(t, err)
	assert.Len(t, emailSubscriptions, 1)
	assert.Equal(t, "daily1@example.com", emailSubscriptions[0].Email)
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
		{
			name: "Find with empty filter",
			test: func() error {
				_, err := repo.Find(ctx, ports.SubscriptionFilter{})
				return err
			},
		},
		{
			name: "Find with zero ID",
			test: func() error {
				id := uint(0)
				_, err := repo.Find(ctx, ports.SubscriptionFilter{ID: &id})
				return err
			},
		},
		{
			name: "Find with empty email",
			test: func() error {
				email := ""
				_, err := repo.Find(ctx, ports.SubscriptionFilter{Email: &email})
				return err
			},
		},
		{
			name: "Find with empty city",
			test: func() error {
				city := ""
				_, err := repo.Find(ctx, ports.SubscriptionFilter{City: &city})
				return err
			},
		},
		{
			name: "Find with empty frequency",
			test: func() error {
				frequency := ""
				_, err := repo.Find(ctx, ports.SubscriptionFilter{Frequency: &frequency})
				return err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.test()
			assert.Error(t, err)

			var infraErr *infrastructure.InfrastructureError
			assert.ErrorAs(t, err, &infraErr)
			assert.Equal(t, "DATABASE_ERROR", infraErr.Type)
		})
	}
}
