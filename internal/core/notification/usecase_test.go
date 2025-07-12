package notification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	mockPorts "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func TestUseCase_SendWeatherUpdates_Success(t *testing.T) {
	// Create mocks using mockery
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)

	// Create a working weather use case with stub dependencies
	mockWeatherProvider := mockPorts.NewWeatherProviderManager(t)
	mockCache := mockPorts.NewWeatherCache(t)
	mockWeatherConfig := mockPorts.NewConfigProvider(t)
	mockWeatherLogger := mockPorts.NewLogger(t)
	mockWeatherMetrics := mockPorts.NewWeatherMetrics(t)

	// Set up weather use case dependencies
	mockWeatherProvider.EXPECT().GetWeather(mock.Anything, mock.Anything).Return(&ports.WeatherData{
		Temperature: 20.0,
		Humidity:    65.0,
		Description: "Sunny",
		City:        "London",
		Timestamp:   time.Now(),
	}, nil).Maybe()
	mockCache.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, errors.NewNotFoundError("cache miss")).Maybe()
	mockCache.EXPECT().Set(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	mockWeatherConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{EnableCache: false}).Maybe()
	mockWeatherLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockWeatherMetrics.EXPECT().GetProviderInfo().Return(map[string]interface{}{}).Maybe()

	mockWeatherUseCase, _ := weather.NewUseCase(weather.UseCaseDependencies{
		WeatherProvider: mockWeatherProvider,
		Cache:           mockCache,
		Config:          mockWeatherConfig,
		Logger:          mockWeatherLogger,
		Metrics:         mockWeatherMetrics,
	})

	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	params := SendWeatherUpdateParams{Frequency: subscription.FrequencyDaily}

	// Setup mock expectations
	subscriptionsData := []*ports.SubscriptionData{
		{
			ID:        1,
			Email:     "user1@example.com",
			City:      "London",
			Frequency: subscription.FrequencyDaily.String(),
			Confirmed: true,
		},
	}

	mockSubRepo.EXPECT().GetConfirmedByFrequency(mock.Anything, subscription.FrequencyDaily.String()).Return(subscriptionsData, nil)

	// Mock unsubscribe token lookup (not found, so create new one)
	mockTokenRepo.EXPECT().FindBySubscriptionIDAndType(mock.Anything, uint(1), subscription.TokenTypeUnsubscribe.String()).Return(nil, errors.NewNotFoundError("token not found"))
	mockTokenRepo.EXPECT().CreateUnsubscribeToken(mock.Anything, uint(1), mock.Anything).Return(&ports.TokenData{Value: "unsub-token"}, nil)

	// Mock email sending
	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
		return params.To == "user1@example.com" && len(params.Subject) > 0
	})).Return(nil)

	// Mock app config for unsubscribe URL
	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{BaseURL: "http://localhost:8080"}).Maybe()

	// Allow logger calls
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Create use case
	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		WeatherUseCase:   mockWeatherUseCase,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	// Execute
	ctx := context.Background()
	err = uc.SendWeatherUpdates(ctx, params)

	// Assert
	assert.NoError(t, err)

	// Verify mocks
	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_SendWeatherUpdates_ValidationError(t *testing.T) {
	mockSubRepo := mockPorts.NewSubscriptionRepository(t)
	mockTokenRepo := mockPorts.NewTokenRepository(t)
	mockEmailProvider := mockPorts.NewEmailProvider(t)
	mockWeatherUseCase := &weather.UseCase{}
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	// Invalid frequency should cause validation error
	params := SendWeatherUpdateParams{Frequency: subscription.FrequencyUnknown}

	// No mock expectations - validation should fail before any calls

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		WeatherUseCase:   mockWeatherUseCase,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.SendWeatherUpdates(ctx, params)

	assert.Error(t, err)
	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ValidationError, appErr.Type)

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
	mockWeatherUseCase := &weather.UseCase{}
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	params := SendWeatherUpdateParams{Frequency: subscription.FrequencyDaily}

	// No subscriptions found
	mockSubRepo.EXPECT().GetConfirmedByFrequency(mock.Anything, subscription.FrequencyDaily.String()).Return([]*ports.SubscriptionData{}, nil)

	// Allow logger calls
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		WeatherUseCase:   mockWeatherUseCase,
		Config:           mockConfig,
		Logger:           mockLogger,
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
	mockWeatherUseCase := &weather.UseCase{}
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	// Mock successful cleanup
	mockTokenRepo.EXPECT().DeleteExpiredTokens(mock.Anything).Return(int64(5), nil)

	// Allow logger calls
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		WeatherUseCase:   mockWeatherUseCase,
		Config:           mockConfig,
		Logger:           mockLogger,
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
	mockWeatherUseCase := &weather.UseCase{}
	mockConfig := mockPorts.NewConfigProvider(t)
	mockLogger := mockPorts.NewLogger(t)

	// Mock repository calls for stats
	mockSubRepo.EXPECT().CountByFrequency(mock.Anything, subscription.FrequencyHourly.String()).Return(int64(10), nil)
	mockSubRepo.EXPECT().CountByFrequency(mock.Anything, subscription.FrequencyDaily.String()).Return(int64(25), nil)
	mockSubRepo.EXPECT().CountConfirmed(mock.Anything).Return(int64(35), nil)

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		WeatherUseCase:   mockWeatherUseCase,
		Config:           mockConfig,
		Logger:           mockLogger,
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
			name: "missing_subscription_repository",
			deps: UseCaseDependencies{
				SubscriptionRepo: nil,
				TokenRepo:        mockPorts.NewTokenRepository(t),
				EmailProvider:    mockPorts.NewEmailProvider(t),
				WeatherUseCase:   &weather.UseCase{},
				Config:           mockPorts.NewConfigProvider(t),
				Logger:           mockPorts.NewLogger(t),
			},
			wantErr: true,
			errMsg:  "subscription repository is required",
		},
		{
			name: "valid_dependencies",
			deps: UseCaseDependencies{
				SubscriptionRepo: mockPorts.NewSubscriptionRepository(t),
				TokenRepo:        mockPorts.NewTokenRepository(t),
				EmailProvider:    mockPorts.NewEmailProvider(t),
				WeatherUseCase:   &weather.UseCase{},
				Config:           mockPorts.NewConfigProvider(t),
				Logger:           mockPorts.NewLogger(t),
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
