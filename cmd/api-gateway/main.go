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
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/gateway/app"
	"weatherapi.app/pkg/logger"
)

const (
	shutdownTimeoutSeconds = 30
	exitCodeError          = 1

	logServiceFailed    = "API Gateway failed"
	logServiceStarting  = "Starting API Gateway"
	logServiceShutdown  = "Shutting down API Gateway..."
	logShutdownSignal   = "Received shutdown signal..."
	logShutdownError    = "Error during graceful shutdown"
	logShutdownComplete = "API Gateway shutdown complete"

	errLoadConfig         = "load configuration: %w"
	errCreateApp          = "create gateway application: %w"
	errInvalidConfig      = "invalid configuration: %w"
	errStartServer        = "start gateway service: %w"
	errConfigNil          = "configuration cannot be nil"
	errGatewayPortInvalid = "gateway port must be positive"
	errGatewayHostEmpty   = "gateway host cannot be empty"
)

func main() {
	_ = godotenv.Load() // Ignore error if .env file doesn't exist

	if err := run(); err != nil {
		log := logger.NewServiceLogger(
			logger.APIGatewayServiceName,
			getEnvironment(),
			isProduction(),
		)
		log.LogCriticalError(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadGatewayConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf(errInvalidConfig, err)
	}

	// Initialize service logger
	log := logger.NewServiceLogger(
		logger.APIGatewayServiceName,
		getEnvironment(),
		isProduction(),
	).WithComponent("main")

	log.Info("configuration loaded successfully",
		"service", logger.APIGatewayServiceName,
		"port", cfg.Services.Gateway.Port,
		"host", cfg.Services.Gateway.Host,
		"weather_service", fmt.Sprintf("%s:%d", cfg.Services.Weather.Host, cfg.Services.Weather.Port),
		"user_service", fmt.Sprintf("%s:%d", cfg.Services.User.Host, cfg.Services.User.Port),
		"subscription_service", fmt.Sprintf("%s:%d", cfg.Services.Subscription.Host, cfg.Services.Subscription.Port),
	)

	gatewayApp, err := app.NewGatewayApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, gatewayApp, log)

	log.Info(logServiceStarting, "port", cfg.Services.Gateway.Port)

	if err := gatewayApp.Start(ctx); err != nil {
		return fmt.Errorf(errStartServer, err)
	}

	<-ctx.Done()
	log.Info(logServiceShutdown)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	if err := gatewayApp.Shutdown(shutdownCtx); err != nil {
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
	if cfg.Services.Gateway.Host == "" {
		return errors.New(errGatewayHostEmpty)
	}
	if cfg.Services.Gateway.Port <= 0 {
		return errors.New(errGatewayPortInvalid)
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.GatewayApplication, log *logger.Logger) {
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
