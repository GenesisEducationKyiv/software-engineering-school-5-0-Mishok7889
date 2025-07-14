package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"weatherapi.app/internal/adapters/middleware"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
)

// SubscriptionTestDependencies contains all dependencies for subscription handler tests
type SubscriptionTestDependencies struct {
	Router           *gin.Engine
	SubscriptionRepo *mocks.SubscriptionRepository
	TokenRepo        *mocks.TokenRepository
	TokenGenerator   *mocks.TokenGenerator
	EmailProvider    *mocks.EmailProvider
}

func setupSubscriptionTestRouter(t *testing.T) SubscriptionTestDependencies {
	gin.SetMode(gin.TestMode)

	mockSubscriptionRepo := mocks.NewSubscriptionRepository(t)
	mockTokenRepo := mocks.NewTokenRepository(t)
	mockTokenGenerator := mocks.NewTokenGenerator(t)
	mockEmailProvider := mocks.NewEmailProvider(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := mocks.NewLogger(t)

	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{
		BaseURL: "http://localhost:8080",
	}).Maybe()

	subscriptionUseCase, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo: mockSubscriptionRepo,
		TokenRepo:        mockTokenRepo,
		TokenGenerator:   mockTokenGenerator,
		EmailProvider:    mockEmailProvider,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	server := &HTTPServerAdapter{
		subscriptionUseCase: subscriptionUseCase,
		logger:              mockLogger,
	}

	validationMiddleware := middleware.NewValidationMiddleware()
	router := gin.New()
	router.POST("/api/subscribe", server.subscribe)
	router.GET("/api/confirm/:token", validationMiddleware.ValidateTokenParam(), server.confirmSubscription)
	router.GET("/api/unsubscribe/:token", validationMiddleware.ValidateTokenParam(), server.unsubscribe)

	return SubscriptionTestDependencies{
		Router:           router,
		SubscriptionRepo: mockSubscriptionRepo,
		TokenRepo:        mockTokenRepo,
		TokenGenerator:   mockTokenGenerator,
		EmailProvider:    mockEmailProvider,
	}
}

func TestSubscriptionHandler_Subscribe_Success_JSON(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	// Mock the repository calls
	deps.SubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, "test@example.com", "London").
		Return(nil, ports.NewNotFoundError("not found"))

	deps.SubscriptionRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == "test@example.com" && sub.City == "London" && sub.Frequency == "daily"
		})).
		Return(nil).
		Run(func(ctx context.Context, sub *ports.SubscriptionData) {
			sub.ID = 1 // Simulate database ID assignment
		})

	deps.TokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.SubscriptionID == uint(1) && token.Type == "confirmation"
		})).
		Return(nil).
		Run(func(ctx context.Context, token *ports.TokenData) {
			token.Value = "test-token" // Simulate token generation
		})

	deps.TokenGenerator.EXPECT().
		GenerateToken().
		Return("test-token")

	deps.EmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	reqBody := SubscriptionRequest{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Subscription successful")
}

func TestSubscriptionHandler_Subscribe_Success_Form(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	// Mock the repository calls
	deps.SubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, "test@example.com", "London").
		Return(nil, ports.NewNotFoundError("not found"))

	deps.SubscriptionRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == "test@example.com" && sub.City == "London" && sub.Frequency == "hourly"
		})).
		Return(nil).
		Run(func(ctx context.Context, sub *ports.SubscriptionData) {
			sub.ID = 1
		})

	deps.TokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.SubscriptionID == uint(1) && token.Type == "confirmation"
		})).
		Return(nil).
		Run(func(ctx context.Context, token *ports.TokenData) {
			token.Value = "test-token" // Simulate token generation
		})

	deps.TokenGenerator.EXPECT().
		GenerateToken().
		Return("test-token")

	deps.EmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	formData := url.Values{}
	formData.Set("email", "test@example.com")
	formData.Set("city", "London")
	formData.Set("frequency", "hourly")

	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Subscription successful")
}

func TestSubscriptionHandler_Subscribe_InvalidEmail(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	reqBody := SubscriptionRequest{
		Email:     "invalid-email",
		City:      "London",
		Frequency: "daily",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "Invalid request format")
}

func TestSubscriptionHandler_Subscribe_MissingFields(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	tests := []struct {
		name string
		body SubscriptionRequest
	}{
		{
			name: "missing email",
			body: SubscriptionRequest{
				City:      "London",
				Frequency: "daily",
			},
		},
		{
			name: "missing city",
			body: SubscriptionRequest{
				Email:     "test@example.com",
				Frequency: "daily",
			},
		},
		{
			name: "missing frequency",
			body: SubscriptionRequest{
				Email: "test@example.com",
				City:  "London",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			deps.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response.Error, "Invalid request format")
		})
	}
}

func TestSubscriptionHandler_Subscribe_AlreadyExists(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	// Mock that subscription already exists and is confirmed
	existingSubscription := &ports.SubscriptionData{
		ID:        1,
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	deps.SubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, "test@example.com", "London").
		Return(existingSubscription, nil)

	reqBody := SubscriptionRequest{
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "already subscribed")
}

func TestSubscriptionHandler_ConfirmSubscription_Success(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	// Mock token lookup
	tokenData := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	deps.TokenRepo.EXPECT().
		FindByToken(mock.Anything, "test-token-123").
		Return(tokenData, nil)

	// Mock subscription lookup
	subscriptionData := &ports.SubscriptionData{
		ID:        1,
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: false,
	}

	deps.SubscriptionRepo.EXPECT().
		FindByID(mock.Anything, uint(1)).
		Return(subscriptionData, nil)

	// Mock subscription update
	deps.SubscriptionRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.ID == 1 && sub.Confirmed == true
		})).
		Return(nil)

	// Mock token deletion
	deps.TokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Mock welcome email
	deps.TokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.SubscriptionID == uint(1) && token.Type == "unsubscribe"
		})).
		Return(nil).
		Run(func(ctx context.Context, token *ports.TokenData) {
			token.Value = "unsubscribe-token" // Simulate token generation
		})

	deps.TokenGenerator.EXPECT().
		GenerateToken().
		Return("unsubscribe-token")

	deps.EmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	req := httptest.NewRequest("GET", "/api/confirm/test-token-123", nil)
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "confirmed successfully")
}

func TestSubscriptionHandler_Unsubscribe_Success(t *testing.T) {
	deps := setupSubscriptionTestRouter(t)

	// Mock token lookup
	tokenData := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "unsubscribe",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	deps.TokenRepo.EXPECT().
		FindByToken(mock.Anything, "test-token-123").
		Return(tokenData, nil)

	// Mock subscription lookup
	subscriptionData := &ports.SubscriptionData{
		ID:        1,
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	deps.SubscriptionRepo.EXPECT().
		FindByID(mock.Anything, uint(1)).
		Return(subscriptionData, nil)

	// Mock subscription deletion
	deps.SubscriptionRepo.EXPECT().
		Delete(mock.Anything, subscriptionData).
		Return(nil)

	// Mock token deletion
	deps.TokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Mock confirmation email
	deps.EmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	req := httptest.NewRequest("GET", "/api/unsubscribe/test-token-123", nil)
	w := httptest.NewRecorder()

	deps.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Unsubscribed successfully")
}
