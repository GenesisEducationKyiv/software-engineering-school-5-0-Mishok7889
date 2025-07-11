package infrastructure

import (
	"context"

	"weatherapi.app/internal/ports"
)

// SystemHealthChecker aggregates all health checks
type SystemHealthChecker struct {
	databaseChecker   ports.DatabaseHealthChecker
	weatherAPIChecker ports.WeatherAPIHealthChecker
	emailChecker      ports.EmailHealthChecker
	configProvider    ports.ConfigProvider
}

// SystemHealthCheckerConfig holds the configuration for creating a system health checker
type SystemHealthCheckerConfig struct {
	DatabaseChecker   ports.DatabaseHealthChecker
	WeatherAPIChecker ports.WeatherAPIHealthChecker
	EmailChecker      ports.EmailHealthChecker
	ConfigProvider    ports.ConfigProvider
}

// NewSystemHealthChecker creates a new system health checker
func NewSystemHealthChecker(config SystemHealthCheckerConfig) *SystemHealthChecker {
	return &SystemHealthChecker{
		databaseChecker:   config.DatabaseChecker,
		weatherAPIChecker: config.WeatherAPIChecker,
		emailChecker:      config.EmailChecker,
		configProvider:    config.ConfigProvider,
	}
}

// CheckAll performs health checks on all components
func (s *SystemHealthChecker) CheckAll(ctx context.Context) map[string]ports.HealthStatus {
	results := make(map[string]ports.HealthStatus)

	if s.databaseChecker != nil {
		results["database"] = s.databaseChecker.Check(ctx)
	}

	if s.weatherAPIChecker != nil {
		results["weatherAPI"] = s.weatherAPIChecker.Check(ctx)
	}

	if s.emailChecker != nil {
		results["smtp"] = s.emailChecker.Check(ctx)
	}

	// Add config information
	if s.configProvider != nil {
		appConfig := s.configProvider.GetAppConfig()
		results["config"] = ports.HealthStatus{
			Component: "config",
			Status:    "healthy",
			Details: map[string]interface{}{
				"appBaseURL": appConfig.BaseURL,
			},
		}
	}

	return results
}
