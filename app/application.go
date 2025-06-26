package app

import (
	"log/slog"

	"gorm.io/gorm"
	"weatherapi.app/api"
	"weatherapi.app/config"
	"weatherapi.app/database"
	"weatherapi.app/providers"
	"weatherapi.app/repository"
	"weatherapi.app/scheduler"
	"weatherapi.app/service"
)

// Application represents the main application with all its dependencies
type Application struct {
	config    *config.Config
	db        *gorm.DB
	server    *api.Server
	scheduler *scheduler.Scheduler
}

// NewApplication creates and initializes a new application instance
func NewApplication() (*Application, error) {
	app := &Application{}

	if err := app.loadConfiguration(); err != nil {
		return nil, err
	}

	if err := app.initializeDatabase(); err != nil {
		return nil, err
	}

	if err := app.initializeServices(); err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) loadConfiguration() error {
	slog.Info("Loading configuration...")

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return err
	}

	app.config = cfg
	slog.Info("Configuration loaded successfully")
	return nil
}

func (app *Application) initializeDatabase() error {
	slog.Info("Initializing database...")

	db, err := database.InitDB(app.config.Database)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		return err
	}

	if err := database.RunMigrations(db); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		return err
	}

	app.db = db
	slog.Info("Database initialized successfully")
	return nil
}

func (app *Application) initializeServices() error {
	slog.Info("Initializing services...")

	weatherProvider := providers.NewWeatherAPIProvider(&app.config.Weather)
	emailProvider := providers.NewSMTPEmailProvider(&app.config.Email)

	weatherService := service.NewWeatherService(weatherProvider)
	emailService := service.NewEmailService(emailProvider)

	subscriptionRepo := repository.NewSubscriptionRepository(app.db)
	tokenRepo := repository.NewTokenRepository(app.db)

	subscriptionService := service.NewSubscriptionService(
		app.db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		app.config,
	)

	app.server = api.NewServer(app.db, app.config, weatherService, subscriptionService)
	app.scheduler = scheduler.NewScheduler(app.db, app.config, subscriptionService)

	slog.Info("Services initialized successfully")
	return nil
}

// Start starts the application
func (app *Application) Start() error {
	slog.Info("Starting application...")

	slog.Info("Starting scheduler...")
	go app.scheduler.Start()

	slog.Info("Starting HTTP server", "port", app.config.Server.Port)
	return app.server.Start()
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown() error {
	slog.Info("Shutting down application...")

	if app.db != nil {
		if err := database.CloseDB(app.db); err != nil {
			slog.Warn("Error closing database", "error", err)
		}
	}

	slog.Info("Application shutdown complete")
	return nil
}

// Config returns the application configuration
func (app *Application) Config() *config.Config {
	return app.config
}
