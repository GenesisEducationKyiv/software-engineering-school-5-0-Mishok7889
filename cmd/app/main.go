package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"weatherapi.app/internal/app"
)

func main() {
	// Load environment variables from .env file if present
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found or error loading it")
	}

	// Create application with dependency injection
	application, err := app.NewApplication()
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully")
	slog.Info("Server configuration", "port", application.Config().Server.Port, "baseURL", application.Config().AppBaseURL)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupGracefulShutdown(cancel, application)

	// Start the application
	slog.Info("Starting Weather Forecast API...")
	if err := application.Start(ctx); err != nil {
		slog.Error("Failed to start application", "error", err)
		os.Exit(1)
	}
}

func setupGracefulShutdown(cancel context.CancelFunc, app *app.Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		slog.Info("Received shutdown signal...")

		// Cancel the context to stop the application
		cancel()

		// Give the application time to shut down gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := app.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during graceful shutdown", "error", err)
		}

		os.Exit(0)
	}()
}
