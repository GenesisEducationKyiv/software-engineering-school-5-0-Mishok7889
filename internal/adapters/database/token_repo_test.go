package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func setupTokenTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&TokenModel{})
	require.NoError(t, err)

	return db
}

func TestTokenRepository_Save_Create(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	err := repo.Save(ctx, token)
	assert.NoError(t, err)
	assert.NotZero(t, token.ID)
}

func TestTokenRepository_FindByToken(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	err := repo.Save(ctx, token)
	require.NoError(t, err)

	found, err := repo.FindByToken(ctx, "test-token-123")
	assert.NoError(t, err)
	assert.Equal(t, token.Value, found.Value)
	assert.Equal(t, token.SubscriptionID, found.SubscriptionID)
	assert.Equal(t, token.Type, found.Type)
}

func TestTokenRepository_FindByToken_NotFound(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	found, err := repo.FindByToken(ctx, "nonexistent-token")
	assert.Error(t, err)
	assert.Nil(t, found)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.NotFoundError, appErr.Type)
}

func TestTokenRepository_FindByToken_Expired(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token := &ports.TokenData{
		Value:          "expired-token",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(-1 * time.Hour),
	}

	err := repo.Save(ctx, token)
	require.NoError(t, err)

	found, err := repo.FindByToken(ctx, "expired-token")
	assert.Error(t, err)
	assert.Nil(t, found)

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.NotFoundError, appErr.Type)
}

func TestTokenRepository_FindBySubscriptionIDAndType(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	err := repo.Save(ctx, token)
	require.NoError(t, err)

	found, err := repo.FindBySubscriptionIDAndType(ctx, 1, "confirmation")
	assert.NoError(t, err)
	assert.Equal(t, token.Value, found.Value)
	assert.Equal(t, token.SubscriptionID, found.SubscriptionID)
	assert.Equal(t, token.Type, found.Type)
}

func TestTokenRepository_Delete(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	err := repo.Save(ctx, token)
	require.NoError(t, err)

	err = repo.Delete(ctx, token)
	assert.NoError(t, err)

	found, err := repo.FindByToken(ctx, token.Value)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestTokenRepository_DeleteExpiredTokens(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	tokens := []*ports.TokenData{
		{
			Value:          "valid-token",
			SubscriptionID: 1,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		},
		{
			Value:          "expired-token-1",
			SubscriptionID: 2,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			Value:          "expired-token-2",
			SubscriptionID: 3,
			Type:           "unsubscribe",
			ExpiresAt:      time.Now().Add(-2 * time.Hour),
		},
	}

	for _, token := range tokens {
		err := repo.Save(ctx, token)
		require.NoError(t, err)
	}

	deletedCount, err := repo.DeleteExpiredTokens(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deletedCount)

	found, err := repo.FindByToken(ctx, "valid-token")
	assert.NoError(t, err)
	assert.NotNil(t, found)

	found, err = repo.FindByToken(ctx, "expired-token-1")
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestTokenRepository_CreateConfirmationToken(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token, err := repo.CreateConfirmationToken(ctx, 1, 24*time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.NotEmpty(t, token.Value)
	assert.Equal(t, uint(1), token.SubscriptionID)
	assert.Equal(t, "confirmation", token.Type)
	assert.True(t, token.ExpiresAt.After(time.Now()))
}

func TestTokenRepository_CreateUnsubscribeToken(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	token, err := repo.CreateUnsubscribeToken(ctx, 1, 365*24*time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.NotEmpty(t, token.Value)
	assert.Equal(t, uint(1), token.SubscriptionID)
	assert.Equal(t, "unsubscribe", token.Type)
	assert.True(t, token.ExpiresAt.After(time.Now()))
}

func TestTokenRepository_ValidationErrors(t *testing.T) {
	db := setupTokenTestDB(t)
	repo := NewTokenRepositoryAdapter(db)
	ctx := context.Background()

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "Save nil token",
			test: func() error {
				return repo.Save(ctx, nil)
			},
		},
		{
			name: "FindByToken empty token",
			test: func() error {
				_, err := repo.FindByToken(ctx, "")
				return err
			},
		},
		{
			name: "FindBySubscriptionIDAndType zero ID",
			test: func() error {
				_, err := repo.FindBySubscriptionIDAndType(ctx, 0, "confirmation")
				return err
			},
		},
		{
			name: "FindBySubscriptionIDAndType empty type",
			test: func() error {
				_, err := repo.FindBySubscriptionIDAndType(ctx, 1, "")
				return err
			},
		},
		{
			name: "CreateConfirmationToken zero ID",
			test: func() error {
				_, err := repo.CreateConfirmationToken(ctx, 0, 24*time.Hour)
				return err
			},
		},
		{
			name: "CreateUnsubscribeToken zero ID",
			test: func() error {
				_, err := repo.CreateUnsubscribeToken(ctx, 0, 24*time.Hour)
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
