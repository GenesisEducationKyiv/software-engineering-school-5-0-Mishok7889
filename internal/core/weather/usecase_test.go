package weather

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mocks "weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func TestUseCase_GetWeather_Success(t *testing.T) {
	// Create mocks using mockery
	mockWeatherProviderManager := mocks.NewWeatherProviderManager(t)
	mockWeatherCache := mocks.NewWeatherCache(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := mocks.NewLogger(t)
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
	mockWeatherCache.EXPECT().Get(mock.Anything, "weather:London").Return((*ports.WeatherData)(nil), errors.NewNotFoundError("cache miss"))
	mockWeatherProviderManager.EXPECT().GetWeather(mock.Anything, "London").Return(expectedWeatherData, nil)
	mockWeatherCache.EXPECT().Set(mock.Anything, "weather:London", expectedWeatherData, mock.Anything).Return(nil)

	// Allow logger calls with variadic arguments (2 or 3 arguments)
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()

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
	mockLogger := mocks.NewLogger(t)
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

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ValidationError, appErr.Type)

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
	mockLogger := mocks.NewLogger(t)
	mockMetrics := mocks.NewWeatherMetrics(t)

	// Mock config to enable cache
	mockConfig.EXPECT().GetWeatherConfig().Return(ports.WeatherConfig{
		EnableCache: true,
		CacheTTL:    time.Duration(10) * time.Minute,
	})

	// Cache miss, then provider error
	mockWeatherCache.EXPECT().Get(mock.Anything, "weather:NonExistentCity").Return((*ports.WeatherData)(nil), errors.NewNotFoundError("cache miss"))
	mockWeatherProviderManager.EXPECT().GetWeather(mock.Anything, "NonExistentCity").Return((*ports.WeatherData)(nil), errors.NewExternalAPIError("city not found", nil))

	// Allow logger calls with variadic arguments (2 or 3 arguments)
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()

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

	var appErr *errors.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, errors.ExternalAPIError, appErr.Type)

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
