package scheduler

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/repository"
	"weatherapi.app/service"
)

type Scheduler struct {
	db                  *gorm.DB
	config              *config.Config
	subscriptionRepo    *repository.SubscriptionRepository
	tokenRepo           *repository.TokenRepository
	weatherService      *service.WeatherService
	emailService        *service.EmailService
	subscriptionService *service.SubscriptionService
}

func NewScheduler(db *gorm.DB, config *config.Config) *Scheduler {
	weatherService := service.NewWeatherService(config)
	emailService := service.NewEmailService(config)
	
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	
	subscriptionService := service.NewSubscriptionService(
		db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		config,
	)
	
	return &Scheduler{
		db:                  db,
		config:              config,
		subscriptionRepo:    subscriptionRepo,
		tokenRepo:           tokenRepo,
		weatherService:      weatherService,
		emailService:        emailService,
		subscriptionService: subscriptionService,
	}
}

func (s *Scheduler) Start() {
	go s.scheduleDaily(24*time.Hour, s.cleanupExpiredTokens)
	
	go s.scheduleInterval(time.Duration(s.config.Scheduler.HourlyInterval)*time.Minute, func() {
		if err := s.subscriptionService.SendWeatherUpdate("hourly"); err != nil {
			fmt.Printf("Error sending hourly weather updates: %v\n", err)
		}
	})
	
	go s.scheduleInterval(time.Duration(s.config.Scheduler.DailyInterval)*time.Minute, func() {
		if err := s.subscriptionService.SendWeatherUpdate("daily"); err != nil {
			fmt.Printf("Error sending daily weather updates: %v\n", err)
		}
	})
}

func (s *Scheduler) scheduleInterval(interval time.Duration, job func()) {
	job()
	
	ticker := time.NewTicker(interval)
	for range ticker.C {
		job()
	}
}

func (s *Scheduler) scheduleDaily(interval time.Duration, job func()) {
	job()
	
	ticker := time.NewTicker(interval)
	for range ticker.C {
		job()
	}
}

func (s *Scheduler) cleanupExpiredTokens() {
	if err := s.tokenRepo.DeleteExpiredTokens(); err != nil {
		fmt.Printf("Error cleaning up expired tokens: %v\n", err)
	}
}