package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"weatherapi.app/pkg/logger"
)

// PrometheusMetricsRecorder interface to avoid circular import
type PrometheusMetricsRecorder interface {
	RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration, responseSize int64)
}

// PrometheusMiddleware provides Prometheus metrics collection for HTTP requests
type PrometheusMiddleware struct {
	prometheusRecorder PrometheusMetricsRecorder
	logger             *logger.Logger
}

// NewPrometheusMiddleware creates a new Prometheus middleware
func NewPrometheusMiddleware(recorder PrometheusMetricsRecorder, logger *logger.Logger) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		prometheusRecorder: recorder,
		logger:             logger,
	}
}

// CollectPrometheusMetrics middleware collects detailed HTTP metrics for Prometheus
func (m *PrometheusMiddleware) CollectPrometheusMetrics() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// Get correlation-aware logger
		requestLogger := GetLoggerFromContext(c)
		if requestLogger == nil {
			requestLogger = m.logger
		}

		method := c.Request.Method
		endpoint := extractEndpointFromPath(c.FullPath())

		requestLogger.Debug("processing request for metrics",
			"method", method,
			"endpoint", endpoint,
			"user_agent", c.Request.UserAgent(),
		)

		c.Next()

		duration := time.Since(start)
		statusCode := strconv.Itoa(c.Writer.Status())
		responseSize := int64(c.Writer.Size())

		// Record metrics in Prometheus
		m.prometheusRecorder.RecordHTTPRequest(method, endpoint, statusCode, duration, responseSize)

		requestLogger.Debug("metrics recorded",
			"duration_ms", duration.Milliseconds(),
			"status_code", statusCode,
			"response_size", responseSize,
		)

		// Log slow requests
		if duration > 1*time.Second {
			requestLogger.Warn("slow request detected",
				"method", method,
				"endpoint", endpoint,
				"duration_ms", duration.Milliseconds(),
				"status_code", statusCode,
			)
		}

		// Log error responses
		if c.Writer.Status() >= 400 {
			requestLogger.Warn("error response",
				"method", method,
				"endpoint", endpoint,
				"status_code", statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		}
	})
}

// Use extractEndpointFromPath from existing metrics_middleware.go
// Function is imported from the middleware package

// MetricsReportingMiddleware periodically syncs legacy metrics with Prometheus
type MetricsReportingMiddleware struct {
	logger       *logger.Logger
	lastSync     time.Time
	syncInterval time.Duration
}

// MetricsSyncer interface to avoid circular import
type MetricsSyncer interface {
	SyncFromLegacyMetrics(ctx context.Context) error
}

// NewMetricsReportingMiddleware creates a middleware that syncs metrics
func NewMetricsReportingMiddleware(logger *logger.Logger) *MetricsReportingMiddleware {
	return &MetricsReportingMiddleware{
		logger:       logger,
		syncInterval: 30 * time.Second, // Sync every 30 seconds
	}
}

// SyncMetrics middleware periodically syncs legacy metrics to Prometheus
func (m *MetricsReportingMiddleware) SyncMetrics(syncer MetricsSyncer) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		now := time.Now()

		if now.Sub(m.lastSync) > m.syncInterval {
			go func() {
				if err := syncer.SyncFromLegacyMetrics(c.Request.Context()); err != nil {
					m.logger.Error("failed to sync legacy metrics to prometheus", "error", err)
				} else {
					m.logger.Debug("successfully synced legacy metrics to prometheus")
				}
			}()
			m.lastSync = now
		}

		c.Next()
	})
}
