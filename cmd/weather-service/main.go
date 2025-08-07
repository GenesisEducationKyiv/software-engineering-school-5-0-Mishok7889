package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	weatherpb "weatherapi.app/api/proto/weather"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/adapters/middleware"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/services/weather/app"
	"weatherapi.app/pkg/logger"
)

const (
	// Application constants
	shutdownTimeoutSeconds = 30
	tcpPortFormat          = ":%d"
	exitCodeError          = 1

	// Log messages
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
	_ = godotenv.Load() // Ignore error if .env file doesn't exist

	if err := run(); err != nil {
		log := logger.NewServiceLogger(
			logger.WeatherServiceName,
			getEnvironment(),
			isProduction(),
		)
		log.LogCriticalError(logServiceFailed, "error", err)
		os.Exit(exitCodeError)
	}
}

func run() error {
	cfg, err := config.LoadWeatherServiceConfig()
	if err != nil {
		return fmt.Errorf(errLoadConfig, err)
	}

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize service logger
	log := logger.NewServiceLogger(
		logger.WeatherServiceName,
		getEnvironment(),
		isProduction(),
	).WithComponent("main")

	log.Info("configuration loaded successfully",
		"service", logger.WeatherServiceName,
		"port", cfg.Services.Weather.Port,
		"host", cfg.Services.Weather.Host,
		"cache_enabled", cfg.Weather.EnableCache,
		"provider_count", len(cfg.Weather.ProviderOrder),
	)

	weatherApp, err := app.NewWeatherApplicationWithLogger(cfg, log)
	if err != nil {
		return fmt.Errorf(errCreateApp, err)
	}

	// Add Prometheus metrics server in background
	go func() {
		metricsFactory := infrastructure.NewMetricsFactory(
			logger.WeatherServiceName,
			getServiceVersion(),
			log.WithComponent("metrics"),
		)

		prometheusAdapter := metricsFactory.CreatePrometheusAdapter(nil)

		metricsServer := metricsFactory.CreateDedicatedMetricsServer(
			infrastructure.MetricsServerConfig{
				Port: 9081, // Dedicated metrics port
				Host: "0.0.0.0",
			},
			prometheusAdapter,
		)

		log.Info("starting weather service metrics server",
			"metrics_port", 9081,
			"metrics_endpoint", "http://localhost:9081/metrics",
		)

		if err := metricsServer.ListenAndServe(); err != nil {
			log.Error("metrics server failed", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create correlation interceptor for gRPC
	correlationInterceptor := middleware.NewGRPCCorrelationInterceptor(
		log.WithComponent("grpc-interceptor"),
	)

	setupGracefulShutdown(cancel, weatherApp, log)

	log.Info(logServiceStarting, "port", cfg.Services.Weather.Port)

	lis, err := net.Listen("tcp", fmt.Sprintf(tcpPortFormat, cfg.Services.Weather.Port))
	if err != nil {
		return fmt.Errorf(errFailedListen, err)
	}

	grpcServer := grpc.NewServer(
		correlationInterceptor.GRPCServerOptions()...,
	)
	weatherpb.RegisterWeatherServiceServer(grpcServer, weatherApp.GetGRPCHandler())

	log.Info("gRPC server registered with correlation support",
		"address", lis.Addr().String(),
		"correlation_enabled", true,
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
		if err := weatherApp.Shutdown(ctx); err != nil {
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

func setupGracefulShutdown(cancel context.CancelFunc, app *app.WeatherApplication, log *logger.Logger) {
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
