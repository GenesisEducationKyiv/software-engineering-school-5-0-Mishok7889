package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
	notificationapi "weatherapi.app/internal/services/notification/adapters/api"
	notificationgrpc "weatherapi.app/internal/services/notification/adapters/grpc"
	messagingadapter "weatherapi.app/internal/services/notification/adapters/messaging"
	"weatherapi.app/internal/services/notification/adapters/messaging/consumers"
)

const (
	logServiceStarting        = "Starting Notification Service"
	logServiceShutdown        = "Shutting down Notification Service..."
	logShutdownComplete       = "Notification service shutdown complete"
	logConsumerManagerStarted = "Consumer manager started successfully"
	logConsumerManagerStopped = "Consumer manager stopped successfully"
	logHTTPServerStarted      = "HTTP server started successfully"
	logHTTPServerStopped      = "HTTP server stopped successfully"

	errInvalidConfig           = "invalid configuration: %w"
	errConfigNil               = "configuration cannot be nil"
	errNotificationHostEmpty   = "notification service host cannot be empty"
	errNotificationPortInvalid = "notification service port must be positive"
	errBrokerURLEmpty          = "message broker URL cannot be empty"
	errInitBroker              = "initialize message broker: %w"
	errInitServices            = "initialize services: %w"
	errInitConsumers           = "initialize consumers: %w"
	errInitHTTPServer          = "initialize HTTP server: %w"
	errStartConsumers          = "start consumers: %w"
	errStartHTTPServer         = "start HTTP server: %w"
)

type NotificationApplication struct {
	config              *config.Config
	subscriptionService ports.SubscriptionService
	weatherService      ports.WeatherService
	emailProvider       ports.EmailProvider
	broker              messaging.MessageBroker
	publisher           messaging.Publisher
	consumerManager     *consumers.ConsumerManager
	httpServer          *notificationapi.HTTPServer
	logger              ports.Logger
	healthChecker       ports.HealthChecker
}

func NewNotificationApplication(cfg *config.Config) (*NotificationApplication, error) {
	if err := validateApplicationConfig(cfg); err != nil {
		return nil, fmt.Errorf(errInvalidConfig, err)
	}

	app := &NotificationApplication{
		config: cfg,
		logger: &infrastructure.SlogLoggerAdapter{},
	}

	if err := app.initializeMessageBroker(); err != nil {
		return nil, fmt.Errorf(errInitBroker, err)
	}

	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf(errInitServices, err)
	}

	if err := app.initializeConsumers(); err != nil {
		return nil, fmt.Errorf(errInitConsumers, err)
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
	if cfg.Services.Notification.Host == "" {
		return errors.New(errNotificationHostEmpty)
	}
	if cfg.Services.Notification.Port <= 0 {
		return errors.New(errNotificationPortInvalid)
	}
	if cfg.MessageBroker.URL == "" {
		return errors.New(errBrokerURLEmpty)
	}
	return nil
}

func (a *NotificationApplication) initializeMessageBroker() error {
	a.logger.Info("Initializing message broker", ports.F("url", a.config.MessageBroker.URL))

	broker, err := messagingadapter.NewNATSBrokerAdapter(messagingadapter.NATSBrokerConfig{
		URL:    a.config.MessageBroker.URL,
		Logger: a.logger,
	})
	if err != nil {
		return fmt.Errorf("create NATS broker adapter: %w", err)
	}

	publisher := messagingadapter.NewEventPublisher(messagingadapter.EventPublisherConfig{
		Broker: broker,
		Logger: a.logger,
	})

	a.broker = broker
	a.publisher = publisher

	a.logger.Info("Message broker initialized successfully")
	return nil
}

func (a *NotificationApplication) initializeServices() error {
	a.logger.Info("Initializing external services")

	subscriptionClient, err := notificationgrpc.NewSubscriptionServiceClient(notificationgrpc.SubscriptionServiceConfig{
		Host: a.config.Services.Subscription.Host,
		Port: a.config.Services.Subscription.Port,
	})
	if err != nil {
		return fmt.Errorf("create subscription service client: %w", err)
	}

	weatherClient, err := notificationgrpc.NewWeatherServiceClient(notificationgrpc.WeatherServiceConfig{
		Host: a.config.Services.Weather.Host,
		Port: a.config.Services.Weather.Port,
	})
	if err != nil {
		return fmt.Errorf("create weather service client: %w", err)
	}

	emailProvider, err := messagingadapter.NewEmailProviderAdapter(messagingadapter.EmailProviderConfig{
		SMTPHost:     a.config.Email.SMTPHost,
		SMTPPort:     a.config.Email.SMTPPort,
		SMTPUsername: a.config.Email.SMTPUsername,
		SMTPPassword: a.config.Email.SMTPPassword,
		FromName:     a.config.Email.FromName,
		FromAddress:  a.config.Email.FromAddress,
	})
	if err != nil {
		return fmt.Errorf("create email provider adapter: %w", err)
	}

	a.subscriptionService = subscriptionClient
	a.weatherService = weatherClient
	a.emailProvider = emailProvider

	a.logger.Info("External services initialized successfully")
	return nil
}

func (a *NotificationApplication) initializeConsumers() error {
	a.logger.Info("Initializing consumers")

	consumerManager := consumers.NewConsumerManager(consumers.ConsumerManagerConfig{
		Broker:              a.broker,
		SubscriptionService: a.subscriptionService,
		WeatherService:      a.weatherService,
		EmailProvider:       a.emailProvider,
		Logger:              a.logger,
	})

	a.consumerManager = consumerManager

	a.logger.Info("Consumers initialized successfully")
	return nil
}

func (a *NotificationApplication) initializeHTTPServer() error {
	a.logger.Info("Initializing HTTP server")

	httpServer := notificationapi.NewHTTPServer(
		notificationapi.ServerConfig{
			Port: a.config.Services.Notification.Port,
		},
		a.publisher,
		a.logger,
	)

	a.httpServer = httpServer

	a.logger.Info("HTTP server initialized successfully")
	return nil
}

func (a *NotificationApplication) Start(ctx context.Context) error {
	a.logger.Info(logServiceStarting, ports.F("port", a.config.Services.Notification.Port))

	if err := a.consumerManager.Start(ctx); err != nil {
		return fmt.Errorf(errStartConsumers, err)
	}
	a.logger.Info(logConsumerManagerStarted)

	if err := a.httpServer.Start(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf(errStartHTTPServer, err)
	}
	a.logger.Info(logHTTPServerStarted)

	return nil
}

func (a *NotificationApplication) Shutdown(ctx context.Context) error {
	a.logger.Info(logServiceShutdown)

	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.logger.Error("Error shutting down HTTP server", ports.F("error", err))
		} else {
			a.logger.Info(logHTTPServerStopped)
		}
	}

	if a.consumerManager != nil {
		if err := a.consumerManager.Stop(ctx); err != nil {
			a.logger.Error("Error stopping consumer manager", ports.F("error", err))
		} else {
			a.logger.Info(logConsumerManagerStopped)
		}
	}

	if a.broker != nil {
		if err := a.broker.Close(); err != nil {
			a.logger.Warn("Error closing message broker", ports.F("error", err))
		}
	}

	a.logger.Info(logShutdownComplete)
	return nil
}

func (a *NotificationApplication) GetPublisher() messaging.Publisher {
	return a.publisher
}

func (a *NotificationApplication) GetHealthChecker() ports.HealthChecker {
	if a.healthChecker == nil {
		a.healthChecker = infrastructure.NewSystemHealthChecker(infrastructure.SystemHealthCheckerConfig{
			ConfigProvider: infrastructure.NewConfigProviderAdapter(a.config),
		})
	}
	return a.healthChecker
}

func (a *NotificationApplication) GetHTTPServer() *notificationapi.HTTPServer {
	return a.httpServer
}

func (a *NotificationApplication) GetConsumerManager() *consumers.ConsumerManager {
	return a.consumerManager
}
