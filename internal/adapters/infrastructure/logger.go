package infrastructure

import (
	"log/slog"

	"weatherapi.app/internal/ports"
)

// SlogLoggerAdapter implements the Logger port using slog
type SlogLoggerAdapter struct{}

// Debug logs a debug message
func (l *SlogLoggerAdapter) Debug(msg string, fields ...ports.Field) {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	slog.Debug(msg, args...)
}

// Info logs an info message
func (l *SlogLoggerAdapter) Info(msg string, fields ...ports.Field) {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	slog.Info(msg, args...)
}

// Warn logs a warning message
func (l *SlogLoggerAdapter) Warn(msg string, fields ...ports.Field) {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	slog.Warn(msg, args...)
}

// Error logs an error message
func (l *SlogLoggerAdapter) Error(msg string, fields ...ports.Field) {
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	slog.Error(msg, args...)
}
