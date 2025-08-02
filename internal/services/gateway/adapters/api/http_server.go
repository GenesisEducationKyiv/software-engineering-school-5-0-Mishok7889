package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"weatherapi.app/internal/ports"
)

type ServerConfig struct {
	Port int
}

type HTTPServer struct {
	router              *gin.Engine
	server              *http.Server
	weatherService      ports.WeatherService
	subscriptionService ports.SubscriptionService
	userService         ports.UserService
	logger              ports.Logger
}

func NewHTTPServer(
	config ServerConfig,
	weatherService ports.WeatherService,
	subscriptionService ports.SubscriptionService,
	userService ports.UserService,
	logger ports.Logger,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	s := &HTTPServer{
		router:              router,
		weatherService:      weatherService,
		subscriptionService: subscriptionService,
		userService:         userService,
		logger:              logger,
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
	// Serve static UI files
	s.router.Static("/static", "./public")
	s.router.StaticFile("/", "./public/index.html")
	s.router.StaticFile("/favicon.ico", "./public/favicon.ico")

	// API routes that proxy to microservices
	api := s.router.Group("/api")
	{
		// Health endpoint
		api.GET("/health", s.getHealth)

		// Weather endpoints - proxy to Weather Service
		api.GET("/weather", s.getWeather)

		// Subscription endpoints - proxy to Subscription Service
		api.POST("/subscribe", s.subscribe)
		api.GET("/confirm/:token", s.confirmSubscription)
		api.GET("/unsubscribe/:token", s.unsubscribe)

		// Metrics endpoint - aggregate from services
		api.GET("/metrics", s.getMetrics)

		// Debug endpoint
		api.GET("/debug", s.getDebug)
	}
}

func (s *HTTPServer) getHealth(c *gin.Context) {
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "api-gateway",
	}

	c.JSON(http.StatusOK, healthStatus)
}

func (s *HTTPServer) getWeather(c *gin.Context) {
	city := c.Query("city")
	if city == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "city parameter is required"})
		return
	}

	weather, err := s.weatherService.GetWeather(c.Request.Context(), city)
	if err != nil {
		s.logger.Error("Failed to get weather", ports.F("error", err), ports.F("city", city))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get weather data"})
		return
	}

	c.JSON(http.StatusOK, weather)
}

func (s *HTTPServer) subscribe(c *gin.Context) {
	var req struct {
		Email     string `form:"email" json:"email" binding:"required,email"`
		City      string `form:"city" json:"city" binding:"required"`
		Frequency string `form:"frequency" json:"frequency" binding:"required,oneof=hourly daily"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.subscriptionService.Subscribe(c.Request.Context(), req.Email, req.City, req.Frequency); err != nil {
		s.logger.Error("Failed to create subscription",
			ports.F("error", err),
			ports.F("email", req.Email),
			ports.F("city", req.City))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Subscription created successfully"})
}

func (s *HTTPServer) confirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := s.subscriptionService.ConfirmSubscription(c.Request.Context(), token); err != nil {
		s.logger.Error("Failed to confirm subscription", ports.F("error", err), ports.F("token", token))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subscription confirmed successfully"})
}

func (s *HTTPServer) unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := s.subscriptionService.Unsubscribe(c.Request.Context(), token); err != nil {
		s.logger.Error("Failed to unsubscribe", ports.F("error", err), ports.F("token", token))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

func (s *HTTPServer) getMetrics(c *gin.Context) {
	metrics := map[string]interface{}{
		"gateway": map[string]interface{}{
			"uptime":    time.Since(time.Now()).String(),
			"requests":  "N/A",
			"timestamp": time.Now().UTC(),
		},
	}

	c.JSON(http.StatusOK, metrics)
}

func (s *HTTPServer) getDebug(c *gin.Context) {
	debug := map[string]interface{}{
		"service":   "api-gateway",
		"timestamp": time.Now().UTC(),
		"routes": []string{
			"GET /",
			"GET /api/health",
			"GET /api/weather",
			"POST /api/subscribe",
			"GET /api/confirm/:token",
			"GET /api/unsubscribe/:token",
			"GET /api/metrics",
		},
	}

	c.JSON(http.StatusOK, debug)
}

func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", ports.F("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}
