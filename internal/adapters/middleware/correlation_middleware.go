package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"weatherapi.app/pkg/logger"
)

const (
	CorrelationIDHeader = "X-Correlation-ID"
	RequestIDHeader     = "X-Request-ID"
)

// CorrelationMiddleware adds correlation ID to requests and sets up request logging
type CorrelationMiddleware struct {
	logger *logger.Logger
}

// NewCorrelationMiddleware creates a new correlation middleware
func NewCorrelationMiddleware(logger *logger.Logger) *CorrelationMiddleware {
	return &CorrelationMiddleware{
		logger: logger,
	}
}

// InjectCorrelationID middleware adds correlation ID to request context and response headers
func (m *CorrelationMiddleware) InjectCorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = c.GetHeader(RequestIDHeader)
		}
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		ctx := logger.WithCorrelationIDContext(c.Request.Context(), correlationID)
		c.Request = c.Request.WithContext(ctx)

		c.Header(CorrelationIDHeader, correlationID)
		c.Header(RequestIDHeader, correlationID)

		requestLogger := m.logger.WithCorrelationID(correlationID).WithOperation(
			c.Request.Method + " " + c.Request.URL.Path,
		)

		requestLogger.Debug("incoming request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"user_agent", c.Request.UserAgent(),
			"remote_addr", c.ClientIP(),
		)

		c.Set("logger", requestLogger)
		c.Set("correlation_id", correlationID)

		c.Next()

		requestLogger.Debug("request completed",
			"status", c.Writer.Status(),
			"response_size", c.Writer.Size(),
		)
	}
}

// GetLoggerFromContext retrieves the correlation-aware logger from gin context
func GetLoggerFromContext(c *gin.Context) *logger.Logger {
	if loggerValue, exists := c.Get("logger"); exists {
		if logger, ok := loggerValue.(*logger.Logger); ok {
			return logger
		}
	}
	return logger.New()
}

// GetCorrelationIDFromContext retrieves correlation ID from gin context
func GetCorrelationIDFromContext(c *gin.Context) string {
	if correlationID, exists := c.Get("correlation_id"); exists {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
}
