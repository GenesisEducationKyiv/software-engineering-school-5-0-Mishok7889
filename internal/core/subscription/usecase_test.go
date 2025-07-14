package subscription

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

func TestUseCase_Subscribe_Success(t *testing.T) {
	// Create mocks using mockery
	mockSubRepo := mocks.NewSubscriptionRepository(t)
	mockTokenRepo := mocks.NewTokenRepository(t)
	mockTokenGenerator := mocks.NewTokenGenerator(t)
	mockEmailProvider := mocks.NewEmailProvider(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	params := SubscribeParams{
		Email:     "test@example.com",
		City:      "London",
		Frequency: FrequencyDaily,
	}

	// Setup mock expectations
	// No existing subscription
	mockSubRepo.EXPECT().FindByEmail(mock.Anything, "test@example.com", "London").Return((*ports.SubscriptionData)(nil), shared.NewNotFoundError("not found"))

	// Create subscription
	mockSubRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
		return sub.Email == "test@example.com" &&
			sub.City == "London" &&
			sub.Frequency == FrequencyDaily.String() &&
			!sub.Confirmed
	})).Return(nil).Run(func(ctx context.Context, sub *ports.SubscriptionData) {
		// Simulate database setting the ID
		sub.ID = 1
	})

	// Create confirmation token
	mockTokenGenerator.EXPECT().
		GenerateToken().
		Return("test-confirmation-token")

	mockTokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.Type == TokenTypeConfirmation.String() && token.Value == "test-confirmation-token"
		})).
		Return(nil)

	// Send confirmation email
	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
		return params.To == "test@example.com" && len(params.Subject) > 0
	})).Return(nil)

	// Mock config for app base URL
	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{
		BaseURL: "http://localhost:8080",
	})

	// Create use case
	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		TokenGenerator:   mockTokenGenerator,
		EmailProvider:    mockEmailProvider,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	// Execute
	ctx := context.Background()
	err = uc.Subscribe(ctx, params)

	// Assert
	assert.NoError(t, err)

	// Verify mocks
	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_Subscribe_ValidationError(t *testing.T) {
	tests := []struct {
		name   string
		params SubscribeParams
		errMsg string
	}{
		{
			name: "invalid_email_format",
			params: SubscribeParams{
				Email:     "invalid-email",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			errMsg: "invalid email format",
		},
		{
			name: "empty_email",
			params: SubscribeParams{
				Email:     "",
				City:      "London",
				Frequency: FrequencyDaily,
			},
			errMsg: "email is required",
		},
		{
			name: "empty_city",
			params: SubscribeParams{
				Email:     "test@example.com",
				City:      "",
				Frequency: FrequencyDaily,
			},
			errMsg: "city is required",
		},
		{
			name: "invalid_frequency",
			params: SubscribeParams{
				Email:     "test@example.com",
				City:      "London",
				Frequency: FrequencyUnknown,
			},
			errMsg: "invalid frequency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSubRepo := mocks.NewSubscriptionRepository(t)
			mockTokenRepo := mocks.NewTokenRepository(t)
			mockTokenGenerator := mocks.NewTokenGenerator(t)
			mockEmailProvider := mocks.NewEmailProvider(t)
			mockConfig := mocks.NewConfigProvider(t)

			// No mock expectations - validation should fail before any calls
			// Allow logger calls that might occur during validation
			mockLogger := setupLoggerMock(t)

			uc, err := NewUseCase(UseCaseDependencies{
				SubscriptionRepo: mockSubRepo,
				TokenRepo:        mockTokenRepo,
				TokenGenerator:   mockTokenGenerator,
				EmailProvider:    mockEmailProvider,
				Config:           mockConfig,
				Logger:           mockLogger,
			})
			assert.NoError(t, err)

			ctx := context.Background()
			err = uc.Subscribe(ctx, tt.params)

			assert.Error(t, err)
			var domainErr *shared.DomainError
			if assert.ErrorAs(t, err, &domainErr) {
				assert.Equal(t, shared.ErrCodeValidation, domainErr.Code)
			}
			assert.Contains(t, err.Error(), tt.errMsg)

			// Verify no unexpected calls were made
			mockSubRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
			mockEmailProvider.AssertExpectations(t)
			mockConfig.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestUseCase_Subscribe_AlreadyExists(t *testing.T) {
	mockSubRepo := mocks.NewSubscriptionRepository(t)
	mockTokenRepo := mocks.NewTokenRepository(t)
	mockTokenGenerator := mocks.NewTokenGenerator(t)
	mockEmailProvider := mocks.NewEmailProvider(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	params := SubscribeParams{
		Email:     "existing@example.com",
		City:      "Paris",
		Frequency: FrequencyHourly,
	}

	// Existing confirmed subscription
	existingSub := &ports.SubscriptionData{
		ID:        1,
		Email:     "existing@example.com",
		City:      "Paris",
		Frequency: FrequencyHourly.String(),
		Confirmed: true,
	}
	mockSubRepo.EXPECT().FindByEmail(mock.Anything, "existing@example.com", "Paris").Return(existingSub, nil)

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		TokenGenerator:   mockTokenGenerator,
		EmailProvider:    mockEmailProvider,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.Subscribe(ctx, params)

	assert.Error(t, err)
	var domainErr *shared.DomainError
	if assert.ErrorAs(t, err, &domainErr) {
		assert.Equal(t, shared.ErrCodeAlreadyExists, domainErr.Code)
	} else {
		// If it's not a domain error, check if it's an "already exists" error by message
		assert.Contains(t, err.Error(), "already exists")
	}

	mockSubRepo.AssertExpectations(t)
	mockTokenRepo.AssertExpectations(t)
	mockEmailProvider.AssertExpectations(t)
	mockConfig.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestUseCase_ConfirmSubscription_Success(t *testing.T) {
	mockSubRepo := mocks.NewSubscriptionRepository(t)
	mockTokenRepo := mocks.NewTokenRepository(t)
	mockTokenGenerator := mocks.NewTokenGenerator(t)
	mockEmailProvider := mocks.NewEmailProvider(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := setupLoggerMock(t)

	params := ConfirmParams{Token: "valid-confirmation-token"}

	// Valid token
	token := &ports.TokenData{
		ID:             1,
		Value:          "valid-confirmation-token",
		SubscriptionID: 1,
		Type:           TokenTypeConfirmation.String(),
		ExpiresAt:      time.Now().Add(24 * time.Hour), // Set expiration in the future
	}
	mockTokenRepo.EXPECT().FindByToken(mock.Anything, "valid-confirmation-token").Return(token, nil)

	// Valid subscription
	subscription := &ports.SubscriptionData{
		ID:        1,
		Email:     "test@example.com",
		City:      "London",
		Frequency: FrequencyDaily.String(),
		Confirmed: false,
	}
	mockSubRepo.EXPECT().FindByID(mock.Anything, uint(1)).Return(subscription, nil)

	// Update subscription to confirmed
	mockSubRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
		return sub.ID == 1 && sub.Confirmed
	})).Return(nil)

	// Create unsubscribe token
	mockTokenGenerator.EXPECT().
		GenerateToken().
		Return("unsubscribe-token")

	mockTokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.SubscriptionID == uint(1) && token.Type == TokenTypeUnsubscribe.String() && token.Value == "unsubscribe-token"
		})).
		Return(nil)

	// Send welcome email
	mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
		return params.To == "test@example.com"
	})).Return(nil)

	// Mock config for app base URL
	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{
		BaseURL: "http://localhost:8080",
	})

	// Delete confirmation token
	mockTokenRepo.EXPECT().Delete(mock.Anything, token).Return(nil)

	uc, err := NewUseCase(UseCaseDependencies{
		SubscriptionRepo: mockSubRepo,
		TokenRepo:        mockTokenRepo,
		TokenGenerator:   mockTokenGenerator,
		EmailProvider:    mockEmailProvider,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	ctx := context.Background()
	err = uc.ConfirmSubscription(ctx, params)

	assert.NoError(t, err)

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
				TokenRepo:        mocks.NewTokenRepository(t),
				TokenGenerator:   mocks.NewTokenGenerator(t),
				EmailProvider:    mocks.NewEmailProvider(t),
				Config:           mocks.NewConfigProvider(t),
				Logger:           mocks.NewLogger(t),
			},
			wantErr: true,
			errMsg:  "subscription repository is required",
		},
		{
			name: "missing_token_generator",
			deps: UseCaseDependencies{
				SubscriptionRepo: mocks.NewSubscriptionRepository(t),
				TokenRepo:        mocks.NewTokenRepository(t),
				TokenGenerator:   nil,
				EmailProvider:    mocks.NewEmailProvider(t),
				Config:           mocks.NewConfigProvider(t),
				Logger:           mocks.NewLogger(t),
			},
			wantErr: true,
			errMsg:  "token generator is required",
		},
		{
			name: "valid_dependencies",
			deps: UseCaseDependencies{
				SubscriptionRepo: mocks.NewSubscriptionRepository(t),
				TokenRepo:        mocks.NewTokenRepository(t),
				TokenGenerator:   mocks.NewTokenGenerator(t),
				EmailProvider:    mocks.NewEmailProvider(t),
				Config:           mocks.NewConfigProvider(t),
				Logger:           mocks.NewLogger(t),
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

func TestFrequency_Validation(t *testing.T) {
	tests := []struct {
		name      string
		frequency Frequency
		isValid   bool
	}{
		{"valid_hourly", FrequencyHourly, true},
		{"valid_daily", FrequencyDaily, true},
		{"invalid_empty", FrequencyUnknown, false},
		{"invalid_weekly", FrequencyUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, tt.frequency.IsValid())
		})
	}
}
