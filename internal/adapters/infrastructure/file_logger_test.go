package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/ports"
)

func TestFileLoggerAdapter_NewFileLoggerAdapter(t *testing.T) {
	tests := []struct {
		name        string
		logPath     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_path",
			logPath:     "test_logs/weather.log",
			expectError: false,
		},
		{
			name:        "nested_path",
			logPath:     "test_logs/nested/deep/weather.log",
			expectError: false,
		},
		{
			name:        "empty_path",
			logPath:     "",
			expectError: true,
			errorMsg:    "log file path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			if tt.logPath != "" {
				defer func() {
					if err := os.RemoveAll(filepath.Dir(tt.logPath)); err != nil {
						t.Logf("Failed to clean up test directory: %v", err)
					}
				}()
			}

			logger, err := NewFileLoggerAdapter(tt.logPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, logger)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, logger)

				// Verify directory was created
				assert.DirExists(t, filepath.Dir(tt.logPath))
			}
		})
	}
}

func TestFileLoggerAdapter_LogLevels(t *testing.T) {
	// Test all log levels
	tests := []struct {
		level   string
		message string
		fields  []ports.Field
	}{
		{
			level:   "DEBUG",
			message: "Debug message",
			fields:  []ports.Field{ports.F("key", "value")},
		},
		{
			level:   "INFO",
			message: "Info message",
			fields:  []ports.Field{ports.F("provider", "weatherapi")},
		},
		{
			level:   "WARN",
			message: "Warning message",
			fields:  []ports.Field{ports.F("city", "London"), ports.F("error", "timeout")},
		},
		{
			level:   "ERROR",
			message: "Error message",
			fields:  []ports.Field{ports.F("provider", "openweathermap"), ports.F("duration_ms", 5000)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			// Create temporary log file for each test
			tempDir := t.TempDir()
			logPath := filepath.Join(tempDir, "test.log")

			logger, err := NewFileLoggerAdapter(logPath)
			require.NoError(t, err)

			// Log the message based on level
			switch tt.level {
			case "DEBUG":
				logger.Debug(tt.message, tt.fields...)
			case "INFO":
				logger.Info(tt.message, tt.fields...)
			case "WARN":
				logger.Warn(tt.message, tt.fields...)
			case "ERROR":
				logger.Error(tt.message, tt.fields...)
			}

			// Wait for file I/O to complete
			require.Eventually(t, func() bool {
				_, err := os.Stat(logPath)
				return err == nil
			}, time.Second, 10*time.Millisecond)

			// Read the log file
			content, err := os.ReadFile(logPath)
			require.NoError(t, err)

			// Parse JSON
			var logEntry map[string]interface{}
			err = json.Unmarshal(content, &logEntry)
			require.NoError(t, err)

			// Verify basic structure
			assert.Equal(t, tt.level, logEntry["level"])
			assert.Equal(t, tt.message, logEntry["message"])
			assert.NotEmpty(t, logEntry["timestamp"])

			// Verify timestamp format
			timestamp, ok := logEntry["timestamp"].(string)
			assert.True(t, ok)
			_, err = time.Parse(time.RFC3339, timestamp)
			assert.NoError(t, err)

			// Verify fields
			for _, field := range tt.fields {
				// JSON unmarshaling converts all numbers to float64
				if expectedInt, ok := field.Value.(int); ok {
					assert.Equal(t, float64(expectedInt), logEntry[field.Key])
				} else {
					assert.Equal(t, field.Value, logEntry[field.Key])
				}
			}
		})
	}
}

func TestFileLoggerAdapter_StructuredLogging(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(t, err)

	// Test structured logging with weather data
	logger.Info("Weather API response",
		ports.F("provider", "weatherapi"),
		ports.F("city", "London"),
		ports.F("event", "response"),
		ports.F("duration_ms", 1250),
		ports.F("temperature", 15.5),
		ports.F("humidity", 76.0),
		ports.F("description", "Partly cloudy"))

	// Wait for file I/O to complete
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logPath)
		return err == nil && len(content) > 0
	}, time.Second, 10*time.Millisecond)

	// Read and verify the log entry
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	require.NoError(t, err)

	// Verify all fields are present and correct
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "Weather API response", logEntry["message"])
	assert.Equal(t, "weatherapi", logEntry["provider"])
	assert.Equal(t, "London", logEntry["city"])
	assert.Equal(t, "response", logEntry["event"])
	assert.Equal(t, float64(1250), logEntry["duration_ms"])
	assert.Equal(t, 15.5, logEntry["temperature"])
	assert.Equal(t, 76.0, logEntry["humidity"])
	assert.Equal(t, "Partly cloudy", logEntry["description"])
}

func TestFileLoggerAdapter_ConcurrentLogging(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "concurrent.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(t, err)

	// Number of concurrent goroutines
	numGoroutines := 10
	messagesPerGoroutine := 5

	// Channel to synchronize goroutines
	done := make(chan bool, numGoroutines)

	// Start concurrent logging
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info(fmt.Sprintf("Message from goroutine %d", goroutineID),
					ports.F("goroutine_id", goroutineID),
					ports.F("message_id", j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Wait for all file I/O to complete
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(lines) >= numGoroutines
	}, time.Second, 10*time.Millisecond)

	// Read and verify log entries
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := numGoroutines * messagesPerGoroutine
	assert.Equal(t, expectedLines, len(lines))

	// Verify each line is valid JSON
	for i, line := range lines {
		var logEntry map[string]interface{}
		err = json.Unmarshal([]byte(line), &logEntry)
		assert.NoError(t, err, "Line %d should be valid JSON: %s", i, line)

		// Verify structure
		assert.Equal(t, "INFO", logEntry["level"])
		assert.Contains(t, logEntry["message"], "Message from goroutine")
		assert.Contains(t, logEntry, "goroutine_id")
		assert.Contains(t, logEntry, "message_id")
	}
}

func TestFileLoggerAdapter_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "permissions.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(t, err)

	// Write a log entry
	logger.Info("Test message")

	// Wait for file I/O to complete
	require.Eventually(t, func() bool {
		_, err := os.Stat(logPath)
		return err == nil
	}, time.Second, 10*time.Millisecond)

	// Check file exists
	fileInfo, err := os.Stat(logPath)
	require.NoError(t, err)

	// On Windows, file permissions work differently
	// We just verify the file is readable and writable
	if runtime.GOOS == "windows" {
		// On Windows, just verify the file exists and is not a directory
		assert.False(t, fileInfo.IsDir())
		assert.Greater(t, fileInfo.Size(), int64(0))

		// Verify we can read the file
		content, err := os.ReadFile(logPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
	} else {
		// On Unix-like systems, verify file permissions are 0644
		assert.Equal(t, os.FileMode(0644), fileInfo.Mode().Perm())
	}
}

func TestFileLoggerAdapter_AppendMode(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "append.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(t, err)

	// Write first message
	logger.Info("First message")

	// Write second message
	logger.Info("Second message")

	// Wait for file I/O to complete
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(lines) >= 2
	}, time.Second, 10*time.Millisecond)

	// Read file content
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, 2, len(lines))

	// Verify both messages are present
	var firstEntry, secondEntry map[string]interface{}
	err = json.Unmarshal([]byte(lines[0]), &firstEntry)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(lines[1]), &secondEntry)
	require.NoError(t, err)

	assert.Equal(t, "First message", firstEntry["message"])
	assert.Equal(t, "Second message", secondEntry["message"])
}

func TestFileLoggerAdapter_InvalidJSONHandling(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "invalid.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(t, err)

	// Create a field that can't be marshaled to JSON
	// (channels can't be marshaled to JSON)
	ch := make(chan int)

	// This should handle the error gracefully
	logger.Info("Test message", ports.F("channel", ch))

	// Wait for file I/O to complete
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logPath)
		return err == nil && len(content) > 0
	}, time.Second, 10*time.Millisecond)

	// Read the log file
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// Should contain an error message about JSON marshaling
	assert.Contains(t, string(content), "failed to marshal log entry")
}

func TestFileLoggerAdapter_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "deep", "nested", "path", "weather.log")

	logger, err := NewFileLoggerAdapter(nestedPath)
	require.NoError(t, err)

	// Write a log entry
	logger.Info("Test message")

	// Wait for file I/O to complete
	require.Eventually(t, func() bool {
		return assert.FileExists(t, nestedPath)
	}, time.Second, 10*time.Millisecond)

	// Verify the nested directory structure was created
	assert.DirExists(t, filepath.Dir(nestedPath))
	assert.FileExists(t, nestedPath)
}

// Benchmark tests for performance
func BenchmarkFileLoggerAdapter_Info(b *testing.B) {
	tempDir := b.TempDir()
	logPath := filepath.Join(tempDir, "benchmark.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(b, err)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Benchmark message",
				ports.F("provider", "weatherapi"),
				ports.F("city", "London"),
				ports.F("temperature", 15.5))
		}
	})
}

func BenchmarkFileLoggerAdapter_Error(b *testing.B) {
	tempDir := b.TempDir()
	logPath := filepath.Join(tempDir, "benchmark_error.log")

	logger, err := NewFileLoggerAdapter(logPath)
	require.NoError(b, err)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Error("Benchmark error message",
				ports.F("provider", "openweathermap"),
				ports.F("error", "connection timeout"),
				ports.F("duration_ms", 5000))
		}
	})
}
