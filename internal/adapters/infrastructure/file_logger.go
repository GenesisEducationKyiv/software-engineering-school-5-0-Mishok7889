package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"weatherapi.app/internal/ports"
)

// FileLoggerAdapter implements structured JSON logging to files
type FileLoggerAdapter struct {
	filePath string
	mutex    sync.Mutex
}

// NewFileLoggerAdapter creates a new file logger adapter
func NewFileLoggerAdapter(logPath string) (ports.Logger, error) {
	if logPath == "" {
		return nil, fmt.Errorf("log file path cannot be empty")
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &FileLoggerAdapter{
		filePath: logPath,
	}, nil
}

// Debug logs a debug message to file
func (f *FileLoggerAdapter) Debug(msg string, fields ...ports.Field) {
	f.writeLogEntry("DEBUG", msg, fields...)
}

// Info logs an info message to file
func (f *FileLoggerAdapter) Info(msg string, fields ...ports.Field) {
	f.writeLogEntry("INFO", msg, fields...)
}

// Warn logs a warning message to file
func (f *FileLoggerAdapter) Warn(msg string, fields ...ports.Field) {
	f.writeLogEntry("WARN", msg, fields...)
}

// Error logs an error message to file
func (f *FileLoggerAdapter) Error(msg string, fields ...ports.Field) {
	f.writeLogEntry("ERROR", msg, fields...)
}

// writeLogEntry writes a structured JSON log entry to file
func (f *FileLoggerAdapter) writeLogEntry(level, msg string, fields ...ports.Field) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Create log entry with timestamp and level
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	// Add fields to log entry
	for _, field := range fields {
		logEntry[field.Key] = field.Value
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		// If JSON marshaling fails, write a simple error message
		f.writeRawLog(fmt.Sprintf("ERROR: failed to marshal log entry: %v", err))
		return
	}

	// Write to file
	f.writeRawLog(string(jsonData))
}

// writeRawLog writes raw log data to file
func (f *FileLoggerAdapter) writeRawLog(data string) {
	file, err := os.OpenFile(f.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open the file, there's not much we can do
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log to stderr as fallback
			fmt.Fprintf(os.Stderr, "failed to close log file: %v\n", closeErr)
		}
	}()

	if _, err := file.WriteString(data + "\n"); err != nil {
		// Log to stderr as fallback
		fmt.Fprintf(os.Stderr, "failed to write log entry: %v\n", err)
	}
}
