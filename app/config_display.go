package app

import (
	"log"
	"os"
	"sort"
	"strings"

	"weatherapi.app/config"
)

// ConfigDisplayer handles configuration and environment variable display
type ConfigDisplayer struct{}

// NewConfigDisplayer creates a new configuration displayer
func NewConfigDisplayer() *ConfigDisplayer {
	return &ConfigDisplayer{}
}

// PrintConfig prints all fields in the configuration
func (cd *ConfigDisplayer) PrintConfig(cfg *config.Config) {
	log.Println("==== APPLICATION CONFIGURATION ====")

	log.Printf("SERVER:\n")
	log.Printf("  Port: %d\n", cfg.Server.Port)

	log.Printf("\nDATABASE:\n")
	log.Printf("  Host: %s\n", cfg.Database.Host)
	log.Printf("  Port: %d\n", cfg.Database.Port)
	log.Printf("  User: %s\n", cfg.Database.User)
	log.Printf("  Password: %s\n", cd.maskString(cfg.Database.Password))
	log.Printf("  Name: %s\n", cfg.Database.Name)
	log.Printf("  SSLMode: %s\n", cfg.Database.SSLMode)

	log.Printf("\nWEATHER API:\n")
	log.Printf("  API Key: %s\n", cd.maskString(cfg.Weather.APIKey))
	log.Printf("  Base URL: %s\n", cfg.Weather.BaseURL)

	log.Printf("\nEMAIL:\n")
	log.Printf("  SMTP Host: %s\n", cfg.Email.SMTPHost)
	log.Printf("  SMTP Port: %d\n", cfg.Email.SMTPPort)
	log.Printf("  SMTP Username: %s\n", cfg.Email.SMTPUsername)
	log.Printf("  SMTP Password: %s\n", cd.maskString(cfg.Email.SMTPPassword))
	log.Printf("  From Name: %s\n", cfg.Email.FromName)
	log.Printf("  From Address: %s\n", cfg.Email.FromAddress)

	log.Printf("\nSCHEDULER:\n")
	log.Printf("  Hourly Interval: %d minutes\n", cfg.Scheduler.HourlyInterval)
	log.Printf("  Daily Interval: %d minutes\n", cfg.Scheduler.DailyInterval)

	log.Printf("\nAPP BASE URL: %s\n", cfg.AppBaseURL)

	log.Println("===================================")
}

// PrintAllEnvVars prints all environment variables available to the application
func (cd *ConfigDisplayer) PrintAllEnvVars() {
	log.Println("==== ENVIRONMENT VARIABLES ====")

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

		log.Printf("%s=%s\n", key, value)
	}

	log.Println("===============================")
}

// maskString masks sensitive information like passwords and API keys
func (cd *ConfigDisplayer) maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	visible := len(s) / 4
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
