package external

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"weatherapi.app/internal/ports"
)

// Simple test using concrete implementations instead of mocks
func TestWeatherProviderLoggingDecorator_BasicFunctionality(t *testing.T) {
	// Create a test provider
	testProvider := &testWeatherProvider{
		name:     "test-provider",
		response: &ports.WeatherData{Temperature: 22.0, Humidity: 55.0, Description: "Test weather"},
	}

	// Create a test logger that captures log entries
	testLogger := &testLogger{entries: []logEntry{}}

	// Create decorator
	decorator := NewWeatherProviderLoggingDecorator(testProvider, testLogger)

	// Execute
	ctx := context.Background()
	result, err := decorator.GetCurrentWeather(ctx, "TestCity")

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 22.0, result.Temperature)

	// Verify logging
	assert.Equal(t, 2, len(testLogger.entries))

	// Verify request log
	requestLog := testLogger.entries[0]
	assert.Equal(t, "INFO", requestLog.level)
	assert.Equal(t, "Weather API request started", requestLog.message)
	assert.Equal(t, "test-provider", requestLog.fields["provider"])
	assert.Equal(t, "TestCity", requestLog.fields["city"])
	assert.Equal(t, "request", requestLog.fields["event"])

	// Verify response log
	responseLog := testLogger.entries[1]
	assert.Equal(t, "INFO", responseLog.level)
	assert.Equal(t, "Weather API request completed", responseLog.message)
	assert.Equal(t, "test-provider", responseLog.fields["provider"])
	assert.Equal(t, "TestCity", responseLog.fields["city"])
	assert.Equal(t, "response", responseLog.fields["event"])
	assert.Equal(t, 22.0, responseLog.fields["temperature"])
	assert.Equal(t, 55.0, responseLog.fields["humidity"])
	assert.Equal(t, "Test weather", responseLog.fields["description"])
	assert.Contains(t, responseLog.fields, "duration_ms")

	// Verify provider name
	assert.Equal(t, "logged(test-provider)", decorator.GetProviderName())
}

func TestWeatherProviderLoggingDecorator_ErrorHandling(t *testing.T) {
	// Create a test provider that returns an error
	testProvider := &testWeatherProvider{
		name: "error-provider",
		err:  errors.New("API rate limit exceeded"),
	}

	// Create a test logger that captures log entries
	testLogger := &testLogger{entries: []logEntry{}}

	// Create decorator
	decorator := NewWeatherProviderLoggingDecorator(testProvider, testLogger)

	// Execute
	ctx := context.Background()
	result, err := decorator.GetCurrentWeather(ctx, "InvalidCity")

	// Verify results
	assert.Error(t, err)
	assert.Equal(t, "API rate limit exceeded", err.Error())
	assert.Nil(t, result)

	// Verify logging
	assert.Equal(t, 2, len(testLogger.entries))

	// Verify request log
	requestLog := testLogger.entries[0]
	assert.Equal(t, "INFO", requestLog.level)
	assert.Equal(t, "Weather API request started", requestLog.message)

	// Verify error log
	errorLog := testLogger.entries[1]
	assert.Equal(t, "ERROR", errorLog.level)
	assert.Equal(t, "Weather API request failed", errorLog.message)
	assert.Equal(t, "error-provider", errorLog.fields["provider"])
	assert.Equal(t, "InvalidCity", errorLog.fields["city"])
	assert.Equal(t, "error", errorLog.fields["event"])
	assert.Equal(t, "API rate limit exceeded", errorLog.fields["error"])
	assert.Contains(t, errorLog.fields, "duration_ms")
}

func TestWeatherProviderLoggingDecorator_DurationTracking(t *testing.T) {
	// Create a test provider with artificial delay
	testProvider := &testWeatherProviderWithDelay{
		name:     "slow-provider",
		response: &ports.WeatherData{Temperature: 20.0, Humidity: 60.0, Description: "Slow weather"},
		delay:    10 * time.Millisecond,
	}

	// Create a test logger
	testLogger := &testLogger{entries: []logEntry{}}

	// Create decorator
	decorator := NewWeatherProviderLoggingDecorator(testProvider, testLogger)

	// Execute
	ctx := context.Background()
	result, err := decorator.GetCurrentWeather(ctx, "SlowCity")

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify logging
	assert.Equal(t, 2, len(testLogger.entries))

	// Verify response log has duration >= 10ms
	responseLog := testLogger.entries[1]
	duration, ok := responseLog.fields["duration_ms"].(int64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, duration, int64(10))
}

func TestWeatherProviderManagerLoggingDecorator_BasicFunctionality(t *testing.T) {
	// Create a test manager
	testManager := &testWeatherProviderManager{
		response: &ports.WeatherData{Temperature: 18.0, Humidity: 65.0, Description: "Manager weather"},
	}

	// Create a test logger
	testLogger := &testLogger{entries: []logEntry{}}

	// Create decorator
	decorator := NewWeatherProviderManagerLoggingDecorator(testManager, testLogger)

	// Execute
	ctx := context.Background()
	result, err := decorator.GetWeather(ctx, "Berlin")

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 18.0, result.Temperature)

	// Verify logging
	assert.Equal(t, 2, len(testLogger.entries))

	// Verify chain start log
	chainStartLog := testLogger.entries[0]
	assert.Equal(t, "INFO", chainStartLog.level)
	assert.Equal(t, "Weather provider chain started", chainStartLog.message)
	assert.Equal(t, "Berlin", chainStartLog.fields["city"])
	assert.Equal(t, "chain_start", chainStartLog.fields["event"])

	// Verify chain success log
	chainSuccessLog := testLogger.entries[1]
	assert.Equal(t, "INFO", chainSuccessLog.level)
	assert.Equal(t, "Weather provider chain completed", chainSuccessLog.message)
	assert.Equal(t, "Berlin", chainSuccessLog.fields["city"])
	assert.Equal(t, "chain_success", chainSuccessLog.fields["event"])
	assert.Equal(t, 18.0, chainSuccessLog.fields["temperature"])
	assert.Equal(t, 65.0, chainSuccessLog.fields["humidity"])
	assert.Equal(t, "Manager weather", chainSuccessLog.fields["description"])
	assert.Contains(t, chainSuccessLog.fields, "duration_ms")
}

func TestWeatherProviderManagerLoggingDecorator_GetProviderInfo(t *testing.T) {
	// Create a test manager
	testManager := &testWeatherProviderManager{
		providerInfo: map[string]interface{}{
			"total_providers": 2,
			"provider_order":  []string{"weatherapi", "openweathermap"},
		},
	}

	// Create a test logger
	testLogger := &testLogger{entries: []logEntry{}}

	// Create decorator
	decorator := NewWeatherProviderManagerLoggingDecorator(testManager, testLogger)

	// Execute
	result := decorator.GetProviderInfo()

	// Verify results
	assert.Equal(t, 2, result["total_providers"])
	assert.Equal(t, []string{"weatherapi", "openweathermap"}, result["provider_order"])
	assert.Equal(t, true, result["logging_enabled"])
}

// Test helper structs
type testWeatherProvider struct {
	name     string
	response *ports.WeatherData
	err      error
}

func (p *testWeatherProvider) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.response, nil
}

func (p *testWeatherProvider) GetProviderName() string {
	return p.name
}

type testWeatherProviderWithDelay struct {
	name     string
	response *ports.WeatherData
	err      error
	delay    time.Duration
}

func (p *testWeatherProviderWithDelay) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	select {
	case <-time.After(p.delay):
		if p.err != nil {
			return nil, p.err
		}
		return p.response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *testWeatherProviderWithDelay) GetProviderName() string {
	return p.name
}

type testWeatherProviderManager struct {
	response     *ports.WeatherData
	err          error
	providerInfo map[string]interface{}
}

func (m *testWeatherProviderManager) GetWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *testWeatherProviderManager) GetProviderInfo() map[string]interface{} {
	if m.providerInfo != nil {
		return m.providerInfo
	}
	return map[string]interface{}{
		"total_providers": 1,
		"provider_order":  []string{"test"},
	}
}

type logEntry struct {
	level   string
	message string
	fields  map[string]interface{}
}

type testLogger struct {
	entries []logEntry
}

func (l *testLogger) Debug(msg string, fields ...ports.Field) {
	l.addEntry("DEBUG", msg, fields...)
}

func (l *testLogger) Info(msg string, fields ...ports.Field) {
	l.addEntry("INFO", msg, fields...)
}

func (l *testLogger) Warn(msg string, fields ...ports.Field) {
	l.addEntry("WARN", msg, fields...)
}

func (l *testLogger) Error(msg string, fields ...ports.Field) {
	l.addEntry("ERROR", msg, fields...)
}

func (l *testLogger) addEntry(level, message string, fields ...ports.Field) {
	fieldMap := make(map[string]interface{})
	for _, field := range fields {
		fieldMap[field.Key] = field.Value
	}

	l.entries = append(l.entries, logEntry{
		level:   level,
		message: message,
		fields:  fieldMap,
	})
}

// Benchmark test
func BenchmarkWeatherProviderLoggingDecorator(b *testing.B) {
	// Create test components
	testProvider := &testWeatherProvider{
		name:     "benchmark-provider",
		response: &ports.WeatherData{Temperature: 20.0, Humidity: 60.0, Description: "Benchmark weather"},
	}

	testLogger := &testLogger{entries: []logEntry{}}

	decorator := NewWeatherProviderLoggingDecorator(testProvider, testLogger)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.Background()
			_, _ = decorator.GetCurrentWeather(ctx, "BenchmarkCity")
		}
	})
}
