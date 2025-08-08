package logger

import (
	"context"
	"fmt"
	"time"
)

// ContextualLogger provides context-aware logging for service operations
type ContextualLogger struct {
	baseLogger *Logger
	service    string
	component  string
}

// NewContextualLogger creates a logger that extracts context information
func NewContextualLogger(baseLogger *Logger, service, component string) *ContextualLogger {
	return &ContextualLogger{
		baseLogger: baseLogger.WithService(service).WithComponent(component),
		service:    service,
		component:  component,
	}
}

// FromContext creates a correlation-aware logger from context
func (c *ContextualLogger) FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return c.baseLogger
	}

	correlationID := GetCorrelationID(ctx)
	if correlationID != "" {
		return c.baseLogger.WithCorrelationID(correlationID)
	}

	return c.baseLogger
}

// WithOperation adds operation context to the logger
func (c *ContextualLogger) WithOperation(ctx context.Context, operation string) *Logger {
	return c.FromContext(ctx).WithOperation(operation)
}

// LogOperationStart logs the beginning of an operation with timing
func (c *ContextualLogger) LogOperationStart(ctx context.Context, operation string, params ...interface{}) *Logger {
	logger := c.WithOperation(ctx, operation)
	logger.Info(fmt.Sprintf("%s started", operation), params...)
	return logger
}

// LogOperationComplete logs successful completion of an operation
func (c *ContextualLogger) LogOperationComplete(ctx context.Context, operation string, duration time.Duration, params ...interface{}) {
	logger := c.WithOperation(ctx, operation)
	args := append([]interface{}{"duration_ms", duration.Milliseconds()}, params...)
	logger.Info(fmt.Sprintf("%s completed", operation), args...)
}

// LogOperationError logs operation failure
func (c *ContextualLogger) LogOperationError(ctx context.Context, operation string, err error, duration time.Duration, params ...interface{}) {
	logger := c.WithOperation(ctx, operation)
	args := append([]interface{}{"duration_ms", duration.Milliseconds(), "error", err}, params...)
	logger.Error(fmt.Sprintf("%s failed", operation), args...)
}

// BusinessEvent logs important business events that should never be sampled
func (c *ContextualLogger) BusinessEvent(ctx context.Context, event string, params ...interface{}) {
	logger := c.FromContext(ctx)
	logger.LogBusinessEvent(event, params...)
}

// SecurityEvent logs security-related events
func (c *ContextualLogger) SecurityEvent(ctx context.Context, event string, params ...interface{}) {
	logger := c.FromContext(ctx)
	logger.LogSecurityEvent(event, params...)
}

// WithField creates a logger with an additional field from context
func (c *ContextualLogger) WithField(ctx context.Context, key string, value interface{}) *Logger {
	return c.FromContext(ctx).WithField(key, value)
}

// WithFields creates a logger with additional fields from context
func (c *ContextualLogger) WithFields(ctx context.Context, fields map[string]interface{}) *Logger {
	return c.FromContext(ctx).WithFields(fields)
}

// Debug logs debug message with context
func (c *ContextualLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	c.FromContext(ctx).Debug(msg, args...)
}

// Info logs info message with context
func (c *ContextualLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	c.FromContext(ctx).Info(msg, args...)
}

// Warn logs warning message with context
func (c *ContextualLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	c.FromContext(ctx).Warn(msg, args...)
}

// Error logs error message with context
func (c *ContextualLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	c.FromContext(ctx).Error(msg, args...)
}

// OperationTimer helps track operation timing with automatic logging
type OperationTimer struct {
	contextLogger *ContextualLogger
	ctx           context.Context
	operation     string
	startTime     time.Time
	logger        *Logger
}

// StartOperation creates a timer for tracking operation duration
func (c *ContextualLogger) StartOperation(ctx context.Context, operation string, params ...interface{}) *OperationTimer {
	logger := c.LogOperationStart(ctx, operation, params...)

	return &OperationTimer{
		contextLogger: c,
		ctx:           ctx,
		operation:     operation,
		startTime:     time.Now(),
		logger:        logger,
	}
}

// Complete logs successful operation completion
func (t *OperationTimer) Complete(params ...interface{}) {
	duration := time.Since(t.startTime)
	t.contextLogger.LogOperationComplete(t.ctx, t.operation, duration, params...)
}

// Fail logs operation failure
func (t *OperationTimer) Fail(err error, params ...interface{}) {
	duration := time.Since(t.startTime)
	t.contextLogger.LogOperationError(t.ctx, t.operation, err, duration, params...)
}

// Logger returns the logger for additional logging during operation
func (t *OperationTimer) Logger() *Logger {
	return t.logger
}

// Duration returns current operation duration
func (t *OperationTimer) Duration() time.Duration {
	return time.Since(t.startTime)
}

// ServiceLoggerFactory creates contextual loggers for different services
type ServiceLoggerFactory struct {
	baseLogger *Logger
}

// NewServiceLoggerFactory creates a factory for service loggers
func NewServiceLoggerFactory(baseLogger *Logger) *ServiceLoggerFactory {
	return &ServiceLoggerFactory{
		baseLogger: baseLogger,
	}
}

// Weather creates a contextual logger for weather service operations
func (f *ServiceLoggerFactory) Weather(component string) *ContextualLogger {
	return NewContextualLogger(f.baseLogger, WeatherServiceName, component)
}

// User creates a contextual logger for user service operations
func (f *ServiceLoggerFactory) User(component string) *ContextualLogger {
	return NewContextualLogger(f.baseLogger, UserServiceName, component)
}

// Subscription creates a contextual logger for subscription service operations
func (f *ServiceLoggerFactory) Subscription(component string) *ContextualLogger {
	return NewContextualLogger(f.baseLogger, SubscriptionServiceName, component)
}

// Notification creates a contextual logger for notification service operations
func (f *ServiceLoggerFactory) Notification(component string) *ContextualLogger {
	return NewContextualLogger(f.baseLogger, NotificationServiceName, component)
}

// Gateway creates a contextual logger for API gateway operations
func (f *ServiceLoggerFactory) Gateway(component string) *ContextualLogger {
	return NewContextualLogger(f.baseLogger, APIGatewayServiceName, component)
}

// UseCaseHelper provides common patterns for use case logging
type UseCaseHelper struct {
	contextLogger *ContextualLogger
}

// NewUseCaseHelper creates a helper for use case logging patterns
func NewUseCaseHelper(contextLogger *ContextualLogger) *UseCaseHelper {
	return &UseCaseHelper{
		contextLogger: contextLogger,
	}
}

// ExecuteWithLogging executes a function with automatic operation logging
func (u *UseCaseHelper) ExecuteWithLogging(
	ctx context.Context,
	operation string,
	fn func(context.Context, *Logger) error,
	params ...interface{},
) error {
	timer := u.contextLogger.StartOperation(ctx, operation, params...)

	err := fn(ctx, timer.Logger())

	if err != nil {
		timer.Fail(err)
		return err
	}

	timer.Complete()
	return nil
}

// ExecuteWithResult executes a function with automatic operation logging and result
func (u *UseCaseHelper) ExecuteWithResult(
	ctx context.Context,
	operation string,
	fn func(context.Context, *Logger) (interface{}, error),
	params ...interface{},
) (interface{}, error) {
	var result interface{}
	timer := u.contextLogger.StartOperation(ctx, operation, params...)

	result, err := fn(ctx, timer.Logger())

	if err != nil {
		timer.Fail(err)
		return result, err
	}

	timer.Complete()
	return result, nil
}

// Repository operations helpers
type RepositoryHelper struct {
	contextLogger *ContextualLogger
}

// NewRepositoryHelper creates a helper for repository logging patterns
func NewRepositoryHelper(contextLogger *ContextualLogger) *RepositoryHelper {
	return &RepositoryHelper{
		contextLogger: contextLogger,
	}
}

// LogQuery logs database query operations
func (r *RepositoryHelper) LogQuery(ctx context.Context, query string, params ...interface{}) *OperationTimer {
	args := append([]interface{}{"query", query}, params...)
	return r.contextLogger.StartOperation(ctx, "database_query", args...)
}

// LogCacheOperation logs cache operations
func (r *RepositoryHelper) LogCacheOperation(ctx context.Context, operation, key string) *OperationTimer {
	return r.contextLogger.StartOperation(ctx, "cache_"+operation, "key", key)
}

// LogExternalCall logs external service calls
func (r *RepositoryHelper) LogExternalCall(ctx context.Context, service, endpoint string) *OperationTimer {
	return r.contextLogger.StartOperation(ctx, "external_call", "service", service, "endpoint", endpoint)
}
