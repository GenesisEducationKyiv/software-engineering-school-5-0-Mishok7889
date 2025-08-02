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
	"weatherapi.app/internal/services/gateway/app"
)

const (
	shutdownTimeoutSeconds = 30
	exitCodeError          = 1

	logNoEnvFile        = "No .env file found or error loading it"
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
	if err := godotenv.Load(); err != nil {
		slog.Info(logNoEnvFile)
	}

	if err := run(); err != nil {
		slog.Error(logServiceFailed, "error", err)
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

	gatewayApp, err := app.NewGatewayApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, gatewayApp)

	slog.Info(logServiceStarting, "port", cfg.Services.Gateway.Port)

	if err := gatewayApp.Start(ctx); err != nil {
		return fmt.Errorf(errStartServer, err)
	}

	<-ctx.Done()
	slog.Info(logServiceShutdown)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	if err := gatewayApp.Shutdown(shutdownCtx); err != nil {
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
	if cfg.Services.Gateway.Host == "" {
		return errors.New(errGatewayHostEmpty)
	}
	if cfg.Services.Gateway.Port <= 0 {
		return errors.New(errGatewayPortInvalid)
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.GatewayApplication) {
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
