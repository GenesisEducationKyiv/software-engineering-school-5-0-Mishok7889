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
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

func setupSubscriptionTestRouter(t *testing.T) (*gin.Engine, *mocks.SubscriptionRepository, *mocks.TokenRepository, *mocks.EmailProvider) {
	gin.SetMode(gin.TestMode)

	// Mock the dependencies
	mockSubscriptionRepo := mocks.NewSubscriptionRepository(t)
	mockTokenRepo := mocks.NewTokenRepository(t)
	mockEmailProvider := mocks.NewEmailProvider(t)
	mockConfig := mocks.NewConfigProvider(t)
	mockLogger := mocks.NewLogger(t)

	// Allow logger calls without strict expectations - handle variadic field parameters
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

	// Mock config provider
	mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{
		BaseURL: "http://localhost:8080",
	}).Maybe()

	// Create real use case with mocked dependencies
	subscriptionUseCase, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo: mockSubscriptionRepo,
		TokenRepo:        mockTokenRepo,
		EmailProvider:    mockEmailProvider,
		Config:           mockConfig,
		Logger:           mockLogger,
	})
	assert.NoError(t, err)

	server := &HTTPServerAdapter{
		subscriptionUseCase: subscriptionUseCase,
	}

	router := gin.New()
	router.POST("/api/subscribe", server.subscribe)
	router.GET("/api/confirm/:token", server.confirmSubscription)
	router.GET("/api/unsubscribe/:token", server.unsubscribe)

	return router, mockSubscriptionRepo, mockTokenRepo, mockEmailProvider
}

func TestSubscriptionHandler_Subscribe_Success_JSON(t *testing.T) {
	router, mockSubscriptionRepo, mockTokenRepo, mockEmailProvider := setupSubscriptionTestRouter(t)

	// Mock the repository calls
	mockSubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, "test@example.com", "London").
		Return(nil, errors.NewNotFoundError("not found"))

	mockSubscriptionRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == "test@example.com" && sub.City == "London" && sub.Frequency == "daily"
		})).
		Return(nil).
		Run(func(ctx context.Context, sub *ports.SubscriptionData) {
			sub.ID = 1 // Simulate database ID assignment
		})

	mockTokenRepo.EXPECT().
		CreateConfirmationToken(mock.Anything, uint(1), mock.Anything).
		Return(&ports.TokenData{
			Value: "test-token",
		}, nil)

	mockEmailProvider.EXPECT().
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

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Subscription successful")
}

func TestSubscriptionHandler_Subscribe_Success_Form(t *testing.T) {
	router, mockSubscriptionRepo, mockTokenRepo, mockEmailProvider := setupSubscriptionTestRouter(t)

	// Mock the repository calls
	mockSubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, "test@example.com", "London").
		Return(nil, errors.NewNotFoundError("not found"))

	mockSubscriptionRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == "test@example.com" && sub.City == "London" && sub.Frequency == "hourly"
		})).
		Return(nil).
		Run(func(ctx context.Context, sub *ports.SubscriptionData) {
			sub.ID = 1
		})

	mockTokenRepo.EXPECT().
		CreateConfirmationToken(mock.Anything, uint(1), mock.Anything).
		Return(&ports.TokenData{
			Value: "test-token",
		}, nil)

	mockEmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	formData := url.Values{}
	formData.Set("email", "test@example.com")
	formData.Set("city", "London")
	formData.Set("frequency", "hourly")

	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Subscription successful")
}

func TestSubscriptionHandler_Subscribe_InvalidEmail(t *testing.T) {
	router, _, _, _ := setupSubscriptionTestRouter(t)

	reqBody := SubscriptionRequest{
		Email:     "invalid-email",
		City:      "London",
		Frequency: "daily",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "invalid request format")
}

func TestSubscriptionHandler_Subscribe_MissingFields(t *testing.T) {
	router, _, _, _ := setupSubscriptionTestRouter(t)

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

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response.Error, "invalid request format")
		})
	}
}

func TestSubscriptionHandler_Subscribe_AlreadyExists(t *testing.T) {
	router, mockSubscriptionRepo, _, _ := setupSubscriptionTestRouter(t)

	// Mock that subscription already exists and is confirmed
	existingSubscription := &ports.SubscriptionData{
		ID:        1,
		Email:     "test@example.com",
		City:      "London",
		Frequency: "daily",
		Confirmed: true,
	}

	mockSubscriptionRepo.EXPECT().
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

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "already exists")
}

func TestSubscriptionHandler_ConfirmSubscription_Success(t *testing.T) {
	router, mockSubscriptionRepo, mockTokenRepo, mockEmailProvider := setupSubscriptionTestRouter(t)

	// Mock token lookup
	tokenData := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	mockTokenRepo.EXPECT().
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

	mockSubscriptionRepo.EXPECT().
		FindByID(mock.Anything, uint(1)).
		Return(subscriptionData, nil)

	// Mock subscription update
	mockSubscriptionRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.ID == 1 && sub.Confirmed == true
		})).
		Return(nil)

	// Mock token deletion
	mockTokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Mock welcome email
	mockTokenRepo.EXPECT().
		CreateUnsubscribeToken(mock.Anything, uint(1), mock.Anything).
		Return(&ports.TokenData{
			Value: "unsubscribe-token",
		}, nil)

	mockEmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	req := httptest.NewRequest("GET", "/api/confirm/test-token-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "confirmed successfully")
}

func TestSubscriptionHandler_ConfirmSubscription_InvalidToken(t *testing.T) {
	router, _, mockTokenRepo, _ := setupSubscriptionTestRouter(t)

	mockTokenRepo.EXPECT().
		FindByToken(mock.Anything, "invalid-token").
		Return(nil, errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/confirm/invalid-token", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // Token errors return 400, not 404

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "token")
}

func TestSubscriptionHandler_Unsubscribe_Success(t *testing.T) {
	router, mockSubscriptionRepo, mockTokenRepo, mockEmailProvider := setupSubscriptionTestRouter(t)

	// Mock token lookup
	tokenData := &ports.TokenData{
		Value:          "test-token-123",
		SubscriptionID: 1,
		Type:           "unsubscribe",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	mockTokenRepo.EXPECT().
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

	mockSubscriptionRepo.EXPECT().
		FindByID(mock.Anything, uint(1)).
		Return(subscriptionData, nil)

	// Mock subscription deletion
	mockSubscriptionRepo.EXPECT().
		Delete(mock.Anything, subscriptionData).
		Return(nil)

	// Mock token deletion
	mockTokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Mock confirmation email
	mockEmailProvider.EXPECT().
		SendEmail(mock.Anything, mock.Anything).
		Return(nil)

	req := httptest.NewRequest("GET", "/api/unsubscribe/test-token-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Unsubscribed successfully")
}

func TestSubscriptionHandler_Unsubscribe_InvalidToken(t *testing.T) {
	router, _, mockTokenRepo, _ := setupSubscriptionTestRouter(t)

	mockTokenRepo.EXPECT().
		FindByToken(mock.Anything, "invalid-token").
		Return(nil, errors.NewNotFoundError("token not found"))

	req := httptest.NewRequest("GET", "/api/unsubscribe/invalid-token", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // Token errors return 400, not 404

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "token")
}
