package app

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"weatherapi.app/internal/adapters/database"
	"weatherapi.app/internal/adapters/external"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

type DependencyContainer struct {
	config DependencyConfig
	db     *gorm.DB
	ports  *ports.ApplicationPorts
}

type DependencyConfig struct {
	Database config.DatabaseConfig
	Weather  config.WeatherConfig
	Email    config.EmailConfig
	Cache    config.CacheConfig
}

func NewDependencyContainer(depConfig DependencyConfig, appConfig *config.Config) (*DependencyContainer, error) {
	container := &DependencyContainer{
		config: depConfig,
	}

	if err := container.initializeDatabase(); err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	if err := container.initializePorts(appConfig); err != nil {
		return nil, fmt.Errorf("initialize ports: %w", err)
	}

	return container, nil
}

func (c *DependencyContainer) initializeDatabase() error {
	slog.Info("Initializing database connection...")

	dsn := c.config.Database.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	// Run migrations
	if err := c.runMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	c.db = db
	slog.Info("Database connection established successfully")
	return nil
}

func (c *DependencyContainer) runMigrations(db *gorm.DB) error {
	slog.Info("Running database migrations...")

	if err := db.AutoMigrate(
		&database.SubscriptionModel{},
		&database.TokenModel{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func (c *DependencyContainer) initializePorts(appConfig *config.Config) error {
	slog.Info("Initializing ports...")

	// Database repositories
	subscriptionRepo := database.NewSubscriptionRepositoryAdapter(c.db)
	tokenRepo := database.NewTokenRepositoryAdapter(c.db)

	// Initialize logger based on configuration
	var logger ports.Logger = &infrastructure.SlogLoggerAdapter{}

	// If file logging is enabled, create a file logger
	if c.config.Weather.EnableLogging && c.config.Weather.LogFilePath != "" {
		fileLogger, err := infrastructure.NewFileLoggerAdapter(c.config.Weather.LogFilePath)
		if err != nil {
			slog.Warn("Failed to create file logger, falling back to slog", "error", err)
		} else {
			logger = fileLogger
			slog.Info("File logging enabled", "path", c.config.Weather.LogFilePath)
		}
	}

	// Weather provider manager with Chain of Responsibility
	providerManager := external.NewWeatherProviderManagerAdapter(external.ProviderManagerConfig{
		WeatherAPIKey:     c.config.Weather.APIKey,
		WeatherAPIBaseURL: c.config.Weather.BaseURL,
		OpenWeatherKey:    c.config.Weather.OpenWeatherMapKey,
		OpenWeatherURL:    c.config.Weather.OpenWeatherMapBaseURL,
		AccuWeatherKey:    c.config.Weather.AccuWeatherKey,
		AccuWeatherURL:    c.config.Weather.AccuWeatherBaseURL,
		ProviderOrder:     c.config.Weather.ProviderOrder,
		Logger:            logger,
	})

	// If logging is enabled, wrap the provider manager with logging decorator
	if c.config.Weather.EnableLogging {
		providerManager = external.NewWeatherProviderManagerLoggingDecorator(providerManager, logger)
		slog.Info("Weather provider logging enabled")
	}

	emailProvider := external.NewSMTPEmailProviderAdapter(external.EmailProviderConfig{
		Host:     c.config.Email.SMTPHost,
		Port:     c.config.Email.SMTPPort,
		Username: c.config.Email.SMTPUsername,
		Password: c.config.Email.SMTPPassword,
		FromName: c.config.Email.FromName,
		FromAddr: c.config.Email.FromAddress,
	})

	cacheFactory := external.NewCacheProviderFactory()
	genericCacheProvider, err := cacheFactory.CreateCacheProvider(&c.config.Cache)
	if err != nil {
		slog.Error("Failed to create cache provider", "error", err)
		return fmt.Errorf("create cache provider: %w", err)
	}

	weatherCacheProvider := external.NewWeatherCacheAdapter(genericCacheProvider)

	// Log cache type being used
	slog.Info("Cache provider initialized",
		"type", c.config.Cache.Type.String(),
		"redis_addr", c.config.Cache.Redis.Addr)

	// Infrastructure (using actual config)
	configProvider := infrastructure.NewConfigProviderAdapter(appConfig)

	// Create provider manager
	// (already created above as providerManager)

	// Weather metrics
	weatherMetrics := external.NewWeatherMetricsAdapter(weatherCacheProvider, providerManager)

	c.ports = &ports.ApplicationPorts{
		// Weather
		WeatherProvider: providerManager,
		WeatherCache:    weatherCacheProvider,
		WeatherMetrics:  weatherMetrics,

		// Subscription
		SubscriptionRepository: subscriptionRepo,
		TokenRepository:        tokenRepo,

		// Communication
		EmailProvider: emailProvider,

		// Cache
		CacheMetrics: genericCacheProvider.(ports.CacheMetrics),

		// Infrastructure
		ConfigProvider: configProvider,
		Logger:         logger,
		Database:       c.db,
	}

	slog.Info("Ports initialized successfully")
	return nil
}

func (c *DependencyContainer) ApplicationPorts() *ports.ApplicationPorts {
	return c.ports
}

func (c *DependencyContainer) Database() *gorm.DB {
	return c.db
}

// Database model structs for GORM migrations
type DatabaseModels struct {
	Subscription *database.SubscriptionModel
	Token        *database.TokenModel
}

func (c *DependencyContainer) GetDatabaseModels() DatabaseModels {
	return DatabaseModels{
		Subscription: &database.SubscriptionModel{},
		Token:        &database.TokenModel{},
	}
}

// Helper functions for creating test dependencies
func NewTestDependencyContainer() (*DependencyContainer, error) {
	depConfig := DependencyConfig{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			Name:     "test_weatherapi",
			SSLMode:  "disable",
		},
		Weather: config.WeatherConfig{
			APIKey:          "test-key",
			BaseURL:         "https://api.weatherapi.com/v1",
			CacheTTLMinutes: 10,
		},
		Email: config.EmailConfig{
			SMTPHost:     "smtp.gmail.com",
			SMTPPort:     587,
			SMTPUsername: "test@example.com",
			SMTPPassword: "test-password",
			FromName:     "Test Weather API",
			FromAddress:  "test@weatherapi.app",
		},
		Cache: config.CacheConfig{
			Type: config.CacheTypeMemory,
		},
	}

	testConfig := &config.Config{
		Server: config.ServerConfig{Port: 8080},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			Name:     "test_weatherapi",
			SSLMode:  "disable",
		},
		AppBaseURL: "http://localhost:8080",
	}

	return NewDependencyContainer(depConfig, testConfig)
}

// Cleanup function for graceful shutdown
func (c *DependencyContainer) Cleanup() error {
	if c.db != nil {
		if db, err := c.db.DB(); err == nil {
			return db.Close()
		}
	}
	return nil
}
