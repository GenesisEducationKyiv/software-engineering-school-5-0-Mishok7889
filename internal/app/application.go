package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"weatherapi.app/internal/adapters/api"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/notification"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
)

type Application struct {
	config *config.Config

	// Use Cases
	weatherUseCase      *weather.UseCase
	subscriptionUseCase *subscription.UseCase
	notificationUseCase *notification.UseCase

	// Adapters
	httpServer *http.Server
	router     *gin.Engine

	// Infrastructure
	ports        *ports.ApplicationPorts
	depContainer *DependencyContainer
	stopChan     chan struct{}
}

// validateFrequency validates the frequency enum value
func validateFrequency(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	freq := subscription.FrequencyFromString(value)
	return freq == subscription.FrequencyHourly || freq == subscription.FrequencyDaily
}

func NewApplication() (*Application, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	app := &Application{
		config:   cfg,
		stopChan: make(chan struct{}),
	}

	if err := app.initializePorts(); err != nil {
		return nil, fmt.Errorf("initialize ports: %w", err)
	}

	if err := app.initializeUseCases(); err != nil {
		return nil, fmt.Errorf("initialize use cases: %w", err)
	}

	if err := app.initializeAdapters(); err != nil {
		return nil, fmt.Errorf("initialize adapters: %w", err)
	}

	return app, nil
}

func (a *Application) initializePorts() error {
	slog.Info("Initializing application ports...")

	deps, err := NewDependencyContainer(DependencyConfig{
		Database: a.config.Database,
		Weather:  a.config.Weather,
		Email:    a.config.Email,
		Cache:    a.config.Cache,
	}, a.config)
	if err != nil {
		return fmt.Errorf("create dependency container: %w", err)
	}

	a.ports = deps.ApplicationPorts()
	a.depContainer = deps
	slog.Info("Application ports initialized successfully")
	return nil
}

func (a *Application) initializeUseCases() error {
	slog.Info("Initializing use cases...")

	weatherUseCase, err := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: a.ports.Weather.Provider,
		Cache:           a.ports.Weather.Cache,
		Config:          a.ports.Infrastructure.ConfigProvider,
		Logger:          a.ports.Infrastructure.Logger,
		Metrics:         a.ports.Weather.Metrics,
	})
	if err != nil {
		return fmt.Errorf("create weather use case: %w", err)
	}
	a.weatherUseCase = weatherUseCase

	subscriptionUseCase, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo: a.ports.Subscription.Repository,
		TokenRepo:        a.ports.Infrastructure.TokenRepo,
		EmailProvider:    a.ports.Notification.EmailProvider,
		Config:           a.ports.Infrastructure.ConfigProvider,
		Logger:           a.ports.Infrastructure.Logger,
	})
	if err != nil {
		return fmt.Errorf("create subscription use case: %w", err)
	}
	a.subscriptionUseCase = subscriptionUseCase

	// Create service adapters
	weatherService := CreateWeatherServiceFromUseCase(a.weatherUseCase)
	a.depContainer.SetWeatherService(weatherService)

	subscriptionService := a.depContainer.CreateSubscriptionServiceFromRepository()
	a.depContainer.SetSubscriptionService(subscriptionService)

	// Create notification use case with service ports
	notificationUseCase, err := notification.NewUseCase(notification.UseCaseDependencies{
		WeatherService:      weatherService,
		SubscriptionService: subscriptionService,
		EmailProvider:       a.ports.Notification.EmailProvider,
		TokenRepo:           a.ports.Infrastructure.TokenRepo,
		SubscriptionRepo:    a.ports.Subscription.Repository,
		Config:              a.ports.Infrastructure.ConfigProvider,
		Logger:              a.ports.Infrastructure.Logger,
	})
	if err != nil {
		return fmt.Errorf("create notification use case: %w", err)
	}
	a.notificationUseCase = notificationUseCase

	slog.Info("Use cases initialized successfully")
	return nil
}

func (a *Application) initializeAdapters() error {
	slog.Info("Initializing adapters...")

	// Register custom validator for Frequency enum
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("frequency", validateFrequency); err != nil {
			slog.Warn("Failed to register frequency validator", "error", err)
		}
	}

	// Create metrics collector for HTTP adapter
	metricsCollector := infrastructure.NewMetricsCollectorAdapter(infrastructure.MetricsCollectorConfig{
		WeatherMetrics: a.ports.Weather.Metrics,
		CacheMetrics:   a.ports.Infrastructure.CacheMetrics,
	})

	// Create health checkers
	databaseHealthChecker := infrastructure.NewDatabaseHealthChecker(a.ports.Infrastructure.Database.(*gorm.DB))
	weatherAPIHealthChecker := infrastructure.NewWeatherAPIHealthChecker(a.ports.Weather.Provider)
	emailHealthChecker := infrastructure.NewEmailHealthChecker(a.ports.Infrastructure.ConfigProvider.GetEmailConfig())

	// Create system health checker
	systemHealthChecker := infrastructure.NewSystemHealthChecker(infrastructure.SystemHealthCheckerConfig{
		DatabaseChecker:   databaseHealthChecker,
		WeatherAPIChecker: weatherAPIHealthChecker,
		EmailChecker:      emailHealthChecker,
		ConfigProvider:    a.ports.Infrastructure.ConfigProvider,
	})

	// Create HTTP server adapter with proper dependency injection
	httpAdapter, err := api.NewHTTPServerAdapter(api.ServerOptions{
		Config: api.ServerConfig{
			Port: a.config.Server.Port,
		},
		WeatherUseCase:      a.weatherUseCase,
		SubscriptionUseCase: a.subscriptionUseCase,
		MetricsCollector:    metricsCollector,
		SystemHealthChecker: systemHealthChecker,
		Logger:              a.ports.Infrastructure.Logger,
	})
	if err != nil {
		return fmt.Errorf("create HTTP adapter: %w", err)
	}

	// Store router for testing access
	a.router = httpAdapter.GetRouter()

	// Create HTTP server
	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      httpAdapter.GetRouter(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Adapters initialized successfully")
	return nil
}

func (a *Application) Start(ctx context.Context) error {
	a.ports.Infrastructure.Logger.Info("Starting application...")

	go a.startScheduler(ctx)

	a.ports.Infrastructure.Logger.Info("Starting HTTP server", ports.F("port", a.config.Server.Port))
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

func (a *Application) startScheduler(ctx context.Context) {
	a.ports.Infrastructure.Logger.Info("Starting notification scheduler...")

	hourlyTicker := time.NewTicker(time.Duration(a.config.Scheduler.HourlyInterval) * time.Minute)
	dailyTicker := time.NewTicker(time.Duration(a.config.Scheduler.DailyInterval) * time.Minute)

	defer hourlyTicker.Stop()
	defer dailyTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.ports.Infrastructure.Logger.Info("Scheduler stopped due to context cancellation")
			return
		case <-a.stopChan:
			a.ports.Infrastructure.Logger.Info("Scheduler stopped")
			return
		case <-hourlyTicker.C:
			params := notification.SendWeatherUpdateParams{
				Frequency: "hourly",
			}
			if err := a.notificationUseCase.SendWeatherUpdates(ctx, params); err != nil {
				a.ports.Infrastructure.Logger.Error("Error sending hourly notifications", ports.F("error", err))
			}
		case <-dailyTicker.C:
			params := notification.SendWeatherUpdateParams{
				Frequency: "daily",
			}
			if err := a.notificationUseCase.SendWeatherUpdates(ctx, params); err != nil {
				a.ports.Infrastructure.Logger.Error("Error sending daily notifications", ports.F("error", err))
			}
		}
	}
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.ports.Infrastructure.Logger.Info("Shutting down application...")

	close(a.stopChan)

	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.ports.Infrastructure.Logger.Error("Error shutting down HTTP server", ports.F("error", err))
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if a.ports != nil && a.ports.Infrastructure.Database != nil {
		if gormDB, ok := a.ports.Infrastructure.Database.(*gorm.DB); ok {
			if db, err := gormDB.DB(); err == nil {
				if err := db.Close(); err != nil {
					a.ports.Infrastructure.Logger.Warn("Error closing database", ports.F("error", err))
				}
			}
		}
	}

	a.ports.Infrastructure.Logger.Info("Application shutdown complete")
	return nil
}

// Config returns the application configuration
func (a *Application) Config() *config.Config {
	return a.config
}

// GetRouter returns the Gin router for testing
func (a *Application) GetRouter() *gin.Engine {
	return a.router
}

// GetWeatherUseCase returns the weather use case for testing
func (a *Application) GetWeatherUseCase() *weather.UseCase {
	return a.weatherUseCase
}

// GetSubscriptionUseCase returns the subscription use case for testing
func (a *Application) GetSubscriptionUseCase() *subscription.UseCase {
	return a.subscriptionUseCase
}

// GetNotificationUseCase returns the notification use case for testing
func (a *Application) GetNotificationUseCase() *notification.UseCase {
	return a.notificationUseCase
}

// NewApplicationWithDependencies creates an application with provided dependencies (for testing)
func NewApplicationWithDependencies(cfg *config.Config, depContainer *DependencyContainer) (*Application, error) {
	app := &Application{
		config:       cfg,
		stopChan:     make(chan struct{}),
		ports:        depContainer.ApplicationPorts(),
		depContainer: depContainer,
	}

	if err := app.initializeUseCases(); err != nil {
		return nil, fmt.Errorf("initialize use cases: %w", err)
	}

	if err := app.initializeAdapters(); err != nil {
		return nil, fmt.Errorf("initialize adapters: %w", err)
	}

	return app, nil
}
