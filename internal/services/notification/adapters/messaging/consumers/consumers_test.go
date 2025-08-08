package consumers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/core/shared"
	mockPorts "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports/messaging"
)

func setupLoggerMock(t *testing.T) *mockPorts.Logger {
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
	return mockLogger
}

func TestWeatherEventConsumer_ProcessWeatherDataFetchedEvent_Success(t *testing.T) {
	tests := []struct {
		name          string
		event         *shared.WeatherDataFetchedEvent
		subscriptions []shared.Subscription
		setupMocks    func(*mockPorts.SubscriptionService, *mockPorts.EmailProvider, *mockPorts.Logger)
		expectedError bool
	}{{
		name:  "successful processing with active subscriptions",
		event: shared.NewWeatherDataFetchedEvent("London", 22.5, 65.0, "Sunny"),
		subscriptions: []shared.Subscription{
			{
				ID:        "sub-1",
				UserID:    "user-1",
				City:      "London",
				Email:     "test@example.com",
				Frequency: shared.FrequencyHourly,
				IsActive:  true,
			},
		},
		setupMocks: func(subService *mockPorts.SubscriptionService, emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
			subService.EXPECT().GetActiveSubscriptionsByCity(mock.Anything, "London").
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

			emailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(req shared.EmailRequest) bool {
				return req.To == "test@example.com" &&
					req.Subject != "" &&
					req.Body != ""
			})).Return(nil)
		},
		expectedError: false,
	},
		{
			name:          "no active subscriptions for city",
			event:         shared.NewWeatherDataFetchedEvent("Paris", 18.0, 70.0, "Cloudy"),
			subscriptions: []shared.Subscription{},
			setupMocks: func(subService *mockPorts.SubscriptionService, emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
				subService.EXPECT().GetActiveSubscriptionsByCity(mock.Anything, "Paris").
					Return([]shared.Subscription{}, nil)
			},
			expectedError: false,
		},
		{
			name:  "subscription service error",
			event: shared.NewWeatherDataFetchedEvent("Berlin", 15.0, 80.0, "Rainy"),
			setupMocks: func(subService *mockPorts.SubscriptionService, emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
				subService.EXPECT().GetActiveSubscriptionsByCity(mock.Anything, "Berlin").
					Return(nil, assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSubService := mockPorts.NewSubscriptionService(t)
			mockEmailProvider := mockPorts.NewEmailProvider(t)
			mockLogger := setupLoggerMock(t)

			tt.setupMocks(mockSubService, mockEmailProvider, mockLogger)

			consumer := NewWeatherEventConsumer(WeatherEventConsumerConfig{
				SubscriptionService: mockSubService,
				EmailProvider:       mockEmailProvider,
				Logger:              mockLogger,
			})

			messageData, err := json.Marshal(tt.event)
			require.NoError(t, err)

			message := &messaging.Message{
				ID:      tt.event.ID(),
				Topic:   tt.event.Topic(),
				Data:    messageData,
				Headers: tt.event.Headers(),
			}

			ctx := context.Background()
			err = consumer.HandleMessage(ctx, message)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSubscriptionEventConsumer_ProcessSubscriptionCreatedEvent_Success(t *testing.T) {
	tests := []struct {
		name          string
		event         *shared.SubscriptionCreatedEvent
		setupMocks    func(*mockPorts.EmailProvider, *mockPorts.Logger)
		expectedError bool
	}{
		{
			name:  "successful welcome email sending",
			event: shared.NewSubscriptionCreatedEvent("sub-1", "user-1", "London", "daily", "test@example.com"),
			setupMocks: func(emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
				emailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(req shared.EmailRequest) bool {
					return req.To == "test@example.com" &&
						req.Subject != "" &&
						req.Body != ""
				})).Return(nil)
			},
			expectedError: false,
		},
		{
			name:  "email provider error",
			event: shared.NewSubscriptionCreatedEvent("sub-2", "user-2", "Paris", "hourly", "error@example.com"),
			setupMocks: func(emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
				emailProvider.EXPECT().SendEmail(mock.Anything, mock.AnythingOfType("shared.EmailRequest")).Return(assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmailProvider := mockPorts.NewEmailProvider(t)
			mockLogger := setupLoggerMock(t)

			tt.setupMocks(mockEmailProvider, mockLogger)

			consumer := NewSubscriptionEventConsumer(SubscriptionEventConsumerConfig{
				EmailProvider: mockEmailProvider,
				Logger:        mockLogger,
			})

			messageData, err := json.Marshal(tt.event)
			require.NoError(t, err)

			message := &messaging.Message{
				ID:      tt.event.ID(),
				Topic:   tt.event.Topic(),
				Data:    messageData,
				Headers: tt.event.Headers(),
			}

			ctx := context.Background()
			err = consumer.HandleMessage(ctx, message)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCommandConsumer_ProcessSendWeatherUpdatesCommand_Success(t *testing.T) {
	tests := []struct {
		name          string
		command       *shared.SendWeatherUpdatesCommand
		subscriptions []shared.Subscription
		weatherData   *shared.WeatherData
		setupMocks    func(*mockPorts.SubscriptionService, *mockPorts.WeatherService, *mockPorts.EmailProvider, *mockPorts.Logger)
		expectedError bool
	}{
		{
			name:    "successful batch weather update processing",
			command: shared.NewSendWeatherUpdatesCommand("daily"),
			subscriptions: []shared.Subscription{
				{
					ID:        "sub-1",
					UserID:    "user-1",
					City:      "London",
					Email:     "test1@example.com",
					Frequency: shared.FrequencyDaily,
					IsActive:  true,
				},
				{
					ID:        "sub-2",
					UserID:    "user-2",
					City:      "Paris",
					Email:     "test2@example.com",
					Frequency: shared.FrequencyDaily,
					IsActive:  true,
				},
			},
			weatherData: &shared.WeatherData{
				City:        "London",
				Temperature: 20.0,
				Humidity:    60.0,
				Description: "Clear sky",
				LastUpdated: time.Now(),
			},
			setupMocks: func(subService *mockPorts.SubscriptionService, weatherService *mockPorts.WeatherService, emailProvider *mockPorts.EmailProvider, logger *mockPorts.Logger) {
				subService.EXPECT().GetActiveSubscriptionsByFrequency(mock.Anything, shared.FrequencyDaily).
					Return([]shared.Subscription{
						{
							ID:        "sub-1",
							UserID:    "user-1",
							City:      "London",
							Email:     "test1@example.com",
							Frequency: shared.FrequencyDaily,
							IsActive:  true,
						},
					}, nil)

				weatherService.EXPECT().GetWeatherByCity(mock.Anything, "London").
					Return(&shared.WeatherData{
						City:        "London",
						Temperature: 20.0,
						Humidity:    60.0,
						Description: "Clear sky",
						LastUpdated: time.Now(),
					}, nil)

				emailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(req shared.EmailRequest) bool {
					return req.To == "test1@example.com"
				})).Return(nil)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSubService := mockPorts.NewSubscriptionService(t)
			mockWeatherService := mockPorts.NewWeatherService(t)
			mockEmailProvider := mockPorts.NewEmailProvider(t)
			mockLogger := setupLoggerMock(t)

			tt.setupMocks(mockSubService, mockWeatherService, mockEmailProvider, mockLogger)

			consumer := NewCommandConsumer(CommandConsumerConfig{
				SubscriptionService: mockSubService,
				WeatherService:      mockWeatherService,
				EmailProvider:       mockEmailProvider,
				Logger:              mockLogger,
			})

			messageData, err := json.Marshal(tt.command)
			require.NoError(t, err)

			message := &messaging.Message{
				ID:      tt.command.ID(),
				Topic:   tt.command.Topic(),
				Data:    messageData,
				Headers: tt.command.Headers(),
			}

			ctx := context.Background()
			err = consumer.HandleMessage(ctx, message)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsumerManager_LifecycleManagement(t *testing.T) {
	mockBroker := mockPorts.NewMessageBroker(t)
	mockLogger := setupLoggerMock(t)

	manager := NewConsumerManager(ConsumerManagerConfig{
		Broker: mockBroker,
		Logger: mockLogger,
	})

	ctx := context.Background()

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

	err := manager.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	err = manager.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, manager.IsRunning())
}

func TestConsumerManager_ErrorHandling(t *testing.T) {
	mockBroker := mockPorts.NewMessageBroker(t)
	mockLogger := setupLoggerMock(t)

	manager := NewConsumerManager(ConsumerManagerConfig{
		Broker: mockBroker,
		Logger: mockLogger,
	})

	ctx := context.Background()

	mockBroker.EXPECT().SubscribeWithGroup(
		mock.Anything,
		string(shared.EventTypeWeatherDataFetched),
		"notification-service",
		mock.AnythingOfType("messaging.MessageHandler"),
	).Return(assert.AnError)

	err := manager.Start(ctx)
	assert.Error(t, err)
	assert.False(t, manager.IsRunning())
}
