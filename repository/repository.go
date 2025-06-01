package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"weatherapi.app/models"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) FindByEmail(email, city string) (*models.Subscription, error) {
	fmt.Printf("[DEBUG] SubscriptionRepository.FindByEmail: email=%s, city=%s\n", email, city)
	
	var subscription models.Subscription
	result := r.db.Where("email = ? AND city = ?", email, city).First(&subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("[DEBUG] No subscription found")
			return nil, nil
		}
		fmt.Printf("[ERROR] Database error when finding subscription: %v\n", result.Error)
		return nil, result.Error
	}
	
	fmt.Printf("[DEBUG] Found subscription: %+v\n", subscription)
	return &subscription, nil
}

func (r *SubscriptionRepository) FindByID(id uint) (*models.Subscription, error) {
	fmt.Printf("[DEBUG] SubscriptionRepository.FindByID: id=%d\n", id)
	
	var subscription models.Subscription
	result := r.db.First(&subscription, id)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when finding subscription by ID: %v\n", result.Error)
		return nil, result.Error
	}
	
	fmt.Printf("[DEBUG] Found subscription: %+v\n", subscription)
	return &subscription, nil
}

func (r *SubscriptionRepository) Create(subscription *models.Subscription) error {
	fmt.Printf("[DEBUG] SubscriptionRepository.Create: %+v\n", subscription)
	
	result := r.db.Create(subscription)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when creating subscription: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Printf("[DEBUG] Created subscription with ID: %d\n", subscription.ID)
	return nil
}

func (r *SubscriptionRepository) Update(subscription *models.Subscription) error {
	fmt.Printf("[DEBUG] SubscriptionRepository.Update: %+v\n", subscription)
	
	result := r.db.Save(subscription)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when updating subscription: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Println("[DEBUG] Updated subscription successfully")
	return nil
}

func (r *SubscriptionRepository) Delete(subscription *models.Subscription) error {
	fmt.Printf("[DEBUG] SubscriptionRepository.Delete: %+v\n", subscription)
	
	result := r.db.Delete(subscription)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when deleting subscription: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Println("[DEBUG] Deleted subscription successfully")
	return nil
}

func (r *SubscriptionRepository) GetSubscriptionsForUpdates(frequency string) ([]models.Subscription, error) {
	fmt.Printf("[DEBUG] SubscriptionRepository.GetSubscriptionsForUpdates: frequency=%s\n", frequency)
	
	var subscriptions []models.Subscription
	result := r.db.Where("frequency = ? AND confirmed = ?", frequency, true).Find(&subscriptions)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when getting subscriptions for updates: %v\n", result.Error)
		return nil, result.Error
	}
	
	fmt.Printf("[DEBUG] Found %d subscriptions for frequency: %s\n", len(subscriptions), frequency)
	return subscriptions, nil
}

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) CreateToken(subscriptionID uint, tokenType string, expiresIn time.Duration) (*models.Token, error) {
	fmt.Printf("[DEBUG] TokenRepository.CreateToken: subscriptionID=%d, type=%s, expiresIn=%v\n", 
		subscriptionID, tokenType, expiresIn)
	
	token := &models.Token{
		Token:          uuid.New().String(),
		SubscriptionID: subscriptionID,
		Type:           tokenType,
		ExpiresAt:      time.Now().Add(expiresIn),
	}
	
	result := r.db.Create(token)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when creating token: %v\n", result.Error)
		return nil, result.Error
	}
	
	fmt.Printf("[DEBUG] Created token: %s, ID: %d\n", token.Token, token.ID)
	return token, nil
}

func (r *TokenRepository) FindByToken(tokenStr string) (*models.Token, error) {
	fmt.Printf("[DEBUG] TokenRepository.FindByToken: token=%s\n", tokenStr)
	
	var token models.Token
	result := r.db.Where("token = ? AND expires_at > ?", tokenStr, time.Now()).First(&token)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when finding token: %v\n", result.Error)
		return nil, result.Error
	}
	
	fmt.Printf("[DEBUG] Found token: %+v\n", token)
	return &token, nil
}

func (r *TokenRepository) DeleteToken(token *models.Token) error {
	fmt.Printf("[DEBUG] TokenRepository.DeleteToken: %+v\n", token)
	
	result := r.db.Delete(token)
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when deleting token: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Println("[DEBUG] Deleted token successfully")
	return nil
}

func (r *TokenRepository) DeleteExpiredTokens() error {
	fmt.Println("[DEBUG] TokenRepository.DeleteExpiredTokens called")
	
	result := r.db.Where("expires_at < ?", time.Now()).Delete(&models.Token{})
	if result.Error != nil {
		fmt.Printf("[ERROR] Database error when deleting expired tokens: %v\n", result.Error)
		return result.Error
	}
	
	fmt.Printf("[DEBUG] Deleted %d expired tokens\n", result.RowsAffected)
	return nil
}