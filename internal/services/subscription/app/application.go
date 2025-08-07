package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"weatherapi.app/internal/adapters/database"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/ports"
	subscriptionapi "weatherapi.app/internal/services/subscription/adapters/api"
	subscriptiongrpc "weatherapi.app/internal/services/subscription/adapters/grpc"
	"weatherapi.app/pkg/logger"
)

const (
	logInitDatabase              = "Initializing database connection..."
	logDatabaseSuccess           = "Database connection established successfully"
	logInitSubscriptionUC        = "Initializing subscription use case..."
	logSubscriptionUCSuccess     = "Subscription use case initialized successfully"
	logInitHTTPServer            = "Initializing HTTP server..."
	logHTTPServerSuccess         = "HTTP server initialized successfully"
	logShuttingDown              = "Shutting down subscription application..."
	logShutdownComplete          = "Subscription application shutdown complete"
	logClosingDB                 = "Error closing database"
	logRunningMigrations         = "Running database migrations for subscription service..."
	logMigrationsComplete        = "Database migrations completed successfully"
	logInitUserClient            = "Initializing User Service gRPC client..."
	logUserClientSuccess         = "User Service gRPC client initialized successfully"
	logInitNotificationClient    = "Initializing Notification Service client..."
	logNotificationClientSuccess = "Notification Service client initialized successfully"

	errInitDatabase           = "initialize database: %w"
	errInitSubscriptionUC     = "initialize subscription use case: %w"
	errInitHTTPServer         = "initialize HTTP server: %w"
	errInitUserClient         = "initialize user service client: %w"
	errInitNotificationClient = "initialize notification service client: %w"
	errConnectDB              = "connect to database: %w"
	errCreateUseCase          = "create subscription use case: %w"
	errRunMigrations          = "run migrations: %w"
	errAutoMigrate            = "auto migrate: %w"

	errConfigNil         = "configuration cannot be nil"
	errDatabaseHostEmpty = "database host cannot be empty"
	errDatabaseNameEmpty = "database name cannot be empty"
	errInvalidConfig     = "invalid configuration: %w"
)

type SubscriptionApplication struct {
	config             *config.Config
	subscriptionUC     *subscription.UseCase
	httpServer         *subscriptionapi.HTTPServer
	userClient         *subscriptiongrpc.UserServiceClient
	notificationClient *subscriptiongrpc.NotificationServiceClient
	db                 *gorm.DB
	logger             ports.Logger
}

func NewSubscriptionApplication(cfg *config.Config) (*SubscriptionApplication, error) {
	return NewSubscriptionApplicationWithLogger(cfg, nil)
}

// NewSubscriptionApplicationWithLogger creates a subscription application with logger
func NewSubscriptionApplicationWithLogger(cfg *config.Config, log *logger.Logger) (*SubscriptionApplication, error) {
	if err := validateApplicationConfig(cfg); err != nil {
		return nil, fmt.Errorf(errInvalidConfig, err)
	}

	app := &SubscriptionApplication{
		config: cfg,
		logger: &infrastructure.SlogLoggerAdapter{},
	}

	// Use provided logger if available
	if log != nil {
		app.logger = &infrastructure.LoggerAdapter{Logger: log}
	}

	if err := app.initializeDatabase(); err != nil {
		return nil, fmt.Errorf(errInitDatabase, err)
	}

	if err := app.initializeUserClient(); err != nil {
		return nil, fmt.Errorf(errInitUserClient, err)
	}

	if err := app.initializeNotificationClient(); err != nil {
		return nil, fmt.Errorf(errInitNotificationClient, err)
	}

	if err := app.initializeSubscriptionUseCase(); err != nil {
		return nil, fmt.Errorf(errInitSubscriptionUC, err)
	}

	if err := app.initializeHTTPServer(); err != nil {
		return nil, fmt.Errorf(errInitHTTPServer, err)
	}

	return app, nil
}

func validateApplicationConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New(errConfigNil)
	}
	if cfg.SubscriptionDB.Host == "" {
		return errors.New(errDatabaseHostEmpty)
	}
	if cfg.SubscriptionDB.Name == "" {
		return errors.New(errDatabaseNameEmpty)
	}
	return nil
}

func (a *SubscriptionApplication) initializeDatabase() error {
	slog.Info(logInitDatabase)

	dsn := a.config.SubscriptionDB.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf(errConnectDB, err)
	}

	if err := a.runMigrations(db); err != nil {
		return fmt.Errorf(errRunMigrations, err)
	}

	a.db = db
	slog.Info(logDatabaseSuccess)
	return nil
}

func (a *SubscriptionApplication) runMigrations(db *gorm.DB) error {
	slog.Info(logRunningMigrations)

	if err := db.AutoMigrate(
		&database.SubscriptionModel{},
		&database.TokenModel{},
	); err != nil {
		return fmt.Errorf(errAutoMigrate, err)
	}

	slog.Info(logMigrationsComplete)
	return nil
}

func (a *SubscriptionApplication) initializeUserClient() error {
	slog.Info(logInitUserClient)

	userClient, err := subscriptiongrpc.NewUserServiceClient(subscriptiongrpc.UserServiceConfig{
		Host: a.config.Services.User.GetHost(),
		Port: a.config.Services.User.GetPort(),
	})
	if err != nil {
		return fmt.Errorf("create user service client: %w", err)
	}

	a.userClient = userClient
	slog.Info(logUserClientSuccess)
	return nil
}

func (a *SubscriptionApplication) initializeNotificationClient() error {
	slog.Info(logInitNotificationClient)

	notificationClient, err := subscriptiongrpc.NewNotificationServiceClient(subscriptiongrpc.NotificationServiceConfig{
		Host: a.config.Services.Notification.GetHost(),
		Port: a.config.Services.Notification.GetPort(),
	})
	if err != nil {
		return fmt.Errorf("create notification service client: %w", err)
	}

	a.notificationClient = notificationClient
	slog.Info(logNotificationClientSuccess)
	return nil
}

func (a *SubscriptionApplication) initializeSubscriptionUseCase() error {
	slog.Info(logInitSubscriptionUC)

	var logger ports.Logger = &infrastructure.SlogLoggerAdapter{}

	subscriptionRepo := database.NewSubscriptionRepositoryAdapter(a.db)
	tokenRepo := database.NewTokenRepositoryAdapter(a.db)
	tokenGenerator := infrastructure.NewUUIDTokenGenerator()
	configProvider := infrastructure.NewConfigProviderAdapter(a.config)

	subscriptionUC, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo:    subscriptionRepo,
		TokenRepo:           tokenRepo,
		TokenGenerator:      tokenGenerator,
		NotificationService: a.notificationClient,
		Config:              configProvider,
		Logger:              logger,
	})
	if err != nil {
		return fmt.Errorf(errCreateUseCase, err)
	}

	a.subscriptionUC = subscriptionUC
	slog.Info(logSubscriptionUCSuccess)
	return nil
}

func (a *SubscriptionApplication) initializeHTTPServer() error {
	slog.Info(logInitHTTPServer)

	var logger ports.Logger = &infrastructure.SlogLoggerAdapter{}

	httpServer := subscriptionapi.NewHTTPServer(
		subscriptionapi.ServerConfig{
			Port: a.config.Services.Subscription.GetPort(),
		},
		a.subscriptionUC,
		logger,
	)

	a.httpServer = httpServer
	slog.Info(logHTTPServerSuccess)
	return nil
}

func (a *SubscriptionApplication) Start(ctx context.Context) error {
	return a.httpServer.Start()
}

func (a *SubscriptionApplication) Shutdown(ctx context.Context) error {
	slog.Info(logShuttingDown)

	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down HTTP server", "error", err)
		}
	}

	if a.userClient != nil {
		if err := a.userClient.Close(); err != nil {
			slog.Warn("Error closing user service client", "error", err)
		}
	}

	if a.db == nil {
		slog.Info(logShutdownComplete)
		return nil
	}

	db, err := a.db.DB()
	if err != nil {
		slog.Info(logShutdownComplete)
		return nil
	}

	if err := db.Close(); err != nil {
		slog.Warn(logClosingDB, "error", err)
	}

	slog.Info(logShutdownComplete)
	return nil
}

func (a *SubscriptionApplication) GetHTTPServer() *subscriptionapi.HTTPServer {
	return a.httpServer
}

func (a *SubscriptionApplication) GetSubscriptionUseCase() *subscription.UseCase {
	return a.subscriptionUC
}
