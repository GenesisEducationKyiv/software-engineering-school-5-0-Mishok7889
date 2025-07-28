package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	authpb "weatherapi.app/api/proto/auth"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/user/app"
)

const (
	shutdownTimeoutSeconds = 30
	tcpPortFormat          = ":%d"
	exitCodeError          = 1

	logNoEnvFile        = "No .env file found or error loading it"
	logServiceFailed    = "User service failed"
	logServiceStarting  = "Starting User Service"
	logServiceShutdown  = "Shutting down User Service..."
	logShutdownSignal   = "Received shutdown signal..."
	logShutdownError    = "Error during graceful shutdown"
	logShutdownComplete = "User service shutdown complete"

	errLoadConfig      = "load configuration: %w"
	errCreateApp       = "create user application: %w"
	errFailedListen    = "failed to listen: %w"
	errGRPCServer      = "gRPC server error: %w"
	errInvalidConfig   = "invalid configuration: %w"
	errConfigNil       = "configuration cannot be nil"
	errUserPortInvalid = "user service port must be positive"
	errUserHostEmpty   = "user service host cannot be empty"
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

	userApp, err := app.NewUserApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, userApp)

	slog.Info(logServiceStarting, "port", cfg.Services.User.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf(tcpPortFormat, cfg.Services.User.Port))
	if err != nil {
		return fmt.Errorf(errFailedListen, err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, userApp.GetGRPCHandler())

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
		if err := userApp.Shutdown(ctx); err != nil {
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
		return errors.New(errConfigNil)
	}
	if cfg.Services.User.Port <= 0 {
		return errors.New(errUserPortInvalid)
	}
	if cfg.Services.User.Host == "" {
		return errors.New(errUserHostEmpty)
	}
	return nil
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.UserApplication) {
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
