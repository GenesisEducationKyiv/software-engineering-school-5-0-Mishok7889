package app

import (
	"context"
	"fmt"
	"net/http"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	gatewayapi "weatherapi.app/internal/services/gateway/adapters/api"
	gatewaygrpc "weatherapi.app/internal/services/gateway/adapters/grpc"
	"weatherapi.app/pkg/logger"
)

const (
	logServiceStarting   = "Starting API Gateway"
	logServiceShutdown   = "Shutting down API Gateway..."
	logShutdownComplete  = "API Gateway shutdown complete"
	logHTTPServerStarted = "HTTP server started successfully"
	logHTTPServerStopped = "HTTP server stopped successfully"

	errInvalidConfig   = "invalid configuration: %w"
	errInitServices    = "initialize services: %w"
	errInitHTTPServer  = "initialize HTTP server: %w"
	errStartHTTPServer = "start HTTP server: %w"
	errStopHTTPServer  = "stop HTTP server: %w"
)

type GatewayApplication struct {
	config              *config.Config
	weatherService      ports.WeatherService
	subscriptionService ports.SubscriptionService
	userService         ports.UserService
	httpServer          *gatewayapi.HTTPServer
	logger              ports.Logger
}

func NewGatewayApplication(cfg *config.Config) (*GatewayApplication, error) {
	return NewGatewayApplicationWithLogger(cfg, nil)
}

// NewGatewayApplicationWithLogger creates a gateway application with logger
func NewGatewayApplicationWithLogger(cfg *config.Config, log *logger.Logger) (*GatewayApplication, error) {
	app := &GatewayApplication{
		config: cfg,
		logger: &infrastructure.SlogLoggerAdapter{},
	}

	// Use provided logger if available
	if log != nil {
		app.logger = &infrastructure.LoggerAdapter{Logger: log}
	}

	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf(errInitServices, err)
	}

	if err := app.initializeHTTPServer(); err != nil {
		return nil, fmt.Errorf(errInitHTTPServer, err)
	}

	return app, nil
}

func (a *GatewayApplication) initializeServices() error {
	a.logger.Info("Initializing microservice clients")

	weatherClient, err := gatewaygrpc.NewWeatherServiceClient(gatewaygrpc.WeatherServiceConfig{
		Host: a.config.Services.Weather.Host,
		Port: a.config.Services.Weather.Port,
	})
	if err != nil {
		return fmt.Errorf("create weather service client: %w", err)
	}

	userClient, err := gatewaygrpc.NewUserServiceClient(gatewaygrpc.UserServiceConfig{
		Host: a.config.Services.User.Host,
		Port: a.config.Services.User.Port,
	})
	if err != nil {
		return fmt.Errorf("create user service client: %w", err)
	}

	subscriptionClient, err := gatewaygrpc.NewSubscriptionServiceHTTPClient(gatewaygrpc.SubscriptionServiceHTTPConfig{
		Host: a.config.Services.Subscription.Host,
		Port: a.config.Services.Subscription.Port,
	})
	if err != nil {
		return fmt.Errorf("create subscription service client: %w", err)
	}

	a.weatherService = weatherClient
	a.userService = userClient
	a.subscriptionService = subscriptionClient

	a.logger.Info("Microservice clients initialized successfully")
	return nil
}

func (a *GatewayApplication) initializeHTTPServer() error {
	a.logger.Info("Initializing HTTP server")

	httpServer := gatewayapi.NewHTTPServer(
		gatewayapi.ServerConfig{
			Port: a.config.Services.Gateway.Port,
		},
		a.weatherService,
		a.subscriptionService,
		a.userService,
		a.logger,
	)

	a.httpServer = httpServer

	a.logger.Info("HTTP server initialized successfully")
	return nil
}

func (a *GatewayApplication) Start(ctx context.Context) error {
	a.logger.Info(logServiceStarting, ports.F("port", a.config.Services.Gateway.Port))

	if err := a.httpServer.Start(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf(errStartHTTPServer, err)
	}
	a.logger.Info(logHTTPServerStarted)

	return nil
}

func (a *GatewayApplication) Shutdown(ctx context.Context) error {
	a.logger.Info(logServiceShutdown)

	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.logger.Error("Error shutting down HTTP server", ports.F("error", err))
		} else {
			a.logger.Info(logHTTPServerStopped)
		}
	}

	a.logger.Info(logShutdownComplete)
	return nil
}

func (a *GatewayApplication) GetHTTPServer() *gatewayapi.HTTPServer {
	return a.httpServer
}
