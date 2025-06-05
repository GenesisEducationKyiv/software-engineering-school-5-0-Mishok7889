// Package config provides configuration functionality for the application
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config represents the application configuration structure
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Weather    WeatherConfig
	Email      EmailConfig
	Scheduler  SchedulerConfig
	AppBaseURL string
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port int
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// GetDSN returns a formatted database connection string
func (c DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// WeatherConfig contains settings for the weather API service
type WeatherConfig struct {
	APIKey  string
	BaseURL string
}

// EmailConfig contains email server and sending settings
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromName     string
	FromAddress  string
}

// SchedulerConfig contains settings for the background task scheduler
type SchedulerConfig struct {
	HourlyInterval int
	DailyInterval  int
}

// LoadConfig loads application configuration from file or environment
func LoadConfig() (*Config, error) {
	dbPort, _ := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	serverPort, _ := strconv.Atoi(getEnvOrDefault("SERVER_PORT", "8080"))
	hourlyInterval, _ := strconv.Atoi(getEnvOrDefault("HOURLY_INTERVAL", "60"))
	dailyInterval, _ := strconv.Atoi(getEnvOrDefault("DAILY_INTERVAL", "1440"))
	smtpPort, _ := strconv.Atoi(getEnvOrDefault("EMAIL_SMTP_PORT", "587"))

	config := &Config{
		Server: ServerConfig{
			Port: serverPort,
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			Name:     getEnvOrDefault("DB_NAME", "weatherapi"),
			SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
		},
		Weather: WeatherConfig{
			APIKey:  getEnvOrDefault("WEATHER_API_KEY", ""),
			BaseURL: getEnvOrDefault("WEATHER_API_BASE_URL", "https://api.weatherapi.com/v1"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnvOrDefault("EMAIL_SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     smtpPort,
			SMTPUsername: getEnvOrDefault("EMAIL_SMTP_USERNAME", ""),
			SMTPPassword: getEnvOrDefault("EMAIL_SMTP_PASSWORD", ""),
			FromName:     getEnvOrDefault("EMAIL_FROM_NAME", "Weather API"),
			FromAddress:  getEnvOrDefault("EMAIL_FROM_ADDRESS", "no-reply@weatherapi.app"),
		},
		Scheduler: SchedulerConfig{
			HourlyInterval: hourlyInterval,
			DailyInterval:  dailyInterval,
		},
		AppBaseURL: getEnvOrDefault("APP_URL", "http://localhost:8080"),
	}

	if config.Weather.APIKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable is required")
	}

	if config.Email.SMTPUsername == "" || config.Email.SMTPPassword == "" {
		return nil, fmt.Errorf("EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD environment variables are required")
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
