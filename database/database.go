// Package database provides database connection and migration functionality
package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"weatherapi.app/config"
	"weatherapi.app/models"
)

// InitDB initializes the database connection
func InitDB(config config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.GetDSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return db, nil
}

// RunMigrations executes database schema migrations
func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Subscription{},
		&models.Token{},
	)
}

// CloseDB safely closes the database connection
func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
