package app

import (
	"context"
	"fmt"
	"log/slog"

	"weatherapi.app/internal/adapters/external"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
	weathergrpc "weatherapi.app/internal/services/weather/adapters/grpc"
)

const (
	// Log messages
	logInitWeatherUC    = "Initializing weather use case..."
	logWeatherUCSuccess = "Weather use case initialized successfully"
	logInitGRPCHandler  = "Initializing gRPC handler..."
	logGRPCSuccess      = "gRPC handler initialized successfully"
	logFileLogEnabled   = "File logging enabled"
	logFileLogFallback  = "Failed to create file logger, falling back to slog"
	logShuttingDown     = "Shutting down weather application..."
	logShutdownComplete = "Weather application shutdown complete"

	// Error messages
	errInitWeatherUC   = "initialize weather use case: %w"
	errInitGRPCHandler = "initialize gRPC handler: %w"
	errCreateProvider  = "create weather provider manager: %w"
	errCreateCache     = "create cache provider: %w"
	errCreateUseCase   = "create weather use case: %w"
)

type WeatherApplication struct {
	config      *config.Config
	weatherUC   *weather.UseCase
	grpcHandler *weathergrpc.WeatherServiceServer
}

func NewWeatherApplication(cfg *config.Config) (*WeatherApplication, error) {
	if err := validateApplicationConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	app := &WeatherApplication{
		config: cfg,
	}

	if err := app.initializeWeatherUseCase(); err != nil {
		return nil, fmt.Errorf(errInitWeatherUC, err)
	}

	if err := app.initializeGRPCHandler(); err != nil {
		return nil, fmt.Errorf(errInitGRPCHandler, err)
	}

	return app, nil
}

func validateApplicationConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if cfg.Weather.APIKey == "" && cfg.Weather.OpenWeatherMapKey == "" && cfg.Weather.AccuWeatherKey == "" {
		return fmt.Errorf("at least one weather API key must be configured")
	}
	return nil
}

func (a *WeatherApplication) initializeWeatherUseCase() error {
	slog.Info(logInitWeatherUC)

	var logger ports.Logger = &infrastructure.SlogLoggerAdapter{}

	if a.config.Weather.EnableLogging && a.config.Weather.LogFilePath != "" {
		fileLogger, err := infrastructure.NewFileLoggerAdapter(a.config.Weather.LogFilePath)
		if err != nil {
			slog.Warn(logFileLogFallback, "error", err)
		} else {
			logger = fileLogger
			slog.Info(logFileLogEnabled, "path", a.config.Weather.LogFilePath)
		}
	}

	providerManager, err := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     a.config.Weather.APIKey,
		WeatherAPIBaseURL: a.config.Weather.BaseURL,
		OpenWeatherKey:    a.config.Weather.OpenWeatherMapKey,
		OpenWeatherURL:    a.config.Weather.OpenWeatherMapBaseURL,
		AccuWeatherKey:    a.config.Weather.AccuWeatherKey,
		AccuWeatherURL:    a.config.Weather.AccuWeatherBaseURL,
		ProviderOrder:     a.config.Weather.ProviderOrder,
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf(errCreateProvider, err)
	}

	if a.config.Weather.EnableLogging {
		providerManager = external.NewWeatherProviderManagerLoggingDecorator(providerManager, logger)
	}

	cacheFactory := external.NewCacheProviderFactory()
	genericCacheProvider, err := cacheFactory.CreateCacheProvider(context.Background(), &a.config.Cache)
	if err != nil {
		return fmt.Errorf(errCreateCache, err)
	}

	weatherCacheProvider := external.NewWeatherCacheAdapter(genericCacheProvider)
	configProvider := infrastructure.NewConfigProviderAdapter(a.config)
	weatherMetrics := external.NewWeatherMetricsAdapter(weatherCacheProvider, providerManager)

	weatherUC, err := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: providerManager,
		Cache:           weatherCacheProvider,
		Config:          configProvider,
		Logger:          logger,
		Metrics:         weatherMetrics,
	})
	if err != nil {
		return fmt.Errorf(errCreateUseCase, err)
	}

	a.weatherUC = weatherUC
	slog.Info(logWeatherUCSuccess)
	return nil
}

func (a *WeatherApplication) initializeGRPCHandler() error {
	slog.Info(logInitGRPCHandler)

	grpcHandler := weathergrpc.NewWeatherServiceServer(a.weatherUC)
	a.grpcHandler = grpcHandler

	slog.Info(logGRPCSuccess)
	return nil
}

func (a *WeatherApplication) GetGRPCHandler() *weathergrpc.WeatherServiceServer {
	return a.grpcHandler
}

func (a *WeatherApplication) Shutdown(ctx context.Context) error {
	slog.Info(logShuttingDown)
	slog.Info(logShutdownComplete)
	return nil
}
