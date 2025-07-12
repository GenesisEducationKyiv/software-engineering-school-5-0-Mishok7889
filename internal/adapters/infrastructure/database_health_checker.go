package infrastructure

import (
	"context"

	"gorm.io/gorm"
	"weatherapi.app/internal/ports"
)

// DatabaseHealthChecker implements database health checking
type DatabaseHealthChecker struct {
	db *gorm.DB
}

// NewDatabaseHealthChecker creates a new database health checker
func NewDatabaseHealthChecker(db *gorm.DB) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{db: db}
}

// Check verifies database connectivity
func (d *DatabaseHealthChecker) Check(ctx context.Context) ports.HealthStatus {
	status := ports.HealthStatus{
		Component: "database",
		Details:   make(map[string]interface{}),
	}

	if d.db == nil {
		status.Status = "unhealthy"
		status.Error = "database instance is nil"
		return status
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		status.Status = "unhealthy"
		status.Error = "failed to get underlying database connection"
		return status
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		status.Status = "unhealthy"
		status.Error = err.Error()
		return status
	}

	status.Status = "healthy"
	status.Details["connected"] = true
	return status
}
