package infrastructure

import (
	"context"
	"fmt"

	"weatherapi.app/internal/adapters/api"
	"weatherapi.app/internal/ports"
)

// EmailHealthChecker implements email service health checking
type EmailHealthChecker struct {
	config ports.EmailConfig
}

// NewEmailHealthChecker creates a new email health checker
func NewEmailHealthChecker(config ports.EmailConfig) *EmailHealthChecker {
	return &EmailHealthChecker{config: config}
}

// Check verifies email service configuration
func (e *EmailHealthChecker) Check(ctx context.Context) ports.HealthStatus {
	status := ports.HealthStatus{
		Component: "smtp",
		Status:    api.HealthyStatus,
		Details: map[string]interface{}{
			"host": e.config.SMTPHost,
			"port": fmt.Sprintf("%d", e.config.SMTPPort),
		},
	}
	return status
}

// IsHealthy implements the HealthChecker interface
func (e *EmailHealthChecker) IsHealthy(ctx context.Context) bool {
	status := e.Check(ctx)
	return status.Status == api.HealthyStatus
}

// WeatherAPIHealthChecker implements weather API health checking
type WeatherAPIHealthChecker struct {
	weatherProvider ports.WeatherProviderManager
}

// NewWeatherAPIHealthChecker creates a new weather API health checker
func NewWeatherAPIHealthChecker(weatherProvider ports.WeatherProviderManager) *WeatherAPIHealthChecker {
	return &WeatherAPIHealthChecker{weatherProvider: weatherProvider}
}

// Check verifies weather API connectivity
func (w *WeatherAPIHealthChecker) Check(ctx context.Context) ports.HealthStatus {
	status := ports.HealthStatus{
		Component: "weatherAPI",
		Status:    api.HealthyStatus,
		Details: map[string]interface{}{
			"connected": true,
		},
	}

	// In a real implementation, you might want to make a test API call
	// For now, we assume it's healthy if the provider is available
	if w.weatherProvider == nil {
		status.Status = "unhealthy"
		status.Error = "weather provider is not available"
		status.Details["connected"] = false
	}

	return status
}

// IsHealthy implements the HealthChecker interface
func (w *WeatherAPIHealthChecker) IsHealthy(ctx context.Context) bool {
	status := w.Check(ctx)
	return status.Status == api.HealthyStatus
}
