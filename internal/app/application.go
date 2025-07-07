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
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/notification"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
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
	ports    *ports.ApplicationPorts
	stopChan chan struct{}
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
	slog.Info("Application ports initialized successfully")
	return nil
}

func (a *Application) initializeUseCases() error {
	slog.Info("Initializing use cases...")

	weatherUseCase, err := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: a.ports.WeatherProvider,
		Cache:           a.ports.WeatherCache,
		Config:          a.ports.ConfigProvider,
		Logger:          a.ports.Logger,
		Metrics:         a.ports.WeatherMetrics,
	})
	if err != nil {
		return fmt.Errorf("create weather use case: %w", err)
	}
	a.weatherUseCase = weatherUseCase

	subscriptionUseCase, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo: a.ports.SubscriptionRepository,
		TokenRepo:        a.ports.TokenRepository,
		EmailProvider:    a.ports.EmailProvider,
		Config:           a.ports.ConfigProvider,
		Logger:           a.ports.Logger,
	})
	if err != nil {
		return fmt.Errorf("create subscription use case: %w", err)
	}
	a.subscriptionUseCase = subscriptionUseCase

	notificationUseCase, err := notification.NewUseCase(notification.UseCaseDependencies{
		SubscriptionRepo: a.ports.SubscriptionRepository,
		TokenRepo:        a.ports.TokenRepository,
		EmailProvider:    a.ports.EmailProvider,
		WeatherUseCase:   a.weatherUseCase,
		Config:           a.ports.ConfigProvider,
		Logger:           a.ports.Logger,
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

	// Create simple HTTP server using Gin directly for now
	server := gin.Default()

	// Store the router for testing access
	a.router = server

	// Serve static files
	server.GET("/", func(c *gin.Context) {
		c.File("public/index.html")
	})
	server.Static("/static", "./public")

	// API routes
	api := server.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// Weather endpoint
		api.GET("/weather", func(c *gin.Context) {
			city := c.Query("city")
			if city == "" {
				c.JSON(400, gin.H{"error": "city parameter is required"})
				return
			}

			weatherRequest := weather.WeatherRequest{City: city}
			weatherData, err := a.weatherUseCase.GetWeather(c.Request.Context(), weatherRequest)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to get weather data"})
				return
			}

			c.JSON(200, gin.H{
				"temperature": weatherData.Temperature,
				"humidity":    weatherData.Humidity,
				"description": weatherData.Description,
			})
		})

		// Debug endpoint
		api.GET("/debug", func(c *gin.Context) {
			// Check database connection
			dbConnected := true
			if gormDB, ok := a.ports.Database.(*gorm.DB); ok {
				if db, err := gormDB.DB(); err == nil {
					if err := db.Ping(); err != nil {
						dbConnected = false
					}
				} else {
					dbConnected = false
				}
			}

			// Check weather API connection (simplified)
			weatherConnected := true // Assume connected for now

			response := gin.H{
				"database": gin.H{
					"connected": dbConnected,
				},
				"weatherAPI": gin.H{
					"connected": weatherConnected,
				},
				"smtp": gin.H{
					"host": a.config.Email.SMTPHost,
					"port": fmt.Sprintf("%d", a.config.Email.SMTPPort),
				},
				"config": gin.H{
					"appBaseURL": a.config.AppBaseURL,
				},
			}

			c.JSON(200, response)
		})

		// Metrics endpoint
		api.GET("/metrics", func(c *gin.Context) {
			response := gin.H{
				"cache": gin.H{
					"type": "memory", // or from config
				},
				"provider_info": gin.H{
					"active_providers": []string{"weatherapi", "openweathermap"},
				},
				"endpoints": gin.H{
					"prometheus_metrics": "/metrics",
					"cache_metrics":      "/api/cache/metrics",
				},
			}

			c.JSON(200, response)
		})

		// Subscription endpoints
		api.POST("/subscribe", func(c *gin.Context) {
			// Use a separate struct for HTTP binding to avoid parsing issues
			var httpReq struct {
				Email     string `json:"email" form:"email" binding:"required,email"`
				City      string `json:"city" form:"city" binding:"required"`
				Frequency string `json:"frequency" form:"frequency" binding:"required"`
			}

			// Log the raw request data for debugging
			a.ports.Logger.Debug("Received subscription request",
				ports.F("content-type", c.GetHeader("Content-Type")),
				ports.F("method", c.Request.Method))

			if err := c.ShouldBind(&httpReq); err != nil {
				a.ports.Logger.Error("Request binding error", ports.F("error", err))
				c.JSON(400, gin.H{"error": "Invalid request format"})
				return
			}

			// Convert string frequency to domain type
			frequency := subscription.FrequencyFromString(httpReq.Frequency)
			if frequency == subscription.FrequencyUnknown {
				a.ports.Logger.Error("Invalid frequency value", ports.F("frequency", httpReq.Frequency))
				c.JSON(400, gin.H{"error": "Invalid request format"})
				return
			}

			// Log the parsed request for debugging
			a.ports.Logger.Debug("Parsed subscription request",
				ports.F("email", httpReq.Email),
				ports.F("city", httpReq.City),
				ports.F("frequency", frequency))

			// Convert to SubscribeParams
			subscribeParams := subscription.SubscribeParams{
				Email:     httpReq.Email,
				City:      httpReq.City,
				Frequency: frequency,
			}

			if err := a.subscriptionUseCase.Subscribe(c.Request.Context(), subscribeParams); err != nil {
				a.ports.Logger.Error("Failed to create subscription", ports.F("error", err), ports.F("email", httpReq.Email))

				// Map domain errors to HTTP status codes
				if errors.IsAlreadyExistsError(err) {
					c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
					return
				}
				if errors.IsValidationError(err) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
					return
				}
				if errors.IsNotFoundError(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}

				// Default to internal server error
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}

			c.JSON(200, gin.H{"message": "Subscription successful. Confirmation email sent."})
		})

		api.GET("/confirm/:token", func(c *gin.Context) {
			token := c.Param("token")
			confirmParams := subscription.ConfirmParams{
				Token: token,
			}

			if err := a.subscriptionUseCase.ConfirmSubscription(c.Request.Context(), confirmParams); err != nil {
				// Map specific errors to appropriate HTTP status codes
				if errors.IsTokenError(err) {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if errors.IsNotFoundError(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				if errors.IsAlreadyExistsError(err) {
					c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
					return
				}
				if errors.IsValidationError(err) {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Default to internal server error
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}

			c.JSON(200, gin.H{"message": "Subscription confirmed successfully"})
		})

		api.GET("/unsubscribe/:token", func(c *gin.Context) {
			token := c.Param("token")
			unsubscribeParams := subscription.UnsubscribeParams{
				Token: token,
			}

			if err := a.subscriptionUseCase.Unsubscribe(c.Request.Context(), unsubscribeParams); err != nil {
				// Map specific errors to appropriate HTTP status codes
				if errors.IsTokenError(err) {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if errors.IsNotFoundError(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				if errors.IsValidationError(err) {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Default to internal server error
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}

			c.JSON(200, gin.H{"message": "Unsubscribed successfully"})
		})
	}

	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.Server.Port),
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Adapters initialized successfully")
	return nil
}

func (a *Application) Start(ctx context.Context) error {
	slog.Info("Starting application...")

	// Start background scheduler
	go a.startScheduler(ctx)

	// Start HTTP server
	slog.Info("Starting HTTP server", "port", a.config.Server.Port)
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

func (a *Application) startScheduler(ctx context.Context) {
	slog.Info("Starting notification scheduler...")

	hourlyTicker := time.NewTicker(time.Duration(a.config.Scheduler.HourlyInterval) * time.Minute)
	dailyTicker := time.NewTicker(time.Duration(a.config.Scheduler.DailyInterval) * time.Minute)

	defer hourlyTicker.Stop()
	defer dailyTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Scheduler stopped due to context cancellation")
			return
		case <-a.stopChan:
			slog.Info("Scheduler stopped")
			return
		case <-hourlyTicker.C:
			params := notification.SendWeatherUpdateParams{
				Frequency: subscription.FrequencyHourly,
			}
			if err := a.notificationUseCase.SendWeatherUpdates(ctx, params); err != nil {
				slog.Error("Error sending hourly notifications", "error", err)
			}
		case <-dailyTicker.C:
			params := notification.SendWeatherUpdateParams{
				Frequency: subscription.FrequencyDaily,
			}
			if err := a.notificationUseCase.SendWeatherUpdates(ctx, params); err != nil {
				slog.Error("Error sending daily notifications", "error", err)
			}
		}
	}
}

func (a *Application) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down application...")

	// Signal scheduler to stop
	close(a.stopChan)

	// Shutdown HTTP server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Error shutting down HTTP server", "error", err)
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	// Close database connections and other resources
	if a.ports != nil && a.ports.Database != nil {
		// Type assert to *gorm.DB to access DB() method
		if gormDB, ok := a.ports.Database.(*gorm.DB); ok {
			if db, err := gormDB.DB(); err == nil {
				if err := db.Close(); err != nil {
					slog.Warn("Error closing database", "error", err)
				}
			}
		}
	}

	slog.Info("Application shutdown complete")
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
		config:   cfg,
		stopChan: make(chan struct{}),
		ports:    depContainer.ApplicationPorts(),
	}

	if err := app.initializeUseCases(); err != nil {
		return nil, fmt.Errorf("initialize use cases: %w", err)
	}

	if err := app.initializeAdapters(); err != nil {
		return nil, fmt.Errorf("initialize adapters: %w", err)
	}

	return app, nil
}
