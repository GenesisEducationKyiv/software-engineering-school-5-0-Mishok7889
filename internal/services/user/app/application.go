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
	"weatherapi.app/internal/core/token"
	"weatherapi.app/internal/ports"
	usergrpc "weatherapi.app/internal/services/user/adapters/grpc"
)

const (
	logInitDatabase       = "Initializing database connection..."
	logDatabaseSuccess    = "Database connection established successfully"
	logInitTokenUC        = "Initializing token use case..."
	logTokenUCSuccess     = "Token use case initialized successfully"
	logInitGRPCHandler    = "Initializing gRPC handler..."
	logGRPCSuccess        = "gRPC handler initialized successfully"
	logShuttingDown       = "Shutting down user application..."
	logShutdownComplete   = "User application shutdown complete"
	logClosingDB          = "Error closing database"
	logRunningMigrations  = "Running database migrations for user service..."
	logMigrationsComplete = "Database migrations completed successfully"

	errInitDatabase    = "initialize database: %w"
	errInitTokenUC     = "initialize token use case: %w"
	errInitGRPCHandler = "initialize gRPC handler: %w"
	errConnectDB       = "connect to database: %w"
	errCreateUseCase   = "create token use case: %w"
	errRunMigrations   = "run migrations: %w"
	errAutoMigrate     = "auto migrate: %w"

	errConfigNil         = "configuration cannot be nil"
	errDatabaseHostEmpty = "database host cannot be empty"
	errDatabaseNameEmpty = "database name cannot be empty"
	errInvalidConfig     = "invalid configuration: %w"
)

type UserApplication struct {
	config      *config.Config
	tokenUC     *token.UseCase
	grpcHandler *usergrpc.AuthServiceServer
	db          *gorm.DB
}

func NewUserApplication(cfg *config.Config) (*UserApplication, error) {
	if err := validateApplicationConfig(cfg); err != nil {
		return nil, fmt.Errorf(errInvalidConfig, err)
	}

	app := &UserApplication{
		config: cfg,
	}

	if err := app.initializeDatabase(); err != nil {
		return nil, fmt.Errorf(errInitDatabase, err)
	}

	if err := app.initializeTokenUseCase(); err != nil {
		return nil, fmt.Errorf(errInitTokenUC, err)
	}

	if err := app.initializeGRPCHandler(); err != nil {
		return nil, fmt.Errorf(errInitGRPCHandler, err)
	}

	return app, nil
}

func validateApplicationConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New(errConfigNil)
	}
	if cfg.UserDB.Host == "" {
		return errors.New(errDatabaseHostEmpty)
	}
	if cfg.UserDB.Name == "" {
		return errors.New(errDatabaseNameEmpty)
	}
	return nil
}

func (a *UserApplication) initializeDatabase() error {
	slog.Info(logInitDatabase)

	dsn := a.config.UserDB.GetDSN()
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

func (a *UserApplication) runMigrations(db *gorm.DB) error {
	slog.Info(logRunningMigrations)

	if err := db.AutoMigrate(
		&database.TokenModel{},
	); err != nil {
		return fmt.Errorf(errAutoMigrate, err)
	}

	slog.Info(logMigrationsComplete)
	return nil
}

func (a *UserApplication) initializeTokenUseCase() error {
	slog.Info(logInitTokenUC)

	var logger ports.Logger = &infrastructure.SlogLoggerAdapter{}

	tokenRepo := database.NewTokenRepositoryAdapter(a.db)
	tokenGenerator := infrastructure.NewUUIDTokenGenerator()

	tokenUC, err := token.NewUseCase(token.UseCaseDependencies{
		TokenRepo:      tokenRepo,
		TokenGenerator: tokenGenerator,
		Logger:         logger,
	})
	if err != nil {
		return fmt.Errorf(errCreateUseCase, err)
	}

	a.tokenUC = tokenUC
	slog.Info(logTokenUCSuccess)
	return nil
}

func (a *UserApplication) initializeGRPCHandler() error {
	slog.Info(logInitGRPCHandler)

	grpcHandler := usergrpc.NewAuthServiceServer(a.tokenUC)
	a.grpcHandler = grpcHandler

	slog.Info(logGRPCSuccess)
	return nil
}

func (a *UserApplication) GetTokenUseCase() *token.UseCase {
	return a.tokenUC
}

func (a *UserApplication) GetGRPCHandler() *usergrpc.AuthServiceServer {
	return a.grpcHandler
}

func (a *UserApplication) Shutdown(ctx context.Context) error {
	slog.Info(logShuttingDown)

	if a.db != nil {
		if db, err := a.db.DB(); err == nil {
			if err := db.Close(); err != nil {
				slog.Warn(logClosingDB, "error", err)
			}
		}
	}

	slog.Info(logShutdownComplete)
	return nil
}
