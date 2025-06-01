package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/models"
	"weatherapi.app/repository"
	"weatherapi.app/service"
)

type Server struct {
	router              *gin.Engine
	db                  *gorm.DB
	config              *config.Config
	weatherService      service.WeatherServiceInterface
	subscriptionService service.SubscriptionServiceInterface
}

func NewServer(db *gorm.DB, config *config.Config) *Server {
	router := gin.Default()

	weatherService := service.NewWeatherService(config)
	emailService := service.NewEmailService(config)

	subscriptionRepo := repository.NewSubscriptionRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	subscriptionService := service.NewSubscriptionService(
		db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		config,
	)

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

		// Add a debug endpoint
		api.GET("/debug", s.debugEndpoint)
	}

	s.ServeStaticFiles()
}

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
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "city is required"})
		return
	}

	fmt.Printf("[DEBUG] Getting weather for city: %s\n", city)
	weather, err := s.weatherService.GetWeather(city)
	if err != nil {
		fmt.Printf("[ERROR] Weather API error: %v\n", err)
		if err.Error() == "city not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "city not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get weather data"})
		return
	}

	fmt.Printf("!!!!!!!!!!!!!!!!!!![DEBUG] Weather result: %+v\n", weather)
	c.JSON(http.StatusOK, weather)
}

func (s *Server) subscribe(c *gin.Context) {
	var req models.SubscriptionRequest
	fmt.Println("[DEBUG] Handling subscription request")

	if err := c.ShouldBind(&req); err != nil {
		fmt.Printf("[ERROR] Binding error: %v\n", err)
		fmt.Printf("[ERROR] Request content-type: %s\n", c.ContentType())
		fmt.Printf("[ERROR] Request body: %+v\n", c.Request.Body)
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	fmt.Printf("[DEBUG] Subscription request received: %+v\n", req)

	if err := s.subscriptionService.Subscribe(&req); err != nil {
		fmt.Printf("[ERROR] Subscription error: %v\n", err)

		if err.Error() == "email already subscribed" {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: "email already subscribed"})
			return
		}

		if err.Error() == "failed to send confirmation email: failed to send email: 426 Upgrade Required" ||
			strings.Contains(err.Error(), "failed to send confirmation email") {
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{Error: "unable to send confirmation email"})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create subscription"})
		return
	}

	fmt.Println("[DEBUG] Subscription created successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Subscription successful. Confirmation email sent."})
}

func (s *Server) confirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "token is required"})
		return
	}

	fmt.Printf("[DEBUG] Confirming subscription with token: %s\n", token)

	if err := s.subscriptionService.ConfirmSubscription(token); err != nil {
		fmt.Printf("[ERROR] Confirmation error: %v\n", err)

		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "token not found"})
			return
		}
		if err.Error() == "invalid token type" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid token"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to confirm subscription"})
		return
	}

	fmt.Println("[DEBUG] Subscription confirmed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Subscription confirmed successfully"})
}

func (s *Server) unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "token is required"})
		return
	}

	fmt.Printf("[DEBUG] Unsubscribing with token: %s\n", token)

	if err := s.subscriptionService.Unsubscribe(token); err != nil {
		fmt.Printf("[ERROR] Unsubscribe error: %v\n", err)

		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "token not found"})
			return
		}
		if err.Error() == "invalid token type" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid token"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to unsubscribe"})
		return
	}

	fmt.Println("[DEBUG] Unsubscribed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

// Debug endpoint to check configuration and connectivity
func (s *Server) debugEndpoint(c *gin.Context) {
	fmt.Println("[DEBUG] Debug endpoint called")

	// Test database connection
	var subscriptionCount int64
	dbErr := s.db.Model(&models.Subscription{}).Count(&subscriptionCount).Error

	// Test weather API
	weatherResponse, weatherErr := s.weatherService.GetWeather("London")

	// Test SMTP configuration
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
