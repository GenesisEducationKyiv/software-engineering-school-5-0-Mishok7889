package notification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/core/shared"
	mockPorts "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
)

// Helper function to set up flexible logger mock expectations
func setupLoggerMock(t *testing.T) *mockPorts.Logger {
	mockLogger := mockPorts.NewLogger(t)

	// Set up flexible mock expectations for variadic logger calls
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()

	return mockLogger
}

func TestUseCase_SendWeatherUpdates_Success(t *testing.T) {
	// Create mocks
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockSubscriptionService := mockPorts.NewSubscriptionService(t)
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	params := SendWeatherUpdateParams{Frequency: "daily"}

	// Setup mock service data
	subscriptionsData := []*ports.SubscriptionServiceData{
		{
			ID:               1,
			Email:            "user1@example.com",
			City:             "London",
			Frequency:        "daily",
			Confirmed:        true,
			UnsubscribeToken: "unsub-token",
		},
	}

	weatherData := &ports.WeatherServiceData{
		Temperature: 20.0,
		Humidity:    65.0,
		Description: "Sunny",
		City:        "London",
		Timestamp:   time.Now(),
	}

	// Setup mock expectations
	mockSubscriptionService.EXPECT().GetConfirmedSubscriptions(mock.Anything, "daily").Return(subscriptionsData, nil)
	mockWeatherService.EXPECT().GetWeather(mock.Anything, "London").Return(weatherData, nil)
	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(req shared.EmailRequest) bool {
		return req.To == "user1@example.com" && len(req.Subject) > 0
	})).Return(nil)
	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{BaseURL: "http://localhost:8080"}).Maybe()

	// Create use case
	uc, err := NewUseCase(UseCaseDependencies{
		WeatherService:      mockWeatherService,
		SubscriptionService: mockSubscriptionService,
		EmailProvider:       mockEmailProvider,
		TokenRepo:           mockTokenRepo,
		SubscriptionRepo:    mockSubRepo,
		Config:              mockConfig,
		Logger:              mockLogger,
	})
	assert.NoError(t, err)

	// Execute
	ctx := context.Background()
	err = uc.SendWeatherUpdates(ctx, params)

	// Assert
	assert.NoError(t, err)

	// Verify mocks
	mockSubscriptionService.AssertExpectations(t)
	mockWeatherService.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_SendWeatherUpdates_ValidationError(t *testing.T) {
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockSubscriptionService := mockPorts.NewSubscriptionService(t)
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	// Empty frequency should cause validation error
	params := SendWeatherUpdateParams{Frequency: ""}

	uc, err := NewUseCase(UseCaseDependencies{
		WeatherService:      mockWeatherService,
		SubscriptionService: mockSubscriptionService,
		EmailProvider:       mockEmailProvider,
		TokenRepo:           mockTokenRepo,
		SubscriptionRepo:    mockSubRepo,
		Config:              mockConfig,
		Logger:              mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.SendWeatherUpdates(ctx, params)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	if assert.ErrorAs(t, err, &domainErr) {
		assert.Equal(t, shared.ErrCodeValidation, domainErr.Code)
	} else {
		// If it's not a domain error, check if it's a validation error by message
		assert.Contains(t, err.Error(), "frequency")
	}

	// Verify no unexpected calls were made
	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_SendWeatherUpdates_NoSubscriptions(t *testing.T) {
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockSubscriptionService := mockPorts.NewSubscriptionService(t)
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	params := SendWeatherUpdateParams{Frequency: "daily"}

	// No subscriptions found
	mockSubscriptionService.EXPECT().GetConfirmedSubscriptions(mock.Anything, "daily").Return([]*ports.SubscriptionServiceData{}, nil)

	uc, err := NewUseCase(UseCaseDependencies{
		WeatherService:      mockWeatherService,
		SubscriptionService: mockSubscriptionService,
		EmailProvider:       mockEmailProvider,
		TokenRepo:           mockTokenRepo,
		SubscriptionRepo:    mockSubRepo,
		Config:              mockConfig,
		Logger:              mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.SendWeatherUpdates(ctx, params)

	// Should not be an error, just no work to do
	assert.NoError(t, err)

	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_CleanupExpiredTokens(t *testing.T) {
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockSubscriptionService := mockPorts.NewSubscriptionService(t)
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	// Mock successful cleanup
	mockTokenRepo.EXPECT().DeleteExpiredTokens(mock.Anything).Return(int64(5), nil)

	uc, err := NewUseCase(UseCaseDependencies{
		WeatherService:      mockWeatherService,
		SubscriptionService: mockSubscriptionService,
		EmailProvider:       mockEmailProvider,
		TokenRepo:           mockTokenRepo,
		SubscriptionRepo:    mockSubRepo,
		Config:              mockConfig,
		Logger:              mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.CleanupExpiredTokens(ctx)

	assert.NoError(t, err)

	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_GetNotificationStats(t *testing.T) {
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherService := mockPorts.NewWeatherService(t)
	mockSubscriptionService := mockPorts.NewSubscriptionService(t)
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	// Mock repository calls for stats
	mockSubRepo.EXPECT().CountByFrequency(mock.Anything, "hourly").Return(int64(10), nil)
	mockSubRepo.EXPECT().CountByFrequency(mock.Anything, "daily").Return(int64(25), nil)
	mockSubRepo.EXPECT().CountConfirmed(mock.Anything).Return(int64(35), nil)

	uc, err := NewUseCase(UseCaseDependencies{
		WeatherService:      mockWeatherService,
		SubscriptionService: mockSubscriptionService,
		EmailProvider:       mockEmailProvider,
		TokenRepo:           mockTokenRepo,
		SubscriptionRepo:    mockSubRepo,
		Config:              mockConfig,
		Logger:              mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	stats, err := uc.GetNotificationStats(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 35, stats.TotalSubscriptions)
	assert.Equal(t, 10, stats.HourlySubscriptions)
	assert.Equal(t, 25, stats.DailySubscriptions)

	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_Constructor_Validation(t *testing.T) {
	tests := []struct {
		name    string
		deps    UseCaseDependencies
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing_weather_service",
			deps: UseCaseDependencies{
				WeatherService:      nil,
				SubscriptionService: mockPorts.NewSubscriptionService(t),
				EmailProvider:       mockPorts.NewEmailProvider(t),
				TokenRepo:           mockPorts.NewTokenRepository(t),
				SubscriptionRepo:    mockPorts.NewSubscriptionRepository(t),
				Config:              mockPorts.NewConfigProvider(t),
				Logger:              setupLoggerMock(t),
			},
			wantErr: true,
			errMsg:  "weather service is required",
		},
		{
			name: "missing_subscription_service",
			deps: UseCaseDependencies{
				WeatherService:      mockPorts.NewWeatherService(t),
				SubscriptionService: nil,
				EmailProvider:       mockPorts.NewEmailProvider(t),
				TokenRepo:           mockPorts.NewTokenRepository(t),
				SubscriptionRepo:    mockPorts.NewSubscriptionRepository(t),
				Config:              mockPorts.NewConfigProvider(t),
				Logger:              setupLoggerMock(t),
			},
			wantErr: true,
			errMsg:  "subscription service is required",
		},
		{
			name: "valid_dependencies",
			deps: UseCaseDependencies{
				WeatherService:      mockPorts.NewWeatherService(t),
				SubscriptionService: mockPorts.NewSubscriptionService(t),
				EmailProvider:       mockPorts.NewEmailProvider(t),
				TokenRepo:           mockPorts.NewTokenRepository(t),
				SubscriptionRepo:    mockPorts.NewSubscriptionRepository(t),
				Config:              mockPorts.NewConfigProvider(t),
				Logger:              setupLoggerMock(t),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, err := NewUseCase(tt.deps)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, uc)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, uc)
			}
		})
	}
}
