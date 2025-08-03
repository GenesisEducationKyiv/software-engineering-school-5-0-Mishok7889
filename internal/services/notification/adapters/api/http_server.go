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
	server        *http.Server
	router        *gin.Engine
	publisher     messaging.Publisher
	emailProvider ports.EmailProvider
	emailBuilder  ports.EmailBuilder
	logger        ports.Logger
}

type ServerConfig struct {
	Port int
}

func NewHTTPServer(
	config ServerConfig,
	publisher messaging.Publisher,
	emailProvider ports.EmailProvider,
	emailBuilder ports.EmailBuilder,
	logger ports.Logger,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	s := &HTTPServer{
		router:        router,
		publisher:     publisher,
		emailProvider: emailProvider,
		emailBuilder:  emailBuilder,
		logger:        logger,
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

		// Email endpoints for subscription service
		api.POST("/notifications/email/confirmation", s.sendConfirmationEmail)
		api.POST("/notifications/email/welcome", s.sendWelcomeEmail)
		api.POST("/notifications/email/unsubscribe", s.sendUnsubscribeEmail)
		api.POST("/notifications/email", s.sendGenericEmail)
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

// Email endpoints
func (s *HTTPServer) sendConfirmationEmail(c *gin.Context) {
	var req struct {
		Email           string `json:"email" binding:"required,email"`
		ConfirmationURL string `json:"confirmation_url" binding:"required"`
		City            string `json:"city" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Invalid confirmation email request", ports.F("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	emailParams, err := s.emailBuilder.BuildConfirmationEmail(req.City, req.ConfirmationURL)
	if err != nil {
		s.logger.Error("Failed to build confirmation email", ports.F("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build email"})
		return
	}

	emailParams.To = req.Email
	emailRequest := s.convertEmailParamsToRequest(emailParams)

	if err := s.emailProvider.SendEmail(c.Request.Context(), emailRequest); err != nil {
		s.logger.Error("Failed to send confirmation email",
			ports.F("error", err),
			ports.F("email", req.Email))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	s.logger.Info("Confirmation email sent successfully", ports.F("email", req.Email))
	c.JSON(http.StatusOK, gin.H{"message": "Confirmation email sent successfully"})
}

func (s *HTTPServer) sendWelcomeEmail(c *gin.Context) {
	var req struct {
		Email          string `json:"email" binding:"required,email"`
		City           string `json:"city" binding:"required"`
		Frequency      string `json:"frequency" binding:"required"`
		UnsubscribeURL string `json:"unsubscribe_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Invalid welcome email request", ports.F("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	emailParams, err := s.emailBuilder.BuildWelcomeEmail(req.City, req.Frequency, req.UnsubscribeURL)
	if err != nil {
		s.logger.Error("Failed to build welcome email", ports.F("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build email"})
		return
	}

	emailParams.To = req.Email
	emailRequest := s.convertEmailParamsToRequest(emailParams)

	if err := s.emailProvider.SendEmail(c.Request.Context(), emailRequest); err != nil {
		s.logger.Error("Failed to send welcome email",
			ports.F("error", err),
			ports.F("email", req.Email))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	s.logger.Info("Welcome email sent successfully", ports.F("email", req.Email))
	c.JSON(http.StatusOK, gin.H{"message": "Welcome email sent successfully"})
}

func (s *HTTPServer) sendUnsubscribeEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		City  string `json:"city" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Invalid unsubscribe email request", ports.F("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	emailParams, err := s.emailBuilder.BuildUnsubscribeEmail(req.City)
	if err != nil {
		s.logger.Error("Failed to build unsubscribe email", ports.F("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build email"})
		return
	}

	emailParams.To = req.Email
	emailRequest := s.convertEmailParamsToRequest(emailParams)

	if err := s.emailProvider.SendEmail(c.Request.Context(), emailRequest); err != nil {
		s.logger.Error("Failed to send unsubscribe email",
			ports.F("error", err),
			ports.F("email", req.Email))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	s.logger.Info("Unsubscribe email sent successfully", ports.F("email", req.Email))
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribe email sent successfully"})
}

func (s *HTTPServer) sendGenericEmail(c *gin.Context) {
	var req struct {
		To      string `json:"to" binding:"required,email"`
		Subject string `json:"subject" binding:"required"`
		Body    string `json:"body" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Error("Invalid generic email request", ports.F("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	emailRequest := shared.EmailRequest{
		To:      req.To,
		Subject: req.Subject,
		Body:    req.Body,
	}

	if err := s.emailProvider.SendEmail(c.Request.Context(), emailRequest); err != nil {
		s.logger.Error("Failed to send generic email",
			ports.F("error", err),
			ports.F("email", req.To))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	s.logger.Info("Generic email sent successfully", ports.F("email", req.To))
	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}

func (s *HTTPServer) convertEmailParamsToRequest(params ports.EmailParams) shared.EmailRequest {
	return shared.EmailRequest{
		To:      params.To,
		Subject: params.Subject,
		Body:    params.Body,
	}
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
