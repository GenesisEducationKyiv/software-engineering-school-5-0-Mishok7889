package app

import (
	"fmt"
	"log/slog"
	"time"

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
		return fmt.Errorf("load application configuration: %w", err)
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
		return fmt.Errorf("initialize database connection: %w", err)
	}

	if err := database.RunMigrations(db); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		return fmt.Errorf("run database migrations: %w", err)
	}

	app.db = db
	slog.Info("Database initialized successfully")
	return nil
}

func (app *Application) initializeServices() error {
	slog.Info("Initializing services...")

	// Create provider manager with all patterns
	providerManager, err := app.createProviderManager()
	if err != nil {
		return fmt.Errorf("create provider manager: %w", err)
	}

	// Create email provider
	emailProvider := providers.NewSMTPEmailProvider(&app.config.Email)

	// Create services
	weatherService := service.NewWeatherService(providerManager)
	emailService := service.NewEmailService(emailProvider)

	// Create repositories
	subscriptionRepo := repository.NewSubscriptionRepository(app.db)
	tokenRepo := repository.NewTokenRepository(app.db)

	// Create subscription service
	subscriptionService := service.NewSubscriptionService(
		app.db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		app.config,
	)

	// Create server and scheduler
	server, err := api.NewServer(
		api.NewServerOptionsBuilder().
			WithDB(app.db).
			WithConfig(app.config).
			WithWeatherService(weatherService).
			WithSubscriptionService(subscriptionService).
			WithProviderManager(providerManager).
			WithProviderMetrics(providerManager).
			Build(),
	)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	app.server = server
	app.scheduler = scheduler.NewScheduler(app.db, app.config, subscriptionService)

	slog.Info("Services initialized successfully")
	return nil
}

// createProviderManager creates and configures the weather provider manager
// Follows Factory Method pattern: creates complex configured object
func (app *Application) createProviderManager() (*providers.ProviderManager, error) {
	slog.Debug("Creating weather provider manager...")

	// Convert config to provider configuration
	providerConfig := &providers.ProviderConfiguration{
		WeatherAPIKey:     app.config.Weather.APIKey,
		WeatherAPIBaseURL: app.config.Weather.BaseURL,
		OpenWeatherMapKey: app.config.Weather.OpenWeatherMapKey,
		AccuWeatherKey:    app.config.Weather.AccuWeatherKey,
		CacheTTL:          time.Duration(app.config.Weather.CacheTTLMinutes) * time.Minute,
		LogFilePath:       app.config.Weather.LogFilePath,
		EnableCache:       app.config.Weather.EnableCache,
		EnableLogging:     app.config.Weather.EnableLogging,
		ProviderOrder:     app.config.Weather.ProviderOrder,
		CacheType:         providers.CacheTypeFromString(app.config.Cache.Type),
		CacheConfig:       &app.config.Cache,
	}

	// Create provider manager
	providerManager, err := providers.NewProviderManager(providerConfig)
	if err != nil {
		return nil, err
	}

	slog.Debug("Provider manager created", "config", providerManager.GetProviderInfo())
	return providerManager, nil
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
