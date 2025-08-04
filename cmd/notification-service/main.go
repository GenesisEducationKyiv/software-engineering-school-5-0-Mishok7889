package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/notification/app"
	"weatherapi.app/pkg/logger"
)

const (
	shutdownTimeoutSeconds = 30
	exitCodeError          = 1

	logNoEnvFile        = "No .env file found or error loading it"
	logServiceFailed    = "Notification service failed"
	logServiceStarting  = "Starting Notification Service"
	logServiceShutdown  = "Shutting down Notification Service..."
	logShutdownSignal   = "Received shutdown signal..."
	logShutdownError    = "Error during graceful shutdown"
	logShutdownComplete = "Notification service shutdown complete"

	errLoadConfig              = "load configuration: %w"
	errCreateApp               = "create notification application: %w"
	errInvalidConfig           = "invalid configuration: %w"
	errStartServer             = "start notification service: %w"
	errConfigNil               = "configuration cannot be nil"
	errNotificationPortInvalid = "notification service port must be positive"
	errNotificationHostEmpty   = "notification service host cannot be empty"
	errBrokerURLEmpty          = "message broker URL cannot be empty"
)

func main() {
	_ = godotenv.Load() // Ignore error if .env file doesn't exist

	if err := run(); err != nil {
		log := logger.NewServiceLogger(
			logger.NotificationServiceName,
			getEnvironment(),
			isProduction(),
		)
		log.LogCriticalError(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadNotificationServiceConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf(errInvalidConfig, err)
	}

	// Initialize service logger
	log := logger.NewServiceLogger(
		logger.NotificationServiceName,
		getEnvironment(),
		isProduction(),
	).WithComponent("main")

	log.Info("configuration loaded successfully",
		"service", logger.NotificationServiceName,
		"port", cfg.Services.Notification.Port,
		"host", cfg.Services.Notification.Host,
		"broker_url", cfg.MessageBroker.URL,
		"weather_service", fmt.Sprintf("%s:%d", cfg.Services.Weather.Host, cfg.Services.Weather.Port),
		"user_service", fmt.Sprintf("%s:%d", cfg.Services.User.Host, cfg.Services.User.Port),
	)

	notificationApp, err := app.NewNotificationApplicationWithLogger(cfg, log)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	// Add Prometheus metrics server in background
	go func() {
		metricsFactory := infrastructure.NewMetricsFactory(
			logger.NotificationServiceName,
			getServiceVersion(),
			log.WithComponent("metrics"),
		)

		prometheusAdapter := metricsFactory.CreatePrometheusAdapter(nil)

		metricsServer := metricsFactory.CreateDedicatedMetricsServer(
			infrastructure.MetricsServerConfig{
				Port: 9084, // Dedicated metrics port
				Host: "0.0.0.0",
			},
			prometheusAdapter,
		)

		log.Info("starting notification service metrics server",
			"metrics_port", 9084,
			"metrics_endpoint", "http://localhost:9084/metrics",
		)

		if err := metricsServer.ListenAndServe(); err != nil {
			log.Error("metrics server failed", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, notificationApp, log)

	log.Info(logServiceStarting, "port", cfg.Services.Notification.Port)

	if err := notificationApp.Start(ctx); err != nil {
		return fmt.Errorf(errStartServer, err)
	}

	<-ctx.Done()
	log.Info(logServiceShutdown)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	if err := notificationApp.Shutdown(shutdownCtx); err != nil {
		log.Error(logShutdownError, "error", err)
		return err
	}

	log.Info(logShutdownComplete)
	return nil
}

func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New(errConfigNil)
	}
	if cfg.Services.Notification.Host == "" {
		return errors.New(errNotificationHostEmpty)
	}
	if cfg.Services.Notification.Port <= 0 {
		return errors.New(errNotificationPortInvalid)
	}
	if cfg.MessageBroker.URL == "" {
		return errors.New(errBrokerURLEmpty)
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.NotificationApplication, log *logger.Logger) {
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

// getServiceVersion returns the service version from env vars
func getServiceVersion() string {
	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		return version
	}
	return "dev"
}
