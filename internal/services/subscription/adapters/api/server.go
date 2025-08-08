package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"weatherapi.app/internal/adapters/middleware"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/ports"
)

const (
	readTimeout  = 30 * time.Second
	writeTimeout = 30 * time.Second
	idleTimeout  = 60 * time.Second
)

type HTTPServer struct {
	server               *http.Server
	router               *gin.Engine
	subscriptionHandlers *SubscriptionHandlers
	validationMiddleware *middleware.ValidationMiddleware
}

type ServerConfig struct {
	Port int
}

func NewHTTPServer(
	cfg ServerConfig,
	subscriptionUC *subscription.UseCase,
	logger ports.Logger,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)

	// Register custom validator for Frequency enum
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("frequency", validateFrequency)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	subscriptionHandlers := NewSubscriptionHandlers(subscriptionUC, logger)
	validationMiddleware := middleware.NewValidationMiddleware()

	server := &HTTPServer{
		router:               router,
		subscriptionHandlers: subscriptionHandlers,
		validationMiddleware: validationMiddleware,
	}

	server.setupRoutes()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	server.server = httpServer
	return server
}
func (s *HTTPServer) setupRoutes() {
	v1 := s.router.Group("/api/v1")
	{
		v1.POST("/subscriptions", s.subscriptionHandlers.Subscribe)
		v1.GET("/confirm/:token",
			s.validationMiddleware.ValidateTokenParam(),
			s.subscriptionHandlers.ConfirmSubscription)
		v1.GET("/unsubscribe/:token",
			s.validationMiddleware.ValidateTokenParam(),
			s.subscriptionHandlers.Unsubscribe)
	}

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)
}

func (s *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "subscription-service",
	})
}

func (s *HTTPServer) Start() error {
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}

// validateFrequency validates the frequency enum value
func validateFrequency(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	freq := subscription.FrequencyFromString(value)
	return freq == subscription.FrequencyHourly || freq == subscription.FrequencyDaily
}
