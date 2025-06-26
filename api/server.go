package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"weatherapi.app/config"
	weathererr "weatherapi.app/errors"
	"weatherapi.app/models"
	"weatherapi.app/service"
)

// Server represents the HTTP server and API handler
type Server struct {
	router              *gin.Engine
	db                  *gorm.DB
	config              *config.Config
	weatherService      service.WeatherServiceInterface
	subscriptionService service.SubscriptionServiceInterface
}

// NewServer creates and configures a new HTTP server
func NewServer(
	db *gorm.DB,
	config *config.Config,
	weatherService service.WeatherServiceInterface,
	subscriptionService service.SubscriptionServiceInterface,
) *Server {
	router := gin.Default()

	server := &Server{
		router:              router,
		db:                  db,
		config:              config,
		weatherService:      weatherService,
		subscriptionService: subscriptionService,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
		api.GET("/weather", s.getWeather)
		api.POST("/subscribe", s.subscribe)
		api.GET("/confirm/:token", s.confirmSubscription)
		api.GET("/unsubscribe/:token", s.unsubscribe)
		api.GET("/debug", s.debugEndpoint)
	}

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

// handleError handles different types of application errors
func (s *Server) handleError(c *gin.Context, err error) {
	var appErr *weathererr.AppError
	var statusCode int
	var message string

	if errors.As(err, &appErr) {
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
	} else {
		statusCode = http.StatusInternalServerError
		message = "Internal server error"
	}

	c.JSON(statusCode, models.ErrorResponse{Error: message})
}
