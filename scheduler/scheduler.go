package scheduler

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/repository"
	"weatherapi.app/service"
)

// Scheduler manages periodic tasks for the application
type Scheduler struct {
	db                  *gorm.DB
	config              *config.Config
	tokenRepo           *repository.TokenRepository
	subscriptionService service.NotificationServiceInterface
}

// NewScheduler creates and configures a new task scheduler
func NewScheduler(
	db *gorm.DB,
	config *config.Config,
	subscriptionService service.NotificationServiceInterface,
) *Scheduler {
	tokenRepo := repository.NewTokenRepository(db)

	return &Scheduler{
		db:                  db,
		config:              config,
		tokenRepo:           tokenRepo,
		subscriptionService: subscriptionService,
	}
}

// Start begins the scheduler's operations
func (s *Scheduler) Start() {
	slog.Info("Starting scheduler...")

	go s.scheduleDaily(24*time.Hour, s.cleanupExpiredTokens)

	go s.scheduleInterval(time.Duration(s.config.Scheduler.HourlyInterval)*time.Minute, func() {
		slog.Info("Running hourly weather updates...")
		if err := s.subscriptionService.SendWeatherUpdate("hourly"); err != nil {
			slog.Error("Failed to send hourly weather updates", "error", err)
		} else {
			slog.Info("Hourly weather updates completed successfully")
		}
	})

	go s.scheduleInterval(time.Duration(s.config.Scheduler.DailyInterval)*time.Minute, func() {
		slog.Info("Running daily weather updates...")
		if err := s.subscriptionService.SendWeatherUpdate("daily"); err != nil {
			slog.Error("Failed to send daily weather updates", "error", err)
		} else {
			slog.Info("Daily weather updates completed successfully")
		}
	})

	slog.Info("Scheduler started successfully")
}

func (s *Scheduler) scheduleInterval(interval time.Duration, job func()) {
	job()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		job()
	}
}

func (s *Scheduler) scheduleDaily(interval time.Duration, job func()) {
	job()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		job()
	}
}

func (s *Scheduler) cleanupExpiredTokens() {
	slog.Info("Running expired token cleanup...")
	if err := s.tokenRepo.DeleteExpiredTokens(); err != nil {
		slog.Error("Failed to cleanup expired tokens", "error", err)
	} else {
		slog.Info("Expired token cleanup completed successfully")
	}
}
