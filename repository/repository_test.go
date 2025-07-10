package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	weathererr "weatherapi.app/errors"
	"weatherapi.app/models"
)

// Setup test database with in-memory SQLite
func setupTestDB(t *testing.T) *gorm.DB {
	// Use unique database for each test to avoid data pollution
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.Subscription{}, &models.Token{})
	assert.NoError(t, err)

	return db
}

// Clean up database after each test
func cleanupTestDB(_ *testing.T, db *gorm.DB) {
	db.Exec("DELETE FROM tokens")
	db.Exec("DELETE FROM subscriptions")
}

// TestSubscriptionRepository_FindByEmail tests finding a subscription by email and city
func TestSubscriptionRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidInput_NotFound", func(t *testing.T) {
		sub, err := repo.FindByEmail("nonexistent@example.com", "London")
		assert.NoError(t, err)
		assert.Nil(t, sub)
	})

	t.Run("ValidInput_Found", func(t *testing.T) {
		testSub := models.Subscription{
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: true,
		}

		result := db.Create(&testSub)
		assert.NoError(t, result.Error)

		sub, err := repo.FindByEmail("test@example.com", "London")
		assert.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, "test@example.com", sub.Email)
		assert.Equal(t, "London", sub.City)
		assert.Equal(t, "daily", sub.Frequency)
		assert.True(t, sub.Confirmed)
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		sub, err := repo.FindByEmail("", "London")
		assert.Error(t, err)
		assert.Nil(t, sub)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "email cannot be empty")
	})

	t.Run("EmptyCity", func(t *testing.T) {
		sub, err := repo.FindByEmail("test@example.com", "")
		assert.Error(t, err)
		assert.Nil(t, sub)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "city cannot be empty")
	})
}

func TestSubscriptionRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidID_Found", func(t *testing.T) {
		testSub := models.Subscription{
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: true,
		}

		result := db.Create(&testSub)
		assert.NoError(t, result.Error)

		sub, err := repo.FindByID(testSub.ID)
		assert.NoError(t, err)
		assert.NotNil(t, sub)
		assert.Equal(t, testSub.ID, sub.ID)
		assert.Equal(t, "test@example.com", sub.Email)
	})

	t.Run("ValidID_NotFound", func(t *testing.T) {
		sub, err := repo.FindByID(999)
		assert.Error(t, err)
		assert.Nil(t, sub)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.NotFoundError, appErr.Type)
	})

	t.Run("ZeroID", func(t *testing.T) {
		sub, err := repo.FindByID(0)
		assert.Error(t, err)
		assert.Nil(t, sub)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "subscription ID cannot be zero")
	})
}

func TestSubscriptionRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidSubscription", func(t *testing.T) {
		testSub := &models.Subscription{
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: false,
		}

		err := repo.Create(testSub)
		assert.NoError(t, err)
		assert.NotZero(t, testSub.ID)

		var dbSub models.Subscription
		result := db.First(&dbSub, testSub.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, "test@example.com", dbSub.Email)
		assert.Equal(t, "London", dbSub.City)
		assert.Equal(t, "daily", dbSub.Frequency)
		assert.False(t, dbSub.Confirmed)
	})

	t.Run("NilSubscription", func(t *testing.T) {
		err := repo.Create(nil)
		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "subscription cannot be nil")
	})
}

func TestSubscriptionRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidUpdate", func(t *testing.T) {
		testSub := &models.Subscription{
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: false,
		}

		err := repo.Create(testSub)
		assert.NoError(t, err)

		testSub.Confirmed = true
		testSub.Frequency = "hourly"

		err = repo.Update(testSub)
		assert.NoError(t, err)

		var dbSub models.Subscription
		result := db.First(&dbSub, testSub.ID)
		assert.NoError(t, result.Error)
		assert.True(t, dbSub.Confirmed)
		assert.Equal(t, "hourly", dbSub.Frequency)
	})

	t.Run("NilSubscription", func(t *testing.T) {
		err := repo.Update(nil)
		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "subscription cannot be nil")
	})
}

func TestSubscriptionRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidDelete", func(t *testing.T) {
		testSub := &models.Subscription{
			Email:     "test@example.com",
			City:      "London",
			Frequency: "daily",
			Confirmed: true,
		}

		err := repo.Create(testSub)
		assert.NoError(t, err)

		err = repo.Delete(testSub)
		assert.NoError(t, err)

		var dbSub models.Subscription
		result := db.First(&dbSub, testSub.ID)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("NilSubscription", func(t *testing.T) {
		err := repo.Delete(nil)
		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "subscription cannot be nil")
	})
}

func TestSubscriptionRepository_GetSubscriptionsForUpdates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)
	defer cleanupTestDB(t, db)

	t.Run("ValidFrequency", func(t *testing.T) {
		// Clean up before this specific test
		cleanupTestDB(t, db)

		testSubs := []models.Subscription{
			{Email: "test1@example.com", City: "London", Frequency: "daily", Confirmed: true},
			{Email: "test2@example.com", City: "Paris", Frequency: "daily", Confirmed: true},
			{Email: "test3@example.com", City: "Berlin", Frequency: "hourly", Confirmed: true},
			{Email: "test4@example.com", City: "Madrid", Frequency: "daily", Confirmed: false},
		}

		for _, sub := range testSubs {
			result := db.Create(&sub)
			assert.NoError(t, result.Error)
		}

		subs, err := repo.GetSubscriptionsForUpdates("daily")
		assert.NoError(t, err)
		assert.Len(t, subs, 2) // Only confirmed daily subscriptions

		for _, sub := range subs {
			assert.Equal(t, "daily", sub.Frequency)
			assert.True(t, sub.Confirmed)
		}
	})

	t.Run("EmptyFrequency", func(t *testing.T) {
		subs, err := repo.GetSubscriptionsForUpdates("")
		assert.Error(t, err)
		assert.Nil(t, subs)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "frequency cannot be empty")
	})
}

func TestTokenRepository_CreateToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)
	defer cleanupTestDB(t, db)

	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}
	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	t.Run("ValidToken", func(t *testing.T) {
		token, err := repo.CreateToken(testSub.ID, "confirmation", 24*time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.Token)
		assert.Equal(t, testSub.ID, token.SubscriptionID)
		assert.Equal(t, "confirmation", token.Type)

		var dbToken models.Token
		result := db.First(&dbToken, token.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, token.Token, dbToken.Token)
		assert.Equal(t, testSub.ID, dbToken.SubscriptionID)
		assert.Equal(t, "confirmation", dbToken.Type)
	})

	t.Run("ZeroSubscriptionID", func(t *testing.T) {
		token, err := repo.CreateToken(0, "confirmation", 24*time.Hour)
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "subscription ID cannot be zero")
	})

	t.Run("EmptyTokenType", func(t *testing.T) {
		token, err := repo.CreateToken(testSub.ID, "", 24*time.Hour)
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "token type cannot be empty")
	})

	t.Run("NegativeExpiration", func(t *testing.T) {
		token, err := repo.CreateToken(testSub.ID, "confirmation", -1*time.Hour)
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "expiration duration must be positive")
	})
}

func TestTokenRepository_FindByToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)
	defer cleanupTestDB(t, db)

	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}
	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	t.Run("ValidToken_Found", func(t *testing.T) {
		tokenString := "test-token-123"
		testToken := models.Token{
			Token:          tokenString,
			SubscriptionID: testSub.ID,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		result := db.Create(&testToken)
		assert.NoError(t, result.Error)

		token, err := repo.FindByToken(tokenString)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, tokenString, token.Token)
		assert.Equal(t, testSub.ID, token.SubscriptionID)
		assert.Equal(t, "confirmation", token.Type)
	})

	t.Run("ValidToken_NotFound", func(t *testing.T) {
		token, err := repo.FindByToken("nonexistent-token")
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.NotFoundError, appErr.Type)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		tokenString := "expired-token-123"
		testToken := models.Token{
			Token:          tokenString,
			SubscriptionID: testSub.ID,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(-1 * time.Hour), // Expired
		}

		result := db.Create(&testToken)
		assert.NoError(t, result.Error)

		token, err := repo.FindByToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.NotFoundError, appErr.Type)
	})

	t.Run("EmptyToken", func(t *testing.T) {
		token, err := repo.FindByToken("")
		assert.Error(t, err)
		assert.Nil(t, token)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "token cannot be empty")
	})
}

func TestTokenRepository_DeleteToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)
	defer cleanupTestDB(t, db)

	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}
	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	t.Run("ValidToken", func(t *testing.T) {
		testToken := &models.Token{
			Token:          "test-token-123",
			SubscriptionID: testSub.ID,
			Type:           "confirmation",
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		result := db.Create(testToken)
		assert.NoError(t, result.Error)

		err := repo.DeleteToken(testToken)
		assert.NoError(t, err)

		var dbToken models.Token
		result = db.First(&dbToken, testToken.ID)
		assert.Error(t, result.Error)
		assert.Equal(t, gorm.ErrRecordNotFound, result.Error)
	})

	t.Run("NilToken", func(t *testing.T) {
		err := repo.DeleteToken(nil)
		assert.Error(t, err)

		var appErr *weathererr.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, weathererr.ValidationError, appErr.Type)
		assert.Contains(t, appErr.Message, "token cannot be nil")
	})
}

func TestTokenRepository_DeleteExpiredTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)
	defer cleanupTestDB(t, db)

	// Clean up before test
	cleanupTestDB(t, db)

	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}
	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	// Create mix of expired and valid tokens
	tokens := []models.Token{
		{Token: "valid1", SubscriptionID: testSub.ID, Type: "confirmation", ExpiresAt: time.Now().Add(1 * time.Hour)},
		{Token: "expired1", SubscriptionID: testSub.ID, Type: "confirmation", ExpiresAt: time.Now().Add(-1 * time.Hour)},
		{Token: "expired2", SubscriptionID: testSub.ID, Type: "unsubscribe", ExpiresAt: time.Now().Add(-2 * time.Hour)},
		{Token: "valid2", SubscriptionID: testSub.ID, Type: "unsubscribe", ExpiresAt: time.Now().Add(2 * time.Hour)},
	}

	for _, token := range tokens {
		result := db.Create(&token)
		assert.NoError(t, result.Error)
	}

	err := repo.DeleteExpiredTokens()
	assert.NoError(t, err)

	var remainingTokens []models.Token
	result = db.Find(&remainingTokens)
	assert.NoError(t, result.Error)
	assert.Len(t, remainingTokens, 2) // Only valid tokens should remain

	for _, token := range remainingTokens {
		assert.True(t, token.ExpiresAt.After(time.Now()))
	}
}
