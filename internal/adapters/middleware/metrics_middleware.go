package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics names for general API metrics
const (
	APIRequestsTotalMetric  = "api_requests_total"
	APIResponsesTotalMetric = "api_responses_total"
)

// Metrics label keys
const (
	EndpointLabelKey = "endpoint"
	MethodLabelKey   = "method"
	StatusLabelKey   = "status"
)

// Metrics label values
const (
	SuccessStatus = "success"
)

// MetricsCollector interface for metrics operations
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
}

// MetricsMiddleware provides general API metrics collection
type MetricsMiddleware struct {
	collector MetricsCollector
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(collector MetricsCollector) *MetricsMiddleware {
	return &MetricsMiddleware{
		collector: collector,
	}
}

// CollectRequestMetrics middleware collects request and response metrics
func (m *MetricsMiddleware) CollectRequestMetrics() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		m.collector.IncrementCounter(APIRequestsTotalMetric, map[string]string{
			EndpointLabelKey: extractEndpointFromPath(c.FullPath()),
			MethodLabelKey:   c.Request.Method,
		})

		c.Next()

		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			m.collector.IncrementCounter(APIResponsesTotalMetric, map[string]string{
				EndpointLabelKey: extractEndpointFromPath(c.FullPath()),
				StatusLabelKey:   SuccessStatus,
			})
		}

		_ = time.Since(start) // For potential latency metrics in the future
	})
}

// extractEndpointFromPath extracts endpoint name from the full path
func extractEndpointFromPath(fullPath string) string {
	switch fullPath {
	case "/api/weather":
		return "weather"
	case "/api/subscribe":
		return "subscribe"
	case "/api/confirm/:token":
		return "confirm"
	case "/api/unsubscribe/:token":
		return "unsubscribe"
	case "/api/health":
		return "health"
	case "/api/debug":
		return "debug"
	case "/api/metrics":
		return "metrics"
	default:
		return "unknown"
	}
}
