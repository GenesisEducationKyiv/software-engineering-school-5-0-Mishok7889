package app

import (
	"log/slog"
	"os"
	"sort"
	"strings"

	"weatherapi.app/config"
)

const (
	// MinVisibleChars defines the minimum number of characters to show when masking
	MinVisibleChars = 4
)

// ConfigDisplayer handles configuration and environment variable display
type ConfigDisplayer struct{}

// NewConfigDisplayer creates a new configuration displayer
func NewConfigDisplayer() *ConfigDisplayer {
	return &ConfigDisplayer{}
}

// PrintConfig prints all fields in the configuration
func (cd *ConfigDisplayer) PrintConfig(cfg *config.Config) {
	slog.Info("Application Configuration",
		"server_port", cfg.Server.Port,
		"db_host", cfg.Database.Host,
		"db_port", cfg.Database.Port,
		"db_user", cfg.Database.User,
		"db_password", cd.maskString(cfg.Database.Password),
		"db_name", cfg.Database.Name,
		"db_ssl_mode", cfg.Database.SSLMode,
		"weather_api_key", cd.maskString(cfg.Weather.APIKey),
		"weather_base_url", cfg.Weather.BaseURL,
		"email_smtp_host", cfg.Email.SMTPHost,
		"email_smtp_port", cfg.Email.SMTPPort,
		"email_smtp_username", cfg.Email.SMTPUsername,
		"email_smtp_password", cd.maskString(cfg.Email.SMTPPassword),
		"email_from_name", cfg.Email.FromName,
		"email_from_address", cfg.Email.FromAddress,
		"scheduler_hourly_interval", cfg.Scheduler.HourlyInterval,
		"scheduler_daily_interval", cfg.Scheduler.DailyInterval,
		"app_base_url", cfg.AppBaseURL,
	)
}

// PrintAllEnvVars prints all environment variables available to the application
func (cd *ConfigDisplayer) PrintAllEnvVars() {
	slog.Debug("Environment Variables")

	envVars := os.Environ()
	sort.Strings(envVars)

	for _, env := range envVars {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		if cd.isSensitive(key) {
			value = cd.maskString(value)
		}

		slog.Debug("Environment variable", "key", key, "value", value)
	}
}

// maskString masks sensitive information like passwords and API keys
func (cd *ConfigDisplayer) maskString(s string) string {
	if len(s) <= MinVisibleChars {
		return "****"
	}
	visible := len(s) / MinVisibleChars
	return s[:visible] + strings.Repeat("*", len(s)-visible)
}

// isSensitive checks if an environment variable key is considered sensitive
func (cd *ConfigDisplayer) isSensitive(key string) bool {
	sensitiveKeys := []string{
		"API_KEY", "PASSWORD", "SECRET", "TOKEN", "KEY", "PASS", "PWD",
	}

	key = strings.ToUpper(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(key, sensitive) {
			return true
		}
	}

	return false
}
