package ports

import "context"

// HealthChecker defines the contract for component health checking
type HealthChecker interface {
	Check(ctx context.Context) HealthStatus
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker interface {
	HealthChecker
}

// WeatherAPIHealthChecker checks weather API connectivity
type WeatherAPIHealthChecker interface {
	HealthChecker
}

// EmailHealthChecker checks email service configuration
type EmailHealthChecker interface {
	HealthChecker
}

// SystemHealthChecker aggregates all health checks
type SystemHealthChecker interface {
	CheckAll(ctx context.Context) map[string]HealthStatus
}
