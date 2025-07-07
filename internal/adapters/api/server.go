// Package api provides HTTP adapters for the hexagonal architecture
// These adapters handle incoming HTTP requests and translate them to use cases
package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"weatherapi.app/internal/core/notification"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/pkg/errors"
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
	notificationUseCase NotificationUseCase
	metricsCollector    MetricsCollector
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

type NotificationUseCase interface {
	SendConfirmationEmail(ctx context.Context, request notification.NotificationRequest) error
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
	NotificationUseCase NotificationUseCase
	MetricsCollector    MetricsCollector
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
		notificationUseCase: opts.NotificationUseCase,
		metricsCollector:    opts.MetricsCollector,
	}

	server.setupRoutes()
	return server, nil
}

// Validate checks if all required dependencies are provided
func (opts *ServerOptions) Validate() error {
	if opts.WeatherUseCase == nil {
		return errors.NewValidationError("weather use case is required")
	}
	if opts.SubscriptionUseCase == nil {
		return errors.NewValidationError("subscription use case is required")
	}
	if opts.NotificationUseCase == nil {
		return errors.NewValidationError("notification use case is required")
	}
	if opts.MetricsCollector == nil {
		return errors.NewValidationError("metrics collector is required")
	}
	return nil
}

// setupRoutes configures all HTTP routes
func (s *HTTPServerAdapter) setupRoutes() {
	api := s.router.Group("/api")
	{
		api.GET("/weather", s.getWeather)
		api.POST("/subscribe", s.subscribe)
		api.GET("/confirm/:token", s.confirmSubscription)
		api.GET("/unsubscribe/:token", s.unsubscribe)
		api.GET("/metrics", s.getMetrics)
	}

	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	s.setupStaticFiles()
}

// Start begins the HTTP server
func (s *HTTPServerAdapter) Start(ctx context.Context) error {
	slog.Info("Starting HTTP server", "port", s.config.Port)
	return s.router.Run(fmt.Sprintf(":%d", s.config.Port))
}

// GetRouter returns the router for testing purposes
func (s *HTTPServerAdapter) GetRouter() *gin.Engine {
	return s.router
}

// setupStaticFiles configures static file serving
func (s *HTTPServerAdapter) setupStaticFiles() {
	s.router.Static("/static", "./public/static")
	s.router.StaticFile("/", "./public/index.html")
	s.router.StaticFile("/favicon.ico", "./public/favicon.ico")
}
