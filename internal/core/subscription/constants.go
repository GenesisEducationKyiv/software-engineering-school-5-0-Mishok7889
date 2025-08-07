package subscription

import "time"

// Token expiration durations
const (
	ConfirmationTokenTTL = 24 * time.Hour
	UnsubscribeTokenTTL  = 365 * 24 * time.Hour
	SubscriptionTTL      = 24 * time.Hour // Unconfirmed subscription expiry
)

// Validation error messages
const (
	ErrEmailRequired         = "email is required"
	ErrEmailEmpty            = "email cannot be empty"
	ErrEmailInvalid          = "invalid email format"
	ErrCityRequired          = "city is required"
	ErrCityEmpty             = "city cannot be empty"
	ErrFrequencyInvalid      = "invalid frequency"
	ErrFrequencyRequired     = "frequency must be valid"
	ErrTokenConfirmEmpty     = "confirmation token is required"
	ErrTokenUnsubEmpty       = "unsubscribe token is required"
	ErrTokenConfirmExpired   = "invalid or expired confirmation token"
	ErrTokenUnsubExpired     = "invalid unsubscribe token"
	ErrTokenInvalidType      = "invalid token type"
	ErrSubscriptionExists    = "already subscribed"
	ErrSubscriptionConfirmed = "subscription is already confirmed"
	ErrSubscriptionNotFound  = "subscription not found"
	ErrSubscriptionIDZero    = "subscription ID cannot be zero"
)

// API paths
const (
	APIPathConfirm     = "/api/confirm/%s"
	APIPathUnsubscribe = "/api/unsubscribe/%s"
)

// Dependency error messages
const (
	ErrRepoRequired                = "subscription repository is required"
	ErrTokenRepoRequired           = "token repository is required"
	ErrGeneratorRequired           = "token generator is required"
	ErrNotificationServiceRequired = "notification service is required"
	ErrConfigRequired              = "config is required"
	ErrLoggerRequired              = "logger is required"
)
