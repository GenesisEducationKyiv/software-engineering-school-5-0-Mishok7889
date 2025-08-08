package logger

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	correlationIDKey = "correlation_id"
	serviceKey       = "service"
	operationKey     = "operation"
	componentKey     = "component"
)

// SamplingConfig defines sampling rates for different log levels
type SamplingConfig struct {
	DebugSampleRate float64 // 0.0 to 1.0, where 1.0 means no sampling
	InfoSampleRate  float64
	WarnSampleRate  float64
	ErrorSampleRate float64 // Should typically be 1.0 (no sampling)
}

// DefaultSamplingConfig returns sensible defaults for production
func DefaultSamplingConfig() SamplingConfig {
	return SamplingConfig{
		DebugSampleRate: 0.01, // 1% of debug logs
		InfoSampleRate:  0.1,  // 10% of info logs
		WarnSampleRate:  0.8,  // 80% of warn logs
		ErrorSampleRate: 1.0,  // 100% of error logs
	}
}

// DevSamplingConfig returns sampling config suitable for development
func DevSamplingConfig() SamplingConfig {
	return SamplingConfig{
		DebugSampleRate: 1.0, // No sampling in dev
		InfoSampleRate:  1.0,
		WarnSampleRate:  1.0,
		ErrorSampleRate: 1.0,
	}
}

// Config holds logger configuration
type Config struct {
	Level        slog.Level
	ServiceName  string
	Environment  string
	Sampling     SamplingConfig
	IsProduction bool
}

// Logger wraps slog with sampling and structured logging capabilities
type Logger struct {
	*slog.Logger
	config  Config
	rand    *rand.Rand
	randMux sync.Mutex
}

// New creates a new logger with default configuration
func New() *Logger {
	return NewWithConfig(Config{
		Level:        slog.LevelInfo,
		ServiceName:  "unknown",
		Environment:  "development",
		Sampling:     DevSamplingConfig(),
		IsProduction: false,
	})
}

// NewWithLevel creates a new logger with specified level
func NewWithLevel(level slog.Level) *Logger {
	return NewWithConfig(Config{
		Level:        level,
		ServiceName:  "unknown",
		Environment:  "development",
		Sampling:     DevSamplingConfig(),
		IsProduction: false,
	})
}

// NewWithConfig creates a new logger with full configuration
func NewWithConfig(config Config) *Logger {
	handlerOpts := &slog.HandlerOptions{
		Level: config.Level,
	}

	var handler slog.Handler
	if config.IsProduction {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	baseLogger := slog.New(handler).With(
		serviceKey, config.ServiceName,
		"environment", config.Environment,
	)

	return &Logger{
		Logger: baseLogger,
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// shouldSample determines if a log should be sampled based on level and config
func (l *Logger) shouldSample(level slog.Level) bool {
	var sampleRate float64

	switch level {
	case slog.LevelDebug:
		sampleRate = l.config.Sampling.DebugSampleRate
	case slog.LevelInfo:
		sampleRate = l.config.Sampling.InfoSampleRate
	case slog.LevelWarn:
		sampleRate = l.config.Sampling.WarnSampleRate
	case slog.LevelError:
		sampleRate = l.config.Sampling.ErrorSampleRate
	default:
		return true
	}

	if sampleRate >= 1.0 {
		return true
	}

	l.randMux.Lock()
	defer l.randMux.Unlock()
	return l.rand.Float64() < sampleRate
}

// WithField returns a logger with a pre-set field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.With(key, value),
		config: l.config,
		rand:   l.rand,
	}
}

// WithFields returns a logger with multiple pre-set fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: l.With(args...),
		config: l.config,
		rand:   l.rand,
	}
}

// WithCorrelationID returns a logger with correlation ID
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return l.WithField(correlationIDKey, correlationID)
}

// WithService returns a logger with service context
func (l *Logger) WithService(serviceName string) *Logger {
	return l.WithField(serviceKey, serviceName)
}

// WithOperation returns a logger with operation context
func (l *Logger) WithOperation(operation string) *Logger {
	return l.WithField(operationKey, operation)
}

// WithComponent returns a logger with component context
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithField(componentKey, component)
}

// WithContext extracts correlation ID from context and adds it to logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		return l.WithCorrelationID(correlationID)
	}
	return l
}

// Debug logs debug level message with sampling
func (l *Logger) Debug(msg string, args ...interface{}) {
	if !l.shouldSample(slog.LevelDebug) {
		return
	}
	l.Logger.Debug(msg, args...)
}

// Info logs info level message with sampling
func (l *Logger) Info(msg string, args ...interface{}) {
	if !l.shouldSample(slog.LevelInfo) {
		return
	}
	l.Logger.Info(msg, args...)
}

// Warn logs warn level message with sampling
func (l *Logger) Warn(msg string, args ...interface{}) {
	if !l.shouldSample(slog.LevelWarn) {
		return
	}
	l.Logger.Warn(msg, args...)
}

// Error logs error level message with sampling (typically no sampling)
func (l *Logger) Error(msg string, args ...interface{}) {
	if !l.shouldSample(slog.LevelError) {
		return
	}
	l.Logger.Error(msg, args...)
}

// Context utilities for correlation ID management

type contextKey string

const correlationIDContextKey contextKey = "correlation_id"

// WithCorrelationIDContext adds correlation ID to context
func WithCorrelationIDContext(ctx context.Context, correlationID string) context.Context {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return context.WithValue(ctx, correlationIDContextKey, correlationID)
}

// GetCorrelationID extracts correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if correlationID, ok := ctx.Value(correlationIDContextKey).(string); ok {
		return correlationID
	}
	return ""
}

// GetOrCreateCorrelationID gets correlation ID from context or creates new one
func GetOrCreateCorrelationID(ctx context.Context) (context.Context, string) {
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		return ctx, correlationID
	}
	correlationID := uuid.New().String()
	return WithCorrelationIDContext(ctx, correlationID), correlationID
}

// Business event helpers that bypass sampling for important events

// LogBusinessEvent logs important business events that should never be sampled
func (l *Logger) LogBusinessEvent(msg string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf("[BUSINESS_EVENT] %s", msg), args...)
}

// LogSecurityEvent logs security-related events that should never be sampled
func (l *Logger) LogSecurityEvent(msg string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf("[SECURITY_EVENT] %s", msg), args...)
}

// LogCriticalError logs critical errors that should never be sampled
func (l *Logger) LogCriticalError(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf("[CRITICAL_ERROR] %s", msg), args...)
}

// Service-specific logger constructors

// NewServiceLogger creates a logger configured for a specific service
func NewServiceLogger(serviceName, environment string, isProduction bool) *Logger {
	level := slog.LevelInfo
	if !isProduction {
		level = slog.LevelDebug
	}

	sampling := DevSamplingConfig()
	if isProduction {
		sampling = DefaultSamplingConfig()
	}

	return NewWithConfig(Config{
		Level:        level,
		ServiceName:  serviceName,
		Environment:  environment,
		Sampling:     sampling,
		IsProduction: isProduction,
	})
}

// Service name constants
const (
	WeatherServiceName      = "weather-service"
	UserServiceName         = "user-service"
	SubscriptionServiceName = "subscription-service"
	NotificationServiceName = "notification-service"
	APIGatewayServiceName   = "api-gateway"
)
