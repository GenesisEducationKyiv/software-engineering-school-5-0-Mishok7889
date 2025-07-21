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

// Email templates
const (
	EmailSubjectConfirmation = "Confirm your weather subscription"
	EmailSubjectWelcome      = "Welcome to Weather Updates!"
	EmailSubjectUnsubscribe  = "You have been unsubscribed from weather updates"
)

// Email body templates
const (
	EmailBodyConfirmation = `
		<h2>Confirm Your Weather Subscription</h2>
		<p>Hello!</p>
		<p>Thank you for subscribing to weather updates for <strong>{{.City}}</strong>.</p>
		<p>Please click the link below to confirm your subscription:</p>
		<p><a href="{{.ConfirmURL}}">Confirm Subscription</a></p>
		<p>If you didn't request this subscription, you can safely ignore this email.</p>
	`

	EmailBodyWelcome = `
		<h2>Welcome to Weather Updates!</h2>
		<p>Hello!</p>
		<p>Your subscription for <strong>{{.City}}</strong> weather updates has been confirmed.</p>
		<p>You will receive <strong>{{.Frequency}}</strong> weather updates.</p>
		<p>If you wish to unsubscribe, click <a href="{{.UnsubscribeURL}}">here</a>.</p>
	`

	EmailBodyUnsubscribe = `
		<h2>Unsubscribed Successfully</h2>
		<p>Hello!</p>
		<p>You have been successfully unsubscribed from weather updates for <strong>{{.City}}</strong>.</p>
		<p>We're sorry to see you go!</p>
	`
)

// Frequency string constants
const (
	FrequencyStringHourly  = "hourly"
	FrequencyStringDaily   = "daily"
	FrequencyStringUnknown = "unknown"
)

// API paths
const (
	APIPathConfirm     = "/api/confirm/%s"
	APIPathUnsubscribe = "/api/unsubscribe/%s"
)

// Dependency error messages
const (
	ErrRepoRequired          = "subscription repository is required"
	ErrTokenRepoRequired     = "token repository is required"
	ErrGeneratorRequired     = "token generator is required"
	ErrEmailProviderRequired = "email provider is required"
	ErrEmailBuilderRequired  = "email builder is required"
	ErrConfigRequired        = "config is required"
	ErrLoggerRequired        = "logger is required"
)
