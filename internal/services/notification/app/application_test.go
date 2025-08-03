package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/shared"
	mockPorts "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports/messaging"
	notificationapi "weatherapi.app/internal/services/notification/adapters/api"
	"weatherapi.app/internal/services/notification/adapters/messaging/consumers"
)

func setupMocks(t *testing.T) (*mockPorts.MessageBroker, *mockPorts.SubscriptionService, *mockPorts.WeatherService, *mockPorts.EmailProvider, *mockPorts.Logger) {
	mockBroker := mockPorts.NewMessageBroker(t)
	mockSubService := mockPorts.NewSubscriptionService(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()

	return mockBroker, mockSubService, mockWeatherService, mockEmailProvider, mockLogger
}

func TestNotificationApplication_ValidateConfig_Success(t *testing.T) {
	cfg := &config.Config{
		Services: config.ServicesConfig{
			Notification: config.NotificationServiceConfig{
				Host: "localhost",
				Port: 8084,
			},
		},
		MessageBroker: config.MessageBrokerConfig{
			URL: "nats://localhost:4222",
		},
	}

	err := validateApplicationConfig(cfg)
	assert.NoError(t, err)
}

func TestNotificationApplication_ValidateConfig_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "empty notification service host",
			config: &config.Config{
				Services: config.ServicesConfig{
					Notification: config.NotificationServiceConfig{
						Host: "",
						Port: 8084,
					},
				},
				MessageBroker: config.MessageBrokerConfig{
					URL: "nats://localhost:4222",
				},
			},
		},
		{
			name: "invalid notification service port",
			config: &config.Config{
				Services: config.ServicesConfig{
					Notification: config.NotificationServiceConfig{
						Host: "localhost",
						Port: 0,
					},
				},
				MessageBroker: config.MessageBrokerConfig{
					URL: "nats://localhost:4222",
				},
			},
		},
		{
			name: "empty message broker URL",
			config: &config.Config{
				Services: config.ServicesConfig{
					Notification: config.NotificationServiceConfig{
						Host: "localhost",
						Port: 8084,
					},
				},
				MessageBroker: config.MessageBrokerConfig{
					URL: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateApplicationConfig(tt.config)
			assert.Error(t, err)
		})
	}
}

func TestNotificationApplication_ConsumerManagerLifecycle(t *testing.T) {
	mockBroker, mockSubService, mockWeatherService, mockEmailProvider, mockLogger := setupMocks(t)

	// Mock successful subscription setup for ConsumerManager
	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeWeatherDataFetched),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeSubscriptionCreated),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeSubscriptionCancelled),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.CommandTypeSendWeatherUpdates),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	// Create ConsumerManager and test its lifecycle
	consumerManager := consumers.NewConsumerManager(consumers.ConsumerManagerConfig{
		Broker:              mockBroker,
		SubscriptionService: mockSubService,
		WeatherService:      mockWeatherService,
		EmailProvider:       mockEmailProvider,
		Logger:              mockLogger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test start
	err := consumerManager.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, consumerManager.IsRunning())

	// Test stop
	err = consumerManager.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, consumerManager.IsRunning())
}

func TestNotificationApplication_ConsumerManagerStartError(t *testing.T) {
	mockBroker, mockSubService, mockWeatherService, mockEmailProvider, mockLogger := setupMocks(t)

	// Mock failed subscription setup
	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeWeatherDataFetched),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(assert.AnError)

	// Create ConsumerManager
	consumerManager := consumers.NewConsumerManager(consumers.ConsumerManagerConfig{
		Broker:              mockBroker,
		SubscriptionService: mockSubService,
		WeatherService:      mockWeatherService,
		EmailProvider:       mockEmailProvider,
		Logger:              mockLogger,
	})

	ctx := context.Background()

	// Test start failure
	err := consumerManager.Start(ctx)
	assert.Error(t, err)
	assert.False(t, consumerManager.IsRunning())
}

func TestNotificationApplication_HTTPServerLifecycle(t *testing.T) {
	_, _, _, _, mockLogger := setupMocks(t)

	// Create HTTP server with mock publisher
	mockPublisher := mockPorts.NewPublisher(t)
	mockPublisher.EXPECT().PublishCommand(mock.Anything, mock.Anything).Return(nil).Maybe()
	mockPublisher.EXPECT().PublishEvent(mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create mock email provider and builder
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockEmailBuilder := mockPorts.NewEmailBuilder(t)

	httpServer := notificationapi.NewHTTPServer(
		notificationapi.ServerConfig{Port: 0}, // Port 0 will assign a random available port
		mockPublisher,
		mockEmailProvider,
		mockEmailBuilder,
		mockLogger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in a goroutine since it's blocking
	startErr := make(chan error, 1)
	go func() {
		startErr <- httpServer.Start()
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	err := httpServer.Shutdown(ctx)
	assert.NoError(t, err)

	// Check if start completed (or with expected server closed error)
	select {
	case err := <-startErr:
		// http.ErrServerClosed is expected when we shut down the server
		if err != nil && err.Error() != "http: Server closed" {
			t.Logf("Start error (this may be expected): %v", err)
		}
	case <-time.After(1 * time.Second):
		// Timeout is fine, the server was probably still starting
	}
}

func TestNotificationApplication_EventProcessing_Integration(t *testing.T) {
	mockBroker, mockSubService, mockWeatherService, mockEmailProvider, mockLogger := setupMocks(t)

	var weatherEventHandler messaging.MessageHandler
	var subscriptionEventHandler messaging.MessageHandler

	// Capture the handlers during subscription setup
	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeWeatherDataFetched),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).RunAndReturn(func(ctx context.Context, topic, group string, handler messaging.MessageHandler) error {
		weatherEventHandler = handler
		return nil
	})

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeSubscriptionCreated),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).RunAndReturn(func(ctx context.Context, topic, group string, handler messaging.MessageHandler) error {
		subscriptionEventHandler = handler
		return nil
	})

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeSubscriptionCancelled),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.CommandTypeSendWeatherUpdates),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(nil)

	// Create ConsumerManager
	consumerManager := consumers.NewConsumerManager(consumers.ConsumerManagerConfig{
		Broker:              mockBroker,
		SubscriptionService: mockSubService,
		WeatherService:      mockWeatherService,
		EmailProvider:       mockEmailProvider,
		Logger:              mockLogger,
	})

	ctx := context.Background()

	// Start consumers
	err := consumerManager.Start(ctx)
	require.NoError(t, err)

	// Test weather event processing
	mockSubService.EXPECT().GetActiveSubscriptionsByCity(mock.Anything, "London").
		Return([]shared.Subscription{
			{
				ID:        "sub-1",
				UserID:    "user-1",
				City:      "London",
				Email:     "test@example.com",
				Frequency: shared.FrequencyHourly,
				IsActive:  true,
			},
		}, nil)

	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.AnythingOfType("shared.EmailRequest")).Return(nil)

	weatherEvent := shared.NewWeatherDataFetchedEvent("London", 22.5, 65.0, "Sunny")
	weatherMessage := &messaging.Message{
		ID:      weatherEvent.ID(),
		Topic:   weatherEvent.Topic(),
		Data:    weatherEvent.Data(),
		Headers: weatherEvent.Headers(),
	}

	err = weatherEventHandler(ctx, weatherMessage)
	assert.NoError(t, err)

	// Test subscription created event processing
	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.AnythingOfType("shared.EmailRequest")).Return(nil)

	subscriptionEvent := shared.NewSubscriptionCreatedEvent("sub-2", "user-2", "Paris", "daily", "user2@example.com")
	subscriptionMessage := &messaging.Message{
		ID:      subscriptionEvent.ID(),
		Topic:   subscriptionEvent.Topic(),
		Data:    subscriptionEvent.Data(),
		Headers: subscriptionEvent.Headers(),
	}

	err = subscriptionEventHandler(ctx, subscriptionMessage)
	assert.NoError(t, err)

	// Stop consumers
	err = consumerManager.Stop(ctx)
	assert.NoError(t, err)
}
