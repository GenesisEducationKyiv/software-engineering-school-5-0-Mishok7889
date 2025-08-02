package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type HTTPServer struct {
	server    *http.Server
	router    *gin.Engine
	publisher messaging.Publisher
	logger    ports.Logger
}

type ServerConfig struct {
	Port int
}

func NewHTTPServer(config ServerConfig, publisher messaging.Publisher, logger ports.Logger) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	s := &HTTPServer{
		router:    router,
		publisher: publisher,
		logger:    logger,
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *HTTPServer) setupRoutes() {
	s.router.Use(gin.Recovery())
	s.router.Use(gin.Logger())

	api := s.router.Group("/api/v1")
	{
		api.GET("/health", s.healthCheck)
		api.POST("/notifications/weather-updates", s.triggerWeatherUpdates)
	}
}

func (s *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "notification-service",
		"timestamp": time.Now().UTC(),
	})
}

func (s *HTTPServer) triggerWeatherUpdates(c *gin.Context) {
	var req struct {
		Frequency string `json:"frequency" binding:"required,oneof=hourly daily"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Invalid request payload", ports.F("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid frequency. Must be 'hourly' or 'daily'"})
		return
	}

	command := shared.NewSendWeatherUpdatesCommand(req.Frequency)

	if err := s.publisher.PublishCommand(c.Request.Context(), command); err != nil {
		s.logger.Error("Failed to publish weather updates command",
			ports.F("error", err),
			ports.F("frequency", req.Frequency))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger weather updates"})
		return
	}

	s.logger.Info("Weather updates command triggered",
		ports.F("frequency", req.Frequency),
		ports.F("command_id", command.ID()))

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Weather updates triggered successfully",
		"command_id": command.ID(),
		"frequency":  req.Frequency,
	})
}

func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", ports.F("address", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}
