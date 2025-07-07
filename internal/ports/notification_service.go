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
	CleanupExpiredTokens(ctx context.Context) error
	GetNotificationStats(ctx context.Context) (NotificationStats, error)
}

// NotificationScheduler defines the contract for scheduling notifications
type NotificationScheduler interface {
	Schedule(ctx context.Context, frequency string) error
	Stop(ctx context.Context) error
}
