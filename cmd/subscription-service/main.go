package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/subscription/app"
	"weatherapi.app/pkg/logger"
)

const (
	shutdownTimeoutSeconds = 30
	exitCodeError          = 1

	logNoEnvFile        = "No .env file found or error loading it"
	logServiceFailed    = "Subscription service failed"
	logServiceStarting  = "Starting Subscription Service"
	logServiceShutdown  = "Shutting down Subscription Service..."
	logShutdownSignal   = "Received shutdown signal..."
	logShutdownError    = "Error during graceful shutdown"
	logShutdownComplete = "Subscription service shutdown complete"

	errLoadConfig     = "load configuration: %w"
	errCreateApp      = "create subscription application: %w"
	errInvalidConfig  = "invalid configuration: %w"
	errConfigNil      = "configuration cannot be nil"
	errSubPortInvalid = "subscription service port must be positive"
	errSubHostEmpty   = "subscription service host cannot be empty"
	errStartServer    = "start HTTP server: %w"
)

func main() {
	_ = godotenv.Load() // Ignore error if .env file doesn't exist

	if err := run(); err != nil {
		log := logger.NewServiceLogger(
			logger.SubscriptionServiceName,
			getEnvironment(),
			isProduction(),
		)
		log.LogCriticalError(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadSubscriptionServiceConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf(errInvalidConfig, err)
	}

	// Initialize service logger
	log := logger.NewServiceLogger(
		logger.SubscriptionServiceName,
		getEnvironment(),
		isProduction(),
	).WithComponent("main")

	log.Info("configuration loaded successfully",
		"service", logger.SubscriptionServiceName,
		"port", cfg.Services.Subscription.Port,
		"host", cfg.Services.Subscription.Host,
		"database", cfg.SubscriptionDB.Name,
		"user_service", fmt.Sprintf("%s:%d", cfg.Services.User.Host, cfg.Services.User.Port),
		"notification_service", fmt.Sprintf("%s:%d", cfg.Services.Notification.Host, cfg.Services.Notification.Port),
	)

	subscriptionApp, err := app.NewSubscriptionApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, subscriptionApp, log)

	log.Info(logServiceStarting, "port", cfg.Services.Subscription.Port)

	if err := subscriptionApp.Start(ctx); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf(errStartServer, err)
	}

	return nil
}

func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New(errConfigNil)
	}
	if cfg.Services.Subscription.Port <= 0 {
		return errors.New(errSubPortInvalid)
	}
	if cfg.Services.Subscription.Host == "" {
		return errors.New(errSubHostEmpty)
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.SubscriptionApplication, log *logger.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info(logShutdownSignal)
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
		defer shutdownCancel()

		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Error(logShutdownError, "error", err)
		}

		log.Info(logShutdownComplete)
	}()
}

// getEnvironment returns the current environment from env vars
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		return "development"
	}
	return env
}

// isProduction determines if we're running in production
func isProduction() bool {
	env := getEnvironment()
	return env == "production" || env == "prod"
}
