// Package config provides configuration functionality for the application
package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the application configuration structure
type Config struct {
	Server     ServerConfig    `split_words:"true"`
	Database   DatabaseConfig  `split_words:"true"`
	Weather    WeatherConfig   `split_words:"true"`
	Email      EmailConfig     `split_words:"true"`
	Scheduler  SchedulerConfig `split_words:"true"`
	AppBaseURL string          `envconfig:"APP_URL" default:"http://localhost:8080"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port int `envconfig:"SERVER_PORT" default:"8080"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST" default:"localhost"`
	Port     int    `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" default:"postgres"`
	Password string `envconfig:"DB_PASSWORD" default:"postgres"`
	Name     string `envconfig:"DB_NAME" default:"weatherapi"`
	SSLMode  string `envconfig:"DB_SSL_MODE" default:"disable"`
}

// GetDSN returns a formatted database connection string
func (c DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// WeatherConfig contains settings for the weather API service
type WeatherConfig struct {
	APIKey  string `envconfig:"WEATHER_API_KEY" required:"true"`
	BaseURL string `envconfig:"WEATHER_API_BASE_URL" default:"https://api.weatherapi.com/v1"`
}

// EmailConfig contains email server and sending settings
type EmailConfig struct {
	SMTPHost     string `envconfig:"EMAIL_SMTP_HOST" default:"smtp.gmail.com"`
	SMTPPort     int    `envconfig:"EMAIL_SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"EMAIL_SMTP_USERNAME" required:"true"`
	SMTPPassword string `envconfig:"EMAIL_SMTP_PASSWORD" required:"true"`
	FromName     string `envconfig:"EMAIL_FROM_NAME" default:"Weather API"`
	FromAddress  string `envconfig:"EMAIL_FROM_ADDRESS" default:"no-reply@weatherapi.app"`
}

// SchedulerConfig contains settings for the background task scheduler
type SchedulerConfig struct {
	HourlyInterval int `envconfig:"HOURLY_INTERVAL" default:"60"`
	DailyInterval  int `envconfig:"DAILY_INTERVAL" default:"1440"`
}

// LoadConfig loads application configuration from environment variables
func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}

	return &config, nil
}
