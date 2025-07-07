package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog for consistent logging across the application
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// NewWithLevel creates a new logger with specified level
func NewWithLevel(level slog.Level) *Logger {
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})),
	}
}

// WithField returns a logger with a pre-set field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		Logger: l.With(key, value),
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
	}
}
