package ports

import (
	"context"
	"time"
)

// NotificationStats represents notification statistics
type NotificationStats struct {
	TotalSubscriptions  int64
	HourlySubscriptions int64
	DailySubscriptions  int64
	LastUpdated         time.Time
}

// NotificationService defines the contract for notification operations
type NotificationService interface {
	SendWeatherUpdates(ctx context.Context, frequency string) error
	GetNotificationStats(ctx context.Context) (NotificationStats, error)
	// Email sending methods for subscription service
	SendConfirmationEmail(ctx context.Context, email, city, confirmationURL string) error
	SendWelcomeEmail(ctx context.Context, email, city, frequency, unsubscribeURL string) error
	SendUnsubscribeConfirmationEmail(ctx context.Context, email, city string) error
}

// NotificationScheduler defines the contract for scheduling notifications
type NotificationScheduler interface {
	Schedule(ctx context.Context, frequency string) error
	Stop(ctx context.Context) error
}
