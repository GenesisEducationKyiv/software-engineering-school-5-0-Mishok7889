package infrastructure

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"weatherapi.app/pkg/logger"
)

// MetricsFactory creates and configures metrics components for services
type MetricsFactory struct {
	serviceName    string
	serviceVersion string
	logger         *logger.Logger
}

// NewMetricsFactory creates a new metrics factory for a specific service
func NewMetricsFactory(serviceName, serviceVersion string, logger *logger.Logger) *MetricsFactory {
	return &MetricsFactory{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		logger:         logger,
	}
}

// CreatePrometheusAdapter creates a Prometheus metrics adapter with existing metrics collector
func (f *MetricsFactory) CreatePrometheusAdapter(existingCollector *MetricsCollectorAdapter) *PrometheusMetricsAdapter {
	config := PrometheusConfig{
		ServiceName:      f.serviceName,
		ServiceVersion:   f.serviceVersion,
		MetricsCollector: existingCollector,
		Logger:           f.logger.WithComponent("prometheus"),
	}

	adapter := NewPrometheusMetricsAdapter(config)

	f.logger.Info("prometheus metrics adapter created",
		"service", f.serviceName,
		"version", f.serviceVersion,
	)

	return adapter
}

// CreateHealthAndMetricsHandler creates a combined health and metrics handler
func (f *MetricsFactory) CreateHealthAndMetricsHandler(router *gin.Engine, prometheusAdapter *PrometheusMetricsAdapter) {
	// Health endpoint with metrics
	router.GET("/health", func(c *gin.Context) {
		health := map[string]interface{}{
			"status":    "healthy",
			"service":   f.serviceName,
			"version":   f.serviceVersion,
			"timestamp": "now",
		}

		// Record health check metric
		prometheusAdapter.RecordHTTPRequest("GET", "health", "200", 0, 0)

		c.JSON(http.StatusOK, health)
	})

	// Metrics info endpoint
	router.GET("/metrics/info", func(c *gin.Context) {
		info := map[string]interface{}{
			"service":          f.serviceName,
			"version":          f.serviceVersion,
			"metrics_endpoint": "/metrics",
			"health_endpoint":  "/health",
			"metrics_format":   "prometheus",
		}

		c.JSON(http.StatusOK, info)
	})

	f.logger.Info("health and metrics endpoints added",
		"service", f.serviceName,
		"endpoints", []string{"/health", "/metrics/info"},
	)
}

// MetricsServerConfig holds configuration for a dedicated metrics server
type MetricsServerConfig struct {
	Port int
	Host string
}

// CreateDedicatedMetricsServer creates a separate HTTP server just for metrics
func (f *MetricsFactory) CreateDedicatedMetricsServer(config MetricsServerConfig, prometheusAdapter *PrometheusMetricsAdapter) *http.Server {
	router := gin.New()
	router.Use(gin.Recovery())

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(prometheusAdapter.Handler()))

	// Health for metrics server
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": f.serviceName + "-metrics",
		})
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler: router,
	}

	f.logger.Info("dedicated metrics server created",
		"service", f.serviceName,
		"metrics_addr", server.Addr,
	)

	return server
}

// WeatherMetricsHelper provides weather-specific metrics recording
type WeatherMetricsHelper struct {
	adapter *PrometheusMetricsAdapter
	logger  *logger.Logger
}

// NewWeatherMetricsHelper creates weather-specific metrics helper
func (f *MetricsFactory) NewWeatherMetricsHelper(adapter *PrometheusMetricsAdapter) *WeatherMetricsHelper {
	return &WeatherMetricsHelper{
		adapter: adapter,
		logger:  f.logger.WithComponent("weather-metrics"),
	}
}

func (w *WeatherMetricsHelper) RecordWeatherFetch(provider, city string, success bool, duration string) {
	status := "success"
	if !success {
		status = "error"
	}

	// Parse duration if provided
	// This would need proper duration parsing in real implementation
	w.adapter.RecordWeatherRequest(provider, city, status, 0)

	w.logger.Debug("weather fetch metrics recorded",
		"provider", provider,
		"city", city,
		"status", status,
	)
}

func (w *WeatherMetricsHelper) RecordProviderFailure(provider, errorType string) {
	w.adapter.RecordWeatherProviderFailure(provider, errorType)

	w.logger.Warn("weather provider failure recorded",
		"provider", provider,
		"error_type", errorType,
	)
}

// NotificationMetricsHelper provides notification-specific metrics recording
type NotificationMetricsHelper struct {
	adapter *PrometheusMetricsAdapter
	logger  *logger.Logger
}

// NewNotificationMetricsHelper creates notification-specific metrics helper
func (f *MetricsFactory) NewNotificationMetricsHelper(adapter *PrometheusMetricsAdapter) *NotificationMetricsHelper {
	return &NotificationMetricsHelper{
		adapter: adapter,
		logger:  f.logger.WithComponent("notification-metrics"),
	}
}

func (n *NotificationMetricsHelper) RecordEmailSent(emailType string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	n.adapter.RecordEmailSent(emailType, status)

	n.logger.Debug("email metrics recorded",
		"type", emailType,
		"status", status,
	)
}

func (n *NotificationMetricsHelper) RecordMessageProcessed(topic, consumerGroup string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	n.adapter.RecordMessageConsumed(topic, consumerGroup, status)

	n.logger.Debug("message processing metrics recorded",
		"topic", topic,
		"consumer_group", consumerGroup,
		"status", status,
	)
}

// SubscriptionMetricsHelper provides subscription-specific metrics recording
type SubscriptionMetricsHelper struct {
	adapter *PrometheusMetricsAdapter
	logger  *logger.Logger
}

// NewSubscriptionMetricsHelper creates subscription-specific metrics helper
func (f *MetricsFactory) NewSubscriptionMetricsHelper(adapter *PrometheusMetricsAdapter) *SubscriptionMetricsHelper {
	return &SubscriptionMetricsHelper{
		adapter: adapter,
		logger:  f.logger.WithComponent("subscription-metrics"),
	}
}

func (s *SubscriptionMetricsHelper) RecordSubscriptionOperation(operation, frequency string) {
	s.adapter.RecordSubscriptionOperation(operation, frequency)

	s.logger.Debug("subscription operation metrics recorded",
		"operation", operation,
		"frequency", frequency,
	)
}

func (s *SubscriptionMetricsHelper) UpdateActiveSubscriptions(count int) {
	s.adapter.UpdateActiveSubscriptions(float64(count))

	s.logger.Debug("active subscriptions updated",
		"count", count,
	)
}
