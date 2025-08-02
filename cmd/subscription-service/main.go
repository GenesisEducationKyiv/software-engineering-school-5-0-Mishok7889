package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/subscription/app"
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
	if err := godotenv.Load(); err != nil {
		slog.Info(logNoEnvFile)
	}

	if err := run(); err != nil {
		slog.Error(logServiceFailed, "error", err)
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

	subscriptionApp, err := app.NewSubscriptionApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, subscriptionApp)

	slog.Info(logServiceStarting, "port", cfg.Services.Subscription.Port)

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

func setupGracefulShutdown(cancel context.CancelFunc, app *app.SubscriptionApplication) {
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

		slog.Info(logShutdownComplete)
	}()
}
