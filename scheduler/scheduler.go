package scheduler

import (
	"log"
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
	log.Println("[INFO] Starting scheduler...")

	go s.scheduleDaily(24*time.Hour, s.cleanupExpiredTokens)

	go s.scheduleInterval(time.Duration(s.config.Scheduler.HourlyInterval)*time.Minute, func() {
		log.Println("[INFO] Running hourly weather updates...")
		if err := s.subscriptionService.SendWeatherUpdate("hourly"); err != nil {
			log.Printf("[ERROR] Failed to send hourly weather updates: %v\n", err)
		} else {
			log.Println("[INFO] Hourly weather updates completed successfully")
		}
	})

	go s.scheduleInterval(time.Duration(s.config.Scheduler.DailyInterval)*time.Minute, func() {
		log.Println("[INFO] Running daily weather updates...")
		if err := s.subscriptionService.SendWeatherUpdate("daily"); err != nil {
			log.Printf("[ERROR] Failed to send daily weather updates: %v\n", err)
		} else {
			log.Println("[INFO] Daily weather updates completed successfully")
		}
	})

	log.Println("[INFO] Scheduler started successfully")
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
	log.Println("[INFO] Running expired token cleanup...")
	if err := s.tokenRepo.DeleteExpiredTokens(); err != nil {
		log.Printf("[ERROR] Failed to cleanup expired tokens: %v\n", err)
	} else {
		log.Println("[INFO] Expired token cleanup completed successfully")
	}
}
