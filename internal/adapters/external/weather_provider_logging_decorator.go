package external

import (
	"context"
	"time"

	"weatherapi.app/internal/ports"
)

// WeatherProviderLoggingDecorator decorates weather providers with structured logging
type WeatherProviderLoggingDecorator struct {
	provider ports.WeatherProvider
	logger   ports.Logger
}

// NewWeatherProviderLoggingDecorator creates a new logging decorator for weather providers
func NewWeatherProviderLoggingDecorator(provider ports.WeatherProvider, logger ports.Logger) ports.WeatherProvider {
	return &WeatherProviderLoggingDecorator{
		provider: provider,
		logger:   logger,
	}
}

// GetCurrentWeather wraps the provider call with structured logging
func (d *WeatherProviderLoggingDecorator) GetCurrentWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	providerName := d.provider.GetProviderName()

	// Log request
	d.logger.Info("Weather API request started",
		ports.F("provider", providerName),
		ports.F("city", city),
		ports.F("event", "request"))

	startTime := time.Now()

	// Execute the actual request
	weatherData, err := d.provider.GetCurrentWeather(ctx, city)
	duration := time.Since(startTime)

	// Log response or error
	if err != nil {
		d.logger.Error("Weather API request failed",
			ports.F("provider", providerName),
			ports.F("city", city),
			ports.F("event", "error"),
			ports.F("duration_ms", duration.Milliseconds()),
			ports.F("error", err.Error()))
		return nil, err
	}

	// Log successful response
	d.logger.Info("Weather API request completed",
		ports.F("provider", providerName),
		ports.F("city", city),
		ports.F("event", "response"),
		ports.F("duration_ms", duration.Milliseconds()),
		ports.F("temperature", weatherData.Temperature),
		ports.F("humidity", weatherData.Humidity),
		ports.F("description", weatherData.Description))

	return weatherData, nil
}

// GetProviderName returns the name of the wrapped provider with logging indication
func (d *WeatherProviderLoggingDecorator) GetProviderName() string {
	return "logged(" + d.provider.GetProviderName() + ")"
}

// WeatherProviderManagerLoggingDecorator decorates the provider manager with logging
type WeatherProviderManagerLoggingDecorator struct {
	manager ports.WeatherProviderManager
	logger  ports.Logger
}

// NewWeatherProviderManagerLoggingDecorator creates a new logging decorator for weather provider manager
func NewWeatherProviderManagerLoggingDecorator(manager ports.WeatherProviderManager, logger ports.Logger) ports.WeatherProviderManager {
	return &WeatherProviderManagerLoggingDecorator{
		manager: manager,
		logger:  logger,
	}
}

// GetWeather wraps the manager call with structured logging
func (d *WeatherProviderManagerLoggingDecorator) GetWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	// Log chain request start
	d.logger.Info("Weather provider chain started",
		ports.F("city", city),
		ports.F("event", "chain_start"))

	startTime := time.Now()

	// Execute the actual request through the chain
	weatherData, err := d.manager.GetWeather(ctx, city)
	duration := time.Since(startTime)

	// Log chain result
	if err != nil {
		d.logger.Error("Weather provider chain failed",
			ports.F("city", city),
			ports.F("event", "chain_error"),
			ports.F("duration_ms", duration.Milliseconds()),
			ports.F("error", err.Error()))
		return nil, err
	}

	// Log successful chain completion
	d.logger.Info("Weather provider chain completed",
		ports.F("city", city),
		ports.F("event", "chain_success"),
		ports.F("duration_ms", duration.Milliseconds()),
		ports.F("temperature", weatherData.Temperature),
		ports.F("humidity", weatherData.Humidity),
		ports.F("description", weatherData.Description))

	return weatherData, nil
}

// GetProviderInfo delegates to the wrapped manager
func (d *WeatherProviderManagerLoggingDecorator) GetProviderInfo() map[string]interface{} {
	info := d.manager.GetProviderInfo()
	info["logging_enabled"] = true
	return info
}
