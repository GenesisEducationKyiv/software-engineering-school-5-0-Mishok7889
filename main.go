package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"weatherapi.app/app"
)

func main() {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found or error loading it")
	}

	// Initialize configuration displayer for debugging (optional)
	configDisplayer := app.NewConfigDisplayer()

	// Uncomment the following lines to display environment variables and configuration
	// configDisplayer.PrintAllEnvVars()

	// Create and initialize the application
	application, err := app.NewApplication()
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Print configuration for debugging (optional)
	configDisplayer.PrintConfig(application.Config())

	// Set up graceful shutdown
	setupGracefulShutdown(application)

	// Start the application
	slog.Info("Starting Weather Forecast API...")
	if err := application.Start(); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}
}

func setupGracefulShutdown(app *app.Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		slog.Info("Received shutdown signal...")
		if err := app.Shutdown(); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
		os.Exit(0)
	}()
}
