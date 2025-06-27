package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"weatherapi.app/errors"
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

// LoadConfig loads and validates application configuration from environment variables
func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, errors.NewConfigurationError("error processing config", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	if err := c.Weather.Validate(); err != nil {
		return err
	}
	if err := c.Email.Validate(); err != nil {
		return err
	}
	if err := c.Scheduler.Validate(); err != nil {
		return err
	}
	if err := c.validateAppBaseURL(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateAppBaseURL() error {
	if c.AppBaseURL == "" {
		return errors.NewConfigurationError("APP_URL cannot be empty", nil)
	}
	if !strings.HasPrefix(c.AppBaseURL, "http://") && !strings.HasPrefix(c.AppBaseURL, "https://") {
		return errors.NewConfigurationError("APP_URL must start with http:// or https://", nil)
	}
	return nil
}

// Validate checks server configuration
func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return errors.NewConfigurationError("SERVER_PORT must be between 1 and 65535", nil)
	}
	return nil
}

// ValidateSSLMode validates the SSL mode configuration
func (d *DatabaseConfig) ValidateSSLMode() error {
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range validSSLModes {
		if d.SSLMode == mode {
			return nil
		}
	}
	return errors.NewConfigurationError(
		fmt.Sprintf("DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", ")), nil)
}

// Validate checks database configuration
func (d *DatabaseConfig) Validate() error {
	if d.Host == "" {
		return errors.NewConfigurationError("DB_HOST cannot be empty", nil)
	}
	if d.Port < 1 || d.Port > 65535 {
		return errors.NewConfigurationError("DB_PORT must be between 1 and 65535", nil)
	}
	if d.User == "" {
		return errors.NewConfigurationError("DB_USER cannot be empty", nil)
	}
	if d.Name == "" {
		return errors.NewConfigurationError("DB_NAME cannot be empty", nil)
	}
	if err := d.ValidateSSLMode(); err != nil {
		return err
	}
	return nil
}

// Validate checks weather API configuration
func (w *WeatherConfig) Validate() error {
	if w.APIKey == "" {
		return errors.NewConfigurationError("WEATHER_API_KEY is required", nil)
	}
	if w.BaseURL == "" {
		return errors.NewConfigurationError("WEATHER_API_BASE_URL cannot be empty", nil)
	}
	if !strings.HasPrefix(w.BaseURL, "http://") && !strings.HasPrefix(w.BaseURL, "https://") {
		return errors.NewConfigurationError("WEATHER_API_BASE_URL must start with http:// or https://", nil)
	}
	return nil
}

// Validate checks email configuration
func (e *EmailConfig) Validate() error {
	if e.SMTPHost == "" {
		return errors.NewConfigurationError("EMAIL_SMTP_HOST cannot be empty", nil)
	}
	if e.SMTPPort < 1 || e.SMTPPort > 65535 {
		return errors.NewConfigurationError("EMAIL_SMTP_PORT must be between 1 and 65535", nil)
	}
	if e.SMTPUsername == "" {
		return errors.NewConfigurationError("EMAIL_SMTP_USERNAME is required", nil)
	}
	if e.SMTPPassword == "" {
		return errors.NewConfigurationError("EMAIL_SMTP_PASSWORD is required", nil)
	}
	if e.FromName == "" {
		return errors.NewConfigurationError("EMAIL_FROM_NAME cannot be empty", nil)
	}
	if e.FromAddress == "" {
		return errors.NewConfigurationError("EMAIL_FROM_ADDRESS cannot be empty", nil)
	}
	if !strings.Contains(e.FromAddress, "@") {
		return errors.NewConfigurationError("EMAIL_FROM_ADDRESS must be a valid email address", nil)
	}
	return nil
}

// Validate checks scheduler configuration
func (s *SchedulerConfig) Validate() error {
	if s.HourlyInterval < 1 {
		return errors.NewConfigurationError("HOURLY_INTERVAL must be at least 1 minute", nil)
	}
	if s.DailyInterval < 1 {
		return errors.NewConfigurationError("DAILY_INTERVAL must be at least 1 minute", nil)
	}
	if s.HourlyInterval > 1440 {
		return errors.NewConfigurationError("HOURLY_INTERVAL cannot exceed 1440 minutes (24 hours)", nil)
	}
	if s.DailyInterval > 10080 {
		return errors.NewConfigurationError("DAILY_INTERVAL cannot exceed 10080 minutes (7 days)", nil)
	}
	return nil
}
