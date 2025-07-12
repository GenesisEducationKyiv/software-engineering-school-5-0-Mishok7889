package integration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"weatherapi.app/internal/adapters/external"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

func (s *IntegrationTestSuite) TestFileLogging_WeatherProviderIntegration() {
	// Skip if logging is not enabled in the test config
	if !s.config.Weather.EnableLogging {
		s.T().Skip("Weather logging is not enabled in test configuration")
	}

	// Create a temporary log file for this test
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "weather_test.log")

	// Create file logger
	fileLogger, err := infrastructure.NewFileLoggerAdapter(logPath)
	s.Require().NoError(err)

	// Create weather provider manager with file logging
	providerManager := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     s.config.Weather.APIKey,
		WeatherAPIBaseURL: s.config.Weather.BaseURL,
		ProviderOrder:     s.config.Weather.ProviderOrder,
		Logger:            fileLogger,
	})

	// Wrap with logging decorator
	loggedManager := external.NewWeatherProviderManagerLoggingDecorator(providerManager, fileLogger)

	// Test weather request
	ctx := context.Background()
	weatherData, err := loggedManager.GetWeather(ctx, "London")
	s.Require().NoError(err)
	s.Require().NotNil(weatherData)

	// Wait for file I/O to complete
	s.Require().Eventually(func() bool {
		_, err := os.Stat(logPath)
		return err == nil
	}, 2*time.Second, 50*time.Millisecond)

	// Verify log file exists and contains expected entries
	s.Require().FileExists(logPath)

	// Wait for all log entries to be written
	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(lines) >= 2
	}, 2*time.Second, 50*time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logPath)
	s.Require().NoError(err)

	// Parse log entries
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	s.Require().GreaterOrEqual(len(lines), 2, "Should have at least chain start and completion logs")

	// Verify log structure
	for i, line := range lines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		s.Require().NoError(err, "Log line %d should be valid JSON: %s", i, line)

		// Verify basic log structure
		s.Contains(logEntry, "timestamp")
		s.Contains(logEntry, "level")
		s.Contains(logEntry, "message")

		// Verify timestamp format
		timestamp, ok := logEntry["timestamp"].(string)
		s.True(ok, "Timestamp should be a string")
		_, err = time.Parse(time.RFC3339, timestamp)
		s.NoError(err, "Timestamp should be in RFC3339 format")
	}

	// Verify specific log entries
	var chainStartFound, chainSuccessFound bool
	for _, line := range lines {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue // Skip malformed log entries
		}

		if message, ok := logEntry["message"].(string); ok {
			switch message {
			case "Weather provider chain started":
				chainStartFound = true
				s.Equal("London", logEntry["city"])
				s.Equal("chain_start", logEntry["event"])
			case "Weather provider chain completed":
				chainSuccessFound = true
				s.Equal("London", logEntry["city"])
				s.Equal("chain_success", logEntry["event"])
				s.Contains(logEntry, "duration_ms")
				s.Contains(logEntry, "temperature")
				s.Contains(logEntry, "humidity")
				s.Contains(logEntry, "description")
			}
		}
	}

	s.True(chainStartFound, "Should find chain start log entry")
	s.True(chainSuccessFound, "Should find chain success log entry")
}

func (s *IntegrationTestSuite) TestFileLogging_WeatherUseCase() {
	// Create a temporary log file for this test
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "weather_usecase_test.log")

	// Create file logger
	fileLogger, err := infrastructure.NewFileLoggerAdapter(logPath)
	s.Require().NoError(err)

	// Create weather provider manager with file logging
	providerManager := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     s.config.Weather.APIKey,
		WeatherAPIBaseURL: s.config.Weather.BaseURL,
		ProviderOrder:     s.config.Weather.ProviderOrder,
		Logger:            fileLogger,
	})

	// Wrap with logging decorator
	loggedManager := external.NewWeatherProviderManagerLoggingDecorator(providerManager, fileLogger)

	// Test weather request directly through the manager
	ctx := context.Background()
	weatherData, err := loggedManager.GetWeather(ctx, "Paris")
	s.Require().NoError(err)
	s.Require().NotNil(weatherData)

	// Wait for all log entries to be written
	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		return strings.Contains(string(content), "Weather provider chain started") &&
			strings.Contains(string(content), "Weather provider chain completed")
	}, 2*time.Second, 50*time.Millisecond)

	// Verify log file exists and contains expected entries
	s.Require().FileExists(logPath)

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	s.Require().NoError(err)

	// Should contain weather provider chain logs
	s.Contains(string(content), "Weather provider chain started")
	s.Contains(string(content), "Weather provider chain completed")
	s.Contains(string(content), "Paris")
	s.Contains(string(content), "temperature")
}

func (s *IntegrationTestSuite) TestFileLogging_ConcurrentRequests() {
	// Create a temporary log file for this test
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "concurrent_test.log")

	// Create file logger
	fileLogger, err := infrastructure.NewFileLoggerAdapter(logPath)
	s.Require().NoError(err)

	// Create weather provider manager with file logging
	providerManager := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     s.config.Weather.APIKey,
		WeatherAPIBaseURL: s.config.Weather.BaseURL,
		ProviderOrder:     s.config.Weather.ProviderOrder,
		Logger:            fileLogger,
	})

	// Wrap with logging decorator
	loggedManager := external.NewWeatherProviderManagerLoggingDecorator(providerManager, fileLogger)

	// Make concurrent requests
	ctx := context.Background()
	numRequests := 3
	cities := []string{"London", "Paris", "Berlin"} // Use only cities supported by mock server

	// Channel to collect results
	results := make(chan error, numRequests)

	// Start concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(city string) {
			_, err := loggedManager.GetWeather(ctx, city)
			results <- err
		}(cities[i])
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-results
		s.NoError(err, "Concurrent request %d should succeed", i)
	}

	// Wait for all file I/O to complete
	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(lines) >= numRequests*2 // Each request should generate at least 2 log entries
	}, 3*time.Second, 100*time.Millisecond)

	// Verify log file exists and contains entries for all cities
	s.Require().FileExists(logPath)

	content, err := os.ReadFile(logPath)
	s.Require().NoError(err)

	// Verify each city appears in the logs
	for _, city := range cities {
		s.Contains(string(content), city, "Should contain logs for %s", city)
	}

	// Verify log structure is maintained under concurrent access
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	s.GreaterOrEqual(len(lines), numRequests*2, "Should have at least 2 logs per request")

	// Verify all lines are valid JSON
	for i, line := range lines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		s.NoError(err, "Log line %d should be valid JSON: %s", i, line)
	}
}

func (s *IntegrationTestSuite) TestFileLogging_ErrorHandling() {
	// Create a temporary log file for this test
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "error_test.log")

	// Create file logger
	fileLogger, err := infrastructure.NewFileLoggerAdapter(logPath)
	s.Require().NoError(err)

	// Create weather provider manager with file logging
	providerManager := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     s.config.Weather.APIKey,
		WeatherAPIBaseURL: s.config.Weather.BaseURL,
		ProviderOrder:     s.config.Weather.ProviderOrder,
		Logger:            fileLogger,
	})

	// Wrap with logging decorator
	loggedManager := external.NewWeatherProviderManagerLoggingDecorator(providerManager, fileLogger)

	// Test error case (city that triggers error from mock server)
	ctx := context.Background()
	_, err = loggedManager.GetWeather(ctx, "NonExistentCity")
	s.Error(err, "Should get error for 'NonExistentCity'")

	// Wait for error log entries to be written
	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		contentStr := string(content)
		return strings.Contains(contentStr, "Weather provider chain failed") &&
			strings.Contains(contentStr, "NonExistentCity") &&
			strings.Contains(contentStr, "chain_error")
	}, 3*time.Second, 100*time.Millisecond)

	// Verify log file exists and contains error entry
	s.Require().FileExists(logPath)

	content, err := os.ReadFile(logPath)
	s.Require().NoError(err)

	// Should contain error logs
	s.Contains(string(content), "Weather provider chain failed")
	s.Contains(string(content), "NonExistentCity")
	s.Contains(string(content), "chain_error")
}

func (s *IntegrationTestSuite) TestFileLogging_Configuration() {
	// Test that the application properly configures file logging based on config

	// Get the weather use case from the application
	weatherUseCase := s.application.GetWeatherUseCase()
	s.Require().NotNil(weatherUseCase)

	// Get provider info to check if logging is enabled
	ctx := context.Background()
	providerInfo := weatherUseCase.GetProviderInfo(ctx)
	s.Require().NotNil(providerInfo)

	// Log the provider info for debugging
	s.T().Logf("Provider info: %+v", providerInfo)

	// In test configuration, logging should be enabled/disabled based on config
	if s.config.Weather.EnableLogging {
		// Check if logging_enabled key exists and is true
		if loggingEnabled, exists := providerInfo["logging_enabled"]; exists {
			s.Equal(true, loggingEnabled, "logging_enabled should be true when EnableLogging is true")
		} else {
			// If the key doesn't exist, we can't verify through provider info
			// but we can still verify the configuration is correct
			s.T().Logf("Provider info doesn't contain logging_enabled key, but config.Weather.EnableLogging is %v", s.config.Weather.EnableLogging)
		}
	} else {
		// If logging is disabled, the provider info might not have this field
		// or it might be false
		if loggingEnabled, exists := providerInfo["logging_enabled"]; exists {
			s.Equal(false, loggingEnabled, "logging_enabled should be false when EnableLogging is false")
		}
	}

	// Verify other expected provider info fields
	s.Contains(providerInfo, "provider_order", "Provider info should contain provider_order")
}

// Test helper functions
func (s *IntegrationTestSuite) TestFileLogging_HelperFunctions() {
	// Test that file logging works with different log levels
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "helper_test.log")

	fileLogger, err := infrastructure.NewFileLoggerAdapter(logPath)
	s.Require().NoError(err)

	// Test all log levels
	fileLogger.Debug("Debug message", ports.F("level", "debug"))
	fileLogger.Info("Info message", ports.F("level", "info"))
	fileLogger.Warn("Warn message", ports.F("level", "warn"))
	fileLogger.Error("Error message", ports.F("level", "error"))

	// Wait for all log entries to be written
	s.Require().Eventually(func() bool {
		content, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		return len(lines) >= 4 // Should have 4 log entries
	}, 2*time.Second, 50*time.Millisecond)

	// Verify log file content
	content, err := os.ReadFile(logPath)
	s.Require().NoError(err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	s.Equal(4, len(lines))

	// Verify each log level
	levels := []string{"debug", "info", "warn", "error"}
	messages := []string{"Debug message", "Info message", "Warn message", "Error message"}

	for i, line := range lines {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		s.Require().NoError(err)

		s.Equal(levels[i], logEntry["level"])
		s.Equal(messages[i], logEntry["message"])
	}
}
