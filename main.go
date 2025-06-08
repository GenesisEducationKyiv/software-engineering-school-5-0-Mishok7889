// Package main is the entry point for the weather subscription service application
package main

import (
	"log"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"weatherapi.app/api"
	"weatherapi.app/config"
	"weatherapi.app/database"
	"weatherapi.app/scheduler"
)

// printConfig prints all fields in the configuration
func printConfig(cfg *config.Config) {
	log.Println("==== APPLICATION CONFIGURATION ====")

	// Print Server config
	log.Printf("SERVER:\n")
	log.Printf("  Port: %d\n", cfg.Server.Port)

	// Print Database config
	log.Printf("\nDATABASE:\n")
	log.Printf("  Host: %s\n", cfg.Database.Host)
	log.Printf("  Port: %d\n", cfg.Database.Port)
	log.Printf("  User: %s\n", cfg.Database.User)
	log.Printf("  Password: %s\n", maskString(cfg.Database.Password))
	log.Printf("  Name: %s\n", cfg.Database.Name)
	log.Printf("  SSLMode: %s\n", cfg.Database.SSLMode)

	// Print Weather config
	log.Printf("\nWEATHER API:\n")
	log.Printf("  API Key: %s\n", maskString(cfg.Weather.APIKey))
	log.Printf("  Base URL: %s\n", cfg.Weather.BaseURL)

	// Print Email config
	log.Printf("\nEMAIL:\n")
	log.Printf("  SMTP Host: %s\n", cfg.Email.SMTPHost)
	log.Printf("  SMTP Port: %d\n", cfg.Email.SMTPPort)
	log.Printf("  SMTP Username: %s\n", cfg.Email.SMTPUsername)
	log.Printf("  SMTP Password: %s\n", maskString(cfg.Email.SMTPPassword))
	log.Printf("  From Name: %s\n", cfg.Email.FromName)
	log.Printf("  From Address: %s\n", cfg.Email.FromAddress)

	// Print Scheduler config
	log.Printf("\nSCHEDULER:\n")
	log.Printf("  Hourly Interval: %d minutes\n", cfg.Scheduler.HourlyInterval)
	log.Printf("  Daily Interval: %d minutes\n", cfg.Scheduler.DailyInterval)

	// Print App Base URL
	log.Printf("\nAPP BASE URL: %s\n", cfg.AppBaseURL)

	log.Println("===================================")
}

// maskString masks sensitive information like passwords and API keys
func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	visible := len(s) / 4
	return s[:visible] + strings.Repeat("*", len(s)-visible)
}

// printAllEnvVars prints all environment variables available to the application
func printAllEnvVars() {
	log.Println("==== ENVIRONMENT VARIABLES ====")

	// Get all environment variables
	envVars := os.Environ()

	// Sort them for better readability
	sort.Strings(envVars)

	// Print each one
	for _, env := range envVars {
		pair := strings.SplitN(env, "=", 2)
		key := pair[0]
		value := pair[1]

		// Mask sensitive values
		if isSensitive(key) {
			value = maskString(value)
		}

		log.Printf("%s=%s\n", key, value)
	}

	log.Println("===============================")
}

// isSensitive checks if an environment variable key is considered sensitive
func isSensitive(key string) bool {
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

func main() {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it")
	}

	// Print all environment variables
	printAllEnvVars()

	// Initialize configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print the loaded configuration
	printConfig(cfg)

	// Initialize database
	db, err := database.InitDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize and start scheduler for sending weather updates
	schedulerService := scheduler.NewScheduler(db, cfg)
	go schedulerService.Start()

	// Initialize and start the API server
	server := api.NewServer(db, cfg)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
