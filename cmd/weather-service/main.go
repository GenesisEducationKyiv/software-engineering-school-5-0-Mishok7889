package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	weatherpb "weatherapi.app/api/proto/weather"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/weather/app"
)

const (
	// Application constants
	shutdownTimeoutSeconds = 30
	tcpPortFormat          = ":%d"
	exitCodeError          = 1

	// Log messages
	logNoEnvFile        = "No .env file found or error loading it"
	logServiceFailed    = "Weather service failed"
	logServiceStarting  = "Starting Weather Service"
	logServiceShutdown  = "Shutting down Weather Service..."
	logShutdownSignal   = "Received shutdown signal..."
	logShutdownError    = "Error during graceful shutdown"
	logShutdownComplete = "Weather service shutdown complete"

	// Error messages
	errLoadConfig   = "load configuration: %w"
	errCreateApp    = "create weather application: %w"
	errFailedListen = "failed to listen: %w"
	errGRPCServer   = "gRPC server error: %w"
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
		return fmt.Errorf("invalid configuration: %w", err)
	}

	weatherApp, err := app.NewWeatherApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, weatherApp)

	slog.Info(logServiceStarting, "port", cfg.Services.Weather.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf(tcpPortFormat, cfg.Services.Weather.Port))
	if err != nil {
		return fmt.Errorf(errFailedListen, err)
	}

	grpcServer := grpc.NewServer()
	weatherpb.RegisterWeatherServiceServer(grpcServer, weatherApp.GetGRPCHandler())

	errChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf(errGRPCServer, err)
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info(logServiceShutdown)
		grpcServer.GracefulStop()
		if err := weatherApp.Shutdown(ctx); err != nil {
			slog.Error(logShutdownError, "error", err)
		}
		slog.Info(logShutdownComplete)
		return nil
	case err := <-errChan:
		return err
	}
}

func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if cfg.Services.Weather.Port <= 0 {
		return fmt.Errorf("weather service port must be positive")
	}
	if cfg.Services.Weather.Host == "" {
		return fmt.Errorf("weather service host cannot be empty")
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.WeatherApplication) {
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
