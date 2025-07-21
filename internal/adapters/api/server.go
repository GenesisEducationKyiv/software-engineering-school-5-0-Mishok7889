// Package api provides HTTP adapters for the hexagonal architecture
// These adapters handle incoming HTTP requests and translate them to use cases
package api

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"weatherapi.app/internal/adapters/middleware"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
)

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port int
}

// HTTPServerAdapter implements HTTP server using Gin framework
type HTTPServerAdapter struct {
	router              *gin.Engine
	config              ServerConfig
	weatherUseCase      WeatherUseCase
	subscriptionUseCase SubscriptionUseCase
	metricsCollector    MetricsCollector
	systemHealthChecker ports.SystemHealthChecker
	logger              ports.Logger
}

// Use case interfaces that the HTTP adapter depends on
type WeatherUseCase interface {
	GetWeather(ctx context.Context, request weather.WeatherRequest) (*weather.Weather, error)
}

type SubscriptionUseCase interface {
	Subscribe(ctx context.Context, params subscription.SubscribeParams) error
	ConfirmSubscription(ctx context.Context, params subscription.ConfirmParams) error
	Unsubscribe(ctx context.Context, params subscription.UnsubscribeParams) error
}

type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
}

// ServerOptions represents options for creating the HTTP server
type ServerOptions struct {
	Config              ServerConfig
	WeatherUseCase      WeatherUseCase
	SubscriptionUseCase SubscriptionUseCase
	MetricsCollector    MetricsCollector
	SystemHealthChecker ports.SystemHealthChecker
	Logger              ports.Logger
}

// NewHTTPServerAdapter creates a new HTTP server adapter
func NewHTTPServerAdapter(opts ServerOptions) (*HTTPServerAdapter, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server options: %w", err)
	}

	router := gin.Default()

	server := &HTTPServerAdapter{
		router:              router,
		config:              opts.Config,
		weatherUseCase:      opts.WeatherUseCase,
		subscriptionUseCase: opts.SubscriptionUseCase,
		metricsCollector:    opts.MetricsCollector,
		systemHealthChecker: opts.SystemHealthChecker,
		logger:              opts.Logger,
	}

	server.setupRoutes()
	return server, nil
}

// Validate checks if all required dependencies are provided
func (opts *ServerOptions) Validate() error {
	if opts.WeatherUseCase == nil {
		return NewValidationError(WeatherUseCaseRequiredMsg)
	}
	if opts.SubscriptionUseCase == nil {
		return NewValidationError(SubscriptionUseCaseRequiredMsg)
	}
	if opts.MetricsCollector == nil {
		return NewValidationError(MetricsCollectorRequiredMsg)
	}
	if opts.SystemHealthChecker == nil {
		return NewValidationError(SystemHealthCheckerRequiredMsg)
	}
	if opts.Logger == nil {
		return NewValidationError(LoggerRequiredMsg)
	}
	return nil
}

// setupRoutes configures all HTTP routes
func (s *HTTPServerAdapter) setupRoutes() {
	validationMiddleware := middleware.NewValidationMiddleware()
	metricsMiddleware := middleware.NewMetricsMiddleware(s.metricsCollector)

	api := s.router.Group("/api")
	api.Use(metricsMiddleware.CollectRequestMetrics())
	{
		api.GET("/health", s.getHealth)
		api.GET("/debug", s.getDebug)
		api.GET("/weather", validationMiddleware.ValidateCityQuery(), s.getWeather)
		api.POST("/subscribe", s.subscribe)
		api.GET("/confirm/:token", validationMiddleware.ValidateTokenParam(), s.confirmSubscription)
		api.GET("/unsubscribe/:token", validationMiddleware.ValidateTokenParam(), s.unsubscribe)
		api.GET("/metrics", s.getMetrics)
	}

	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	s.setupStaticFiles()
}

// Start begins the HTTP server
func (s *HTTPServerAdapter) Start(ctx context.Context) error {
	s.logger.Info(StartingHTTPServerMsg, ports.F(PortField, s.config.Port))
	return s.router.Run(fmt.Sprintf(":%d", s.config.Port))
}

// GetRouter returns the router for testing purposes
func (s *HTTPServerAdapter) GetRouter() *gin.Engine {
	return s.router
}

// setupStaticFiles configures static file serving
func (s *HTTPServerAdapter) setupStaticFiles() {
	s.router.Static("/static", "./public")
	s.router.StaticFile("/", "./public/index.html")
	s.router.StaticFile("/favicon.ico", "./public/favicon.ico")
}
