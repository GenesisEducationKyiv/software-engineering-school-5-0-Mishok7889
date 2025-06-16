package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"weatherapi.app/app"
)

func main() {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] No .env file found or error loading it")
	}

	// Initialize configuration displayer for debugging (optional)
	configDisplayer := app.NewConfigDisplayer()

	// Uncomment the following lines to display environment variables and configuration
	// configDisplayer.PrintAllEnvVars()

	// Create and initialize the application
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize application: %v", err)
	}

	// Print configuration for debugging (optional)
	configDisplayer.PrintConfig(application.Config())

	// Set up graceful shutdown
	setupGracefulShutdown(application)

	// Start the application
	log.Println("[INFO] Starting Weather Forecast API...")
	if err := application.Start(); err != nil {
		log.Fatalf("[FATAL] Failed to start application: %v", err)
	}
}

func setupGracefulShutdown(app *app.Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("[INFO] Received shutdown signal...")
		if err := app.Shutdown(); err != nil {
			log.Printf("[ERROR] Error during shutdown: %v\n", err)
		}
		os.Exit(0)
	}()
}
