// Package models defines data structures used throughout the application
package models

import (
	"time"

	"gorm.io/gorm"
)

// Subscription represents a user's weather notification subscription
type Subscription struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"index;not null"`
	City      string         `json:"city" gorm:"not null"`
	Frequency string         `json:"frequency" gorm:"not null"`
	Confirmed bool           `json:"confirmed" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Token represents an authentication or verification token
type Token struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Token          string         `json:"token" gorm:"uniqueIndex;not null"`
	SubscriptionID uint           `json:"subscription_id" gorm:"index;not null"`
	Subscription   Subscription   `json:"-" gorm:"foreignKey:SubscriptionID"`
	Type           string         `json:"type" gorm:"not null"` // "confirmation" or "unsubscribe"
	ExpiresAt      time.Time      `json:"expires_at"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// WeatherResponse represents weather data returned from the API
type WeatherResponse struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Description string  `json:"description"`
}

// SubscriptionRequest represents data required to create a subscription
type SubscriptionRequest struct {
	Email     string `json:"email" form:"email" binding:"required,email"`
	City      string `json:"city" form:"city" binding:"required"`
	Frequency string `json:"frequency" form:"frequency" binding:"required,oneof=hourly daily"`
}

// ErrorResponse represents an error message structure for API responses
type ErrorResponse struct {
	Error string `json:"error"`
}
