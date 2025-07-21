package infrastructure

import (
	"log/slog"

	"weatherapi.app/internal/ports"
)

// SlogLoggerAdapter implements the Logger port using slog
type SlogLoggerAdapter struct{}

func (l *SlogLoggerAdapter) fieldsToArgs(fields []ports.Field) []interface{} {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	return args
}

// Debug logs a debug message
func (l *SlogLoggerAdapter) Debug(msg string, fields ...ports.Field) {
	args := l.fieldsToArgs(fields)
	slog.Debug(msg, args...)
}

// Info logs an info message
func (l *SlogLoggerAdapter) Info(msg string, fields ...ports.Field) {
	args := l.fieldsToArgs(fields)
	slog.Info(msg, args...)
}

// Warn logs a warning message
func (l *SlogLoggerAdapter) Warn(msg string, fields ...ports.Field) {
	args := l.fieldsToArgs(fields)
	slog.Warn(msg, args...)
}

// Error logs an error message
func (l *SlogLoggerAdapter) Error(msg string, fields ...ports.Field) {
	args := l.fieldsToArgs(fields)
	slog.Error(msg, args...)
}
