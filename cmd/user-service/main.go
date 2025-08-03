package main

import (
	"context"
	"errors"
	"fmt"
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
	"weatherapi.app/pkg/logger"
)

const (
	shutdownTimeoutSeconds = 30
	tcpPortFormat          = ":%d"
	exitCodeError          = 1

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
	_ = godotenv.Load() // Ignore error if .env file doesn't exist

	if err := run(); err != nil {
		log := logger.NewServiceLogger(
			logger.UserServiceName,
			getEnvironment(),
			isProduction(),
		)
		log.LogCriticalError(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadUserServiceConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf(errInvalidConfig, err)
	}

	// Initialize service logger
	log := logger.NewServiceLogger(
		logger.UserServiceName,
		getEnvironment(),
		isProduction(),
	).WithComponent("main")

	log.Info("configuration loaded successfully",
		"service", logger.UserServiceName,
		"port", cfg.Services.User.Port,
		"host", cfg.Services.User.Host,
		"database", cfg.UserDB.Name,
	)

	userApp, err := app.NewUserApplication(cfg)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, userApp, log)

	log.Info(logServiceStarting, "port", cfg.Services.User.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf(tcpPortFormat, cfg.Services.User.Port))
	if err != nil {
		return fmt.Errorf(errFailedListen, err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, userApp.GetGRPCHandler())

	log.Info("gRPC server registered, starting to serve",
		"address", lis.Addr().String(),
	)

	errChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf(errGRPCServer, err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Info(logServiceShutdown)
		grpcServer.GracefulStop()
		if err := userApp.Shutdown(ctx); err != nil {
			log.Error(logShutdownError, "error", err)
		}
		log.Info(logShutdownComplete)
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

func setupGracefulShutdown(cancel context.CancelFunc, app *app.UserApplication, log *logger.Logger) {
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
