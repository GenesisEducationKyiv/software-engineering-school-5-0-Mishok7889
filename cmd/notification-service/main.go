package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/notification/app"
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
	if err := godotenv.Load(); err != nil {
		slog.Info(logNoEnvFile)
	}

	if err := run(); err != nil {
		slog.Error(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf(errInvalidConfig, err)
	}

	notificationApp, err := app.NewNotificationApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, notificationApp)

	slog.Info(logServiceStarting, "port", cfg.Services.Notification.Port)

	if err := notificationApp.Start(ctx); err != nil {
		return fmt.Errorf(errStartServer, err)
	}

	<-ctx.Done()
	slog.Info(logServiceShutdown)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	if err := notificationApp.Shutdown(shutdownCtx); err != nil {
		slog.Error(logShutdownError, "error", err)
		return err
	}

	slog.Info(logShutdownComplete)
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

func setupGracefulShutdown(cancel context.CancelFunc, app *app.NotificationApplication) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		slog.Info(logShutdownSignal)
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
		defer shutdownCancel()

		if err := app.Shutdown(shutdownCtx); err != nil {
			slog.Error(logShutdownError, "error", err)
		}
	}()
}
