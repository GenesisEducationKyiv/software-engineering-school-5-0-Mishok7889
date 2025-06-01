package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"weatherapi.app/models"
)

// Setup test database with in-memory SQLite
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&models.Subscription{}, &models.Token{})
	assert.NoError(t, err)

	return db
}

// TestSubscriptionRepository_FindByEmail tests finding a subscription by email and city
func TestSubscriptionRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)

	// Test with non-existent subscription
	sub, err := repo.FindByEmail("nonexistent@example.com", "London")
	assert.NoError(t, err)
	assert.Nil(t, sub)

	// Create a test subscription
	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	// Test with existing subscription
	sub, err = repo.FindByEmail("test@example.com", "London")
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, "test@example.com", sub.Email)
	assert.Equal(t, "London", sub.City)
	assert.Equal(t, "daily", sub.Frequency)
	assert.True(t, sub.Confirmed)
}

// TestSubscriptionRepository_Create tests creating a new subscription
func TestSubscriptionRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSubscriptionRepository(db)

	// Create a test subscription
	testSub := &models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	err := repo.Create(testSub)
	assert.NoError(t, err)
	assert.NotZero(t, testSub.ID)

	// Verify subscription was created in DB
	var dbSub models.Subscription
	result := db.First(&dbSub, testSub.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "test@example.com", dbSub.Email)
	assert.Equal(t, "London", dbSub.City)
	assert.Equal(t, "daily", dbSub.Frequency)
	assert.False(t, dbSub.Confirmed)
}

// TestTokenRepository_CreateToken tests creating a new token
func TestTokenRepository_CreateToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)

	// Create a test subscription first
	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	// Create a confirmation token
	token, err := repo.CreateToken(testSub.ID, "confirmation", 24*time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.NotEmpty(t, token.Token)
	assert.Equal(t, testSub.ID, token.SubscriptionID)
	assert.Equal(t, "confirmation", token.Type)

	// Verify token was created in DB
	var dbToken models.Token
	result = db.First(&dbToken, token.ID)
	assert.NoError(t, result.Error)
	assert.Equal(t, token.Token, dbToken.Token)
	assert.Equal(t, testSub.ID, dbToken.SubscriptionID)
	assert.Equal(t, "confirmation", dbToken.Type)
}

// TestTokenRepository_FindByToken tests finding a token by its string value
func TestTokenRepository_FindByToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTokenRepository(db)

	// Create a test subscription
	testSub := models.Subscription{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	result := db.Create(&testSub)
	assert.NoError(t, result.Error)

	// Create a test token
	tokenString := "test-token-123"
	testToken := models.Token{
		Token:          tokenString,
		SubscriptionID: testSub.ID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	result = db.Create(&testToken)
	assert.NoError(t, result.Error)

	// Find the token
	token, err := repo.FindByToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, tokenString, token.Token)
	assert.Equal(t, testSub.ID, token.SubscriptionID)
	assert.Equal(t, "confirmation", token.Type)

	// Test with non-existent token
	token, err = repo.FindByToken("nonexistent-token")
	assert.Error(t, err)
	assert.Nil(t, token)
}
