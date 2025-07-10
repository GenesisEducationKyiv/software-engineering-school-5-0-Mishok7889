package providers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"weatherapi.app/models"
)

type FileLoggerImpl struct {
	filePath string
	mutex    sync.Mutex
}

func NewFileLogger(logPath string) (FileLogger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	return &FileLoggerImpl{
		filePath: logPath,
	}, nil
}

func (l *FileLoggerImpl) LogRequest(providerName, city string) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"provider":  providerName,
		"event":     "request",
		"city":      city,
	}

	l.writeLog(logEntry)
}

// LogResponse logs a successful weather response
func (l *FileLoggerImpl) LogResponse(providerName, city string, response *models.WeatherResponse, duration time.Duration) {
	logEntry := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"provider":    providerName,
		"event":       "response",
		"city":        city,
		"duration_ms": duration.Milliseconds(),
		"response": map[string]interface{}{
			"temperature": response.Temperature,
			"humidity":    response.Humidity,
			"description": response.Description,
		},
	}

	l.writeLog(logEntry)
}

// LogError logs an error during weather request
func (l *FileLoggerImpl) LogError(providerName, city string, err error, duration time.Duration) {
	logEntry := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"provider":    providerName,
		"event":       "error",
		"city":        city,
		"duration_ms": duration.Milliseconds(),
		"error":       err.Error(),
	}

	l.writeLog(logEntry)
}

func (l *FileLoggerImpl) writeLog(entry map[string]interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		slog.Error("open log file", "error", err)
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("close log file", "error", closeErr)
		}
	}()

	jsonData, err := json.Marshal(entry)
	if err != nil {
		slog.Error("marshal log entry", "error", err)
		return
	}

	if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
		slog.Error("write log entry", "error", err)
	}
}
