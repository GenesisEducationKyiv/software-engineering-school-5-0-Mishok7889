package weather

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/core/shared"
	mocks "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
)

// Helper function to set up flexible logger mock expectations
func setupLoggerMock(t *testing.T) *mocks.Logger {
	mockLogger := mocks.NewLogger(t)

	// Set up flexible mock expectations for variadic logger calls
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	return mockLogger
}

func TestUseCase_GetWeather_Success(t *testing.T) {
	// Create mocks using mockery
	mockWeatherProviderManager := mocks.NewWeatherProviderManager(t)
	mockWeatherCache := mocks.NewWeatherCache(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)
	mockMetrics := mocks.NewWeatherMetrics(t)

	// Setup mock expectations
	expectedWeatherData := &ports.WeatherData{
		Temperature: 20.0,
		Humidity:    65.0,
		Description: "Sunny",
		City:        "London",
	}

	// Mock config to enable cache
	mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    time.Duration(10) * time.Minute,
	})

	// Cache miss, then provider success
	mockWeatherCache.EXPECT().Get(mock.Anything, "weather:London").Return((*ports.WeatherData)(nil), shared.NewNotFoundError("cache miss"))
	mockWeatherProviderManager.EXPECT().GetWeather(mock.Anything, "London").Return(expectedWeatherData, nil)
	mockWeatherCache.EXPECT().Set(mock.Anything, "weather:London", expectedWeatherData, mock.Anything).Return(nil)

	// Create use case with mocked dependencies
	uc, err := NewUseCase(UseCaseDependencies{
		WeatherProvider: mockWeatherProviderManager,
		Cache:           mockWeatherCache,
		Config:          mockConfig,
		Logger:          mockLogger,
		Metrics:         mockMetrics,
	})
	assert.NoError(t, err)

	// Execute
	ctx := context.Background()
	result, err := uc.GetWeather(ctx, WeatherRequest{City: "London"})

	// Assert results
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedWeatherData.Temperature, result.Temperature)
	assert.Equal(t, expectedWeatherData.City, result.City)

	// Verify all mock expectations were met
	mockWeatherProviderManager.AssertExpectations(t)
	mockWeatherCache.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestUseCase_GetWeather_ValidationError(t *testing.T) {
	// Create mocks
	mockWeatherProviderManager := mocks.NewWeatherProviderManager(t)
	mockWeatherCache := mocks.NewWeatherCache(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)
	mockMetrics := mocks.NewWeatherMetrics(t)

	// No mock expectations needed - validation should fail before any calls

	// Create use case
	uc, err := NewUseCase(UseCaseDependencies{
		WeatherProvider: mockWeatherProviderManager,
		Cache:           mockWeatherCache,
		Config:          mockConfig,
		Logger:          mockLogger,
		Metrics:         mockMetrics,
	})
	assert.NoError(t, err)

	// Execute with invalid request
	ctx := context.Background()
	result, err := uc.GetWeather(ctx, WeatherRequest{City: ""})

	// Assert validation error
	assert.Error(t, err)
	assert.Nil(t, result)

	var domainErr *shared.DomainError
	if assert.ErrorAs(t, err, &domainErr) {
		assert.Equal(t, shared.ErrCodeValidation, domainErr.Code)
	} else {
		// If it's not a domain error, check if it's a validation error by message
		assert.Contains(t, err.Error(), "invalid weather request")
	}

	// Verify no unexpected calls were made
	mockWeatherProviderManager.AssertExpectations(t)
	mockWeatherCache.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestUseCase_GetWeather_ProviderError(t *testing.T) {
	mockWeatherProviderManager := mocks.NewWeatherProviderManager(t)
	mockWeatherCache := mocks.NewWeatherCache(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)
	mockMetrics := mocks.NewWeatherMetrics(t)

	// Mock config to enable cache
	mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    time.Duration(10) * time.Minute,
	})

	// Cache miss, then provider error
	mockWeatherCache.EXPECT().Get(mock.Anything, "weather:NonExistentCity").Return((*ports.WeatherData)(nil), shared.NewNotFoundError("cache miss"))
	mockWeatherProviderManager.EXPECT().GetWeather(mock.Anything, "NonExistentCity").Return((*ports.WeatherData)(nil), shared.NewDomainErrorWithCause(shared.ErrCodeInternal, "city not found", nil))

	uc, err := NewUseCase(UseCaseDependencies{
		WeatherProvider: mockWeatherProviderManager,
		Cache:           mockWeatherCache,
		Config:          mockConfig,
		Logger:          mockLogger,
		Metrics:         mockMetrics,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := uc.GetWeather(ctx, WeatherRequest{City: "NonExistentCity"})

	assert.Error(t, err)
	assert.Nil(t, result)

	var domainErr *shared.DomainError
	if assert.ErrorAs(t, err, &domainErr) {
		assert.Equal(t, shared.ErrCodeExternalService, domainErr.Code)
	} else {
		assert.Contains(t, err.Error(), "weather provider failed")
	}

	mockWeatherProviderManager.AssertExpectations(t)
	mockWeatherCache.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockMetrics.AssertExpectations(t)
}

func TestUseCase_Constructor_Validation(t *testing.T) {
	tests := []struct {
		name    string
		deps    UseCaseDependencies
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing_weather_provider",
			deps: UseCaseDependencies{
				WeatherProvider: nil,
				Cache:           mocks.NewWeatherCache(t),
				Config:          mocks.NewConfigProvider(t),
				Logger:          mocks.NewLogger(t),
				Metrics:         mocks.NewWeatherMetrics(t),
			},
			wantErr: true,
			errMsg:  "weather provider is required",
		},
		{
			name: "valid_dependencies",
			deps: UseCaseDependencies{
				WeatherProvider: mocks.NewWeatherProviderManager(t),
				Cache:           mocks.NewWeatherCache(t),
				Config:          mocks.NewConfigProvider(t),
				Logger:          mocks.NewLogger(t),
				Metrics:         mocks.NewWeatherMetrics(t),
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
