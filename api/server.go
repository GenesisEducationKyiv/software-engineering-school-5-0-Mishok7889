package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
	"weatherapi.app/config"
	weathererr "weatherapi.app/errors"
	"weatherapi.app/models"
	"weatherapi.app/providers"
	"weatherapi.app/service"
)

// Server represents the HTTP server and API handler
type Server struct {
	router              *gin.Engine
	db                  *gorm.DB
	config              *config.Config
	weatherService      service.WeatherServiceInterface
	subscriptionService service.SubscriptionServiceInterface
	providerManager     providers.WeatherManager
	providerMetrics     providers.WeatherProviderMetrics
}

// ServerOptions contains all dependencies needed to create a new server
type ServerOptions struct {
	DB                  *gorm.DB
	Config              *config.Config
	WeatherService      service.WeatherServiceInterface
	SubscriptionService service.SubscriptionServiceInterface
	ProviderManager     providers.WeatherManager
	ProviderMetrics     providers.WeatherProviderMetrics
}

// Validate checks if all required dependencies are provided
func (opts *ServerOptions) Validate() error {
	if opts.Config == nil {
		return errors.New("config is required")
	}
	if opts.WeatherService == nil {
		return errors.New("weather service is required")
	}
	if opts.SubscriptionService == nil {
		return errors.New("subscription service is required")
	}
	if opts.ProviderManager == nil {
		return errors.New("provider manager is required")
	}
	if opts.ProviderMetrics == nil {
		return errors.New("provider metrics is required")
	}
	return nil
}

// ServerOptionsBuilder helps build ServerOptions with a fluent interface
type ServerOptionsBuilder struct {
	opts ServerOptions
}

// NewServerOptionsBuilder creates a new ServerOptionsBuilder
func NewServerOptionsBuilder() *ServerOptionsBuilder {
	return &ServerOptionsBuilder{}
}

// WithDB sets the database
func (b *ServerOptionsBuilder) WithDB(db *gorm.DB) *ServerOptionsBuilder {
	b.opts.DB = db
	return b
}

// WithConfig sets the configuration
func (b *ServerOptionsBuilder) WithConfig(config *config.Config) *ServerOptionsBuilder {
	b.opts.Config = config
	return b
}

// WithWeatherService sets the weather service
func (b *ServerOptionsBuilder) WithWeatherService(weatherService service.WeatherServiceInterface) *ServerOptionsBuilder {
	b.opts.WeatherService = weatherService
	return b
}

// WithSubscriptionService sets the subscription service
func (b *ServerOptionsBuilder) WithSubscriptionService(subscriptionService service.SubscriptionServiceInterface) *ServerOptionsBuilder {
	b.opts.SubscriptionService = subscriptionService
	return b
}

// WithProviderManager sets the provider manager
func (b *ServerOptionsBuilder) WithProviderManager(providerManager providers.WeatherManager) *ServerOptionsBuilder {
	b.opts.ProviderManager = providerManager
	return b
}

func (b *ServerOptionsBuilder) WithProviderMetrics(providerMetrics providers.WeatherProviderMetrics) *ServerOptionsBuilder {
	b.opts.ProviderMetrics = providerMetrics
	return b
}

// Build creates the ServerOptions
func (b *ServerOptionsBuilder) Build() ServerOptions {
	return b.opts
}

// NewServer creates and configures a new HTTP server
func NewServer(opts ServerOptions) (*Server, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid server options: %w", err)
	}

	router := gin.Default()

	server := &Server{
		router:              router,
		db:                  opts.DB,
		config:              opts.Config,
		weatherService:      opts.WeatherService,
		subscriptionService: opts.SubscriptionService,
		providerManager:     opts.ProviderManager,
		providerMetrics:     opts.ProviderMetrics,
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
		api.GET("/weather", s.getWeather)
		api.POST("/subscribe", s.subscribe)
		api.GET("/confirm/:token", s.confirmSubscription)
		api.GET("/unsubscribe/:token", s.unsubscribe)
		api.GET("/debug", s.debugEndpoint)
		api.GET("/metrics", s.metricsEndpoint)
	}

	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	s.ServeStaticFiles()
}

// Start begins the HTTP server
func (s *Server) Start() error {
	return s.router.Run(fmt.Sprintf(":%d", s.config.Server.Port))
}

// GetRouter returns the router for testing purposes
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

func (s *Server) getWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		s.handleError(c, weathererr.NewValidationError("city parameter is required"))
		return
	}

	slog.Debug("Getting weather for city", "city", city)
	weather, err := s.weatherService.GetWeather(city)
	if err != nil {
		slog.Error("Weather service error", "error", err, "city", city)
		s.handleError(c, err)
		return
	}

	slog.Debug("Weather result", "weather", weather, "city", city)
	c.JSON(http.StatusOK, weather)
}

func (s *Server) subscribe(c *gin.Context) {
	var req models.SubscriptionRequest
	slog.Debug("Handling subscription request")

	if err := c.ShouldBind(&req); err != nil {
		slog.Error("Request binding error", "error", err)
		s.handleError(c, weathererr.NewValidationError("invalid request format"))
		return
	}

	slog.Debug("Subscription request received", "email", req.Email, "city", req.City, "frequency", req.Frequency)

	if err := s.subscriptionService.Subscribe(&req); err != nil {
		slog.Error("Subscription error", "error", err, "email", req.Email, "city", req.City)
		s.handleError(c, err)
		return
	}

	slog.Debug("Subscription created successfully", "email", req.Email, "city", req.City)
	c.JSON(http.StatusOK, gin.H{"message": "Subscription successful. Confirmation email sent."})
}

func (s *Server) confirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, weathererr.NewValidationError("token parameter is required"))
		return
	}

	slog.Debug("Confirming subscription", "token", token)

	if err := s.subscriptionService.ConfirmSubscription(token); err != nil {
		slog.Error("Confirmation error", "error", err, "token", token)
		s.handleError(c, err)
		return
	}

	slog.Debug("Subscription confirmed successfully", "token", token)
	c.JSON(http.StatusOK, gin.H{"message": "Subscription confirmed successfully"})
}

func (s *Server) unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		s.handleError(c, weathererr.NewValidationError("token parameter is required"))
		return
	}

	slog.Debug("Unsubscribing", "token", token)

	if err := s.subscriptionService.Unsubscribe(token); err != nil {
		slog.Error("Unsubscribe error", "error", err, "token", token)
		s.handleError(c, err)
		return
	}

	slog.Debug("Unsubscribed successfully", "token", token)
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

func (s *Server) debugEndpoint(c *gin.Context) {
	slog.Debug("Debug endpoint called")

	var subscriptionCount int64
	dbErr := s.db.Model(&models.Subscription{}).Count(&subscriptionCount).Error

	weatherResponse, weatherErr := s.weatherService.GetWeather("London")

	smtpConfig := map[string]string{
		"host":        s.config.Email.SMTPHost,
		"port":        fmt.Sprintf("%d", s.config.Email.SMTPPort),
		"username":    s.config.Email.SMTPUsername,
		"fromAddress": s.config.Email.FromAddress,
		"fromName":    s.config.Email.FromName,
	}

	response := gin.H{
		"database": map[string]interface{}{
			"connected":         dbErr == nil,
			"error":             dbErr,
			"subscriptionCount": subscriptionCount,
		},
		"weatherAPI": map[string]interface{}{
			"connected": weatherErr == nil,
			"error":     weatherErr,
			"response":  weatherResponse,
		},
		"smtp": smtpConfig,
		"config": map[string]string{
			"appBaseURL": s.config.AppBaseURL,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) metricsEndpoint(c *gin.Context) {
	slog.Debug("Metrics endpoint called")

	cacheMetrics, err := s.providerMetrics.GetCacheMetrics()
	if err != nil {
		slog.Error("Error getting cache metrics", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cache metrics unavailable"})
		return
	}

	providerInfo := s.providerMetrics.GetProviderInfo()

	response := gin.H{
		"cache":         cacheMetrics,
		"provider_info": providerInfo,
		"endpoints": gin.H{
			"prometheus_metrics": "/metrics",
			"cache_metrics":      "/api/metrics",
		},
	}

	c.JSON(http.StatusOK, response)
}

// handleError handles different types of application errors
func (s *Server) handleError(c *gin.Context, err error) {
	var appErr *weathererr.AppError
	var statusCode int
	var message string

	if !errors.As(err, &appErr) {
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
		c.JSON(statusCode, models.ErrorResponse{Error: message})
		return
	}

	switch appErr.Type {
	case weathererr.ValidationError:
		statusCode = http.StatusBadRequest
		message = appErr.Message
	case weathererr.NotFoundError:
		statusCode = http.StatusNotFound
		message = appErr.Message
	case weathererr.AlreadyExistsError:
		statusCode = http.StatusConflict
		message = appErr.Message
	case weathererr.ExternalAPIError:
		statusCode = http.StatusServiceUnavailable
		message = "External service unavailable"
	case weathererr.DatabaseError:
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	case weathererr.EmailError:
		statusCode = http.StatusServiceUnavailable
		message = "Unable to send email"
	case weathererr.TokenError:
		statusCode = http.StatusBadRequest
		message = appErr.Message
	default:
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	}

	c.JSON(statusCode, models.ErrorResponse{Error: message})
}
