package consumers

import (
	"context"
	"fmt"
	"sync"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type ConsumerManager struct {
	broker                    messaging.MessageBroker
	logger                    ports.Logger
	weatherEventConsumer      *WeatherEventConsumer
	subscriptionEventConsumer *SubscriptionEventConsumer
	commandConsumer           *CommandConsumer
	running                   bool
	mu                        sync.RWMutex
	cancelFuncs               []context.CancelFunc
}

type ConsumerManagerConfig struct {
	Broker              messaging.MessageBroker
	SubscriptionService ports.SubscriptionService
	WeatherService      ports.WeatherService
	EmailProvider       ports.EmailProvider
	Logger              ports.Logger
}

func NewConsumerManager(config ConsumerManagerConfig) *ConsumerManager {
	weatherEventConsumer := NewWeatherEventConsumer(WeatherEventConsumerConfig{
		SubscriptionService: config.SubscriptionService,
		EmailProvider:       config.EmailProvider,
		Logger:              config.Logger,
	})

	subscriptionEventConsumer := NewSubscriptionEventConsumer(SubscriptionEventConsumerConfig{
		EmailProvider: config.EmailProvider,
		Logger:        config.Logger,
	})

	commandConsumer := NewCommandConsumer(CommandConsumerConfig{
		SubscriptionService: config.SubscriptionService,
		WeatherService:      config.WeatherService,
		EmailProvider:       config.EmailProvider,
		Logger:              config.Logger,
	})

	return &ConsumerManager{
		broker:                    config.Broker,
		logger:                    config.Logger,
		weatherEventConsumer:      weatherEventConsumer,
		subscriptionEventConsumer: subscriptionEventConsumer,
		commandConsumer:           commandConsumer,
	}
}

func (cm *ConsumerManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return fmt.Errorf("consumer manager is already running")
	}

	cm.logger.Info("Starting consumer manager")

	if err := cm.setupWeatherEventConsumers(ctx); err != nil {
		return fmt.Errorf("setup weather event consumers: %w", err)
	}

	if err := cm.setupSubscriptionEventConsumers(ctx); err != nil {
		return fmt.Errorf("setup subscription event consumers: %w", err)
	}

	if err := cm.setupCommandConsumers(ctx); err != nil {
		return fmt.Errorf("setup command consumers: %w", err)
	}

	cm.running = true
	cm.logger.Info("Consumer manager started successfully")

	return nil
}

func (cm *ConsumerManager) Stop(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return nil
	}

	cm.logger.Info("Stopping consumer manager")

	for _, cancel := range cm.cancelFuncs {
		cancel()
	}

	cm.running = false
	cm.cancelFuncs = nil
	cm.logger.Info("Consumer manager stopped successfully")

	return nil
}

func (cm *ConsumerManager) IsRunning() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.running
}

func (cm *ConsumerManager) setupWeatherEventConsumers(ctx context.Context) error {
	return cm.broker.SubscribeWithGroup(
		ctx,
		string(shared.EventTypeWeatherDataFetched),
		"notification-service",
		func(ctx context.Context, message *messaging.Message) error {
			return cm.weatherEventConsumer.HandleMessage(ctx, message)
		},
	)
}

func (cm *ConsumerManager) setupSubscriptionEventConsumers(ctx context.Context) error {
	if err := cm.broker.SubscribeWithGroup(
		ctx,
		string(shared.EventTypeSubscriptionCreated),
		"notification-service",
		func(ctx context.Context, message *messaging.Message) error {
			return cm.subscriptionEventConsumer.HandleMessage(ctx, message)
		},
	); err != nil {
		return err
	}

	return cm.broker.SubscribeWithGroup(
		ctx,
		string(shared.EventTypeSubscriptionCancelled),
		"notification-service",
		func(ctx context.Context, message *messaging.Message) error {
			return cm.subscriptionEventConsumer.HandleMessage(ctx, message)
		},
	)
}

func (cm *ConsumerManager) setupCommandConsumers(ctx context.Context) error {
	return cm.broker.SubscribeWithGroup(
		ctx,
		string(shared.CommandTypeSendWeatherUpdates),
		"notification-service",
		func(ctx context.Context, message *messaging.Message) error {
			return cm.commandConsumer.HandleMessage(ctx, message)
		},
	)
}
