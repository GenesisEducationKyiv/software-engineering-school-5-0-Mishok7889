package subscription_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	subscriptionapi "weatherapi.app/internal/services/subscription/adapters/api"
)

const (
	testTimeout       = 30 * time.Second
	numConcurrentReqs = 10

	// Test data
	testEmail          = "test@example.com"
	testCity           = "London"
	testFrequency      = "hourly"
	testToken          = "test-token-123"
	testSubscriptionID = uint(1)
)

type SubscriptionServiceIntegrationSuite struct {
	suite.Suite

	// Service under test
	httpServer *subscriptionapi.HTTPServer
	router     *gin.Engine

	// Test HTTP client
	httpClient *http.Client

	// Mockery-generated mocks
	mockSubscriptionRepo    *mocks.SubscriptionRepository
	mockTokenRepo           *mocks.TokenRepository
	mockTokenGenerator      *mocks.TokenGenerator
	mockNotificationService *mocks.NotificationService
	mockConfig              *mocks.ConfigProvider
	mockLogger              *mocks.Logger
}

func TestSubscriptionServiceIntegration(t *testing.T) {
	suite.Run(t, new(SubscriptionServiceIntegrationSuite))
}

func (s *SubscriptionServiceIntegrationSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.setupMocks()
	s.createSubscriptionApplication()
	s.setupHTTPClient()
}

func (s *SubscriptionServiceIntegrationSuite) setupMocks() {
	// Create mockery-generated mocks
	s.mockSubscriptionRepo = mocks.NewSubscriptionRepository(s.T())
	s.mockTokenRepo = mocks.NewTokenRepository(s.T())
	s.mockTokenGenerator = mocks.NewTokenGenerator(s.T())
	s.mockNotificationService = mocks.NewNotificationService(s.T())
	s.mockConfig = mocks.NewConfigProvider(s.T())
	s.mockLogger = mocks.NewLogger(s.T())

	// Set up flexible logger mock expectations (logger is called frequently)
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Info(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Info(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Error(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Error(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Warn(mock.Anything, mock.Anything).Maybe()
	s.mockLogger.EXPECT().Warn(mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Set up config mock expectations
	s.mockConfig.EXPECT().GetAppConfig().Return(ports.AppConfig{
		BaseURL: "http://localhost:8080",
	}).Maybe()
	s.mockConfig.EXPECT().GetAppBaseURL().Return("http://localhost:8080").Maybe()
	s.mockConfig.EXPECT().GetEmailConfig().Return(ports.EmailConfig{
		SMTPHost:     "localhost",
		SMTPPort:     587,
		SMTPUsername: "test",
		SMTPPassword: "test",
		FromName:     "Weather API",
		FromAddress:  "noreply@weatherapi.com",
	}).Maybe()
}

func (s *SubscriptionServiceIntegrationSuite) createSubscriptionApplication() {
	// Create subscription use case with mocks
	subscriptionUC, err := subscription.NewUseCase(subscription.UseCaseDependencies{
		SubscriptionRepo:    s.mockSubscriptionRepo,
		TokenRepo:           s.mockTokenRepo,
		TokenGenerator:      s.mockTokenGenerator,
		NotificationService: s.mockNotificationService,
		Config:              s.mockConfig,
		Logger:              s.mockLogger,
	})
	s.Require().NoError(err)

	// Create HTTP server directly (this is what we're testing)
	s.httpServer = subscriptionapi.NewHTTPServer(
		subscriptionapi.ServerConfig{Port: 8080},
		subscriptionUC,
		s.mockLogger,
	)
	s.router = s.httpServer.GetRouter()
}

func (s *SubscriptionServiceIntegrationSuite) setupHTTPClient() {
	s.httpClient = &http.Client{
		Timeout: testTimeout,
	}
}

func (s *SubscriptionServiceIntegrationSuite) makeRequest(method, path string, body interface{}) (*http.Response, error) {
	var requestBody []byte
	var err error

	if body != nil {
		requestBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(requestBody))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	return recorder.Result(), nil
}

func (s *SubscriptionServiceIntegrationSuite) TestSubscribe_Success() {
	// Setup mock expectations
	s.mockSubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, testEmail, testCity).
		Return(nil, ports.NewNotFoundError("not found"))

	s.mockSubscriptionRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == testEmail && sub.City == testCity && sub.Frequency == testFrequency
		})).
		Return(nil).
		Run(func(ctx context.Context, sub *ports.SubscriptionData) {
			sub.ID = testSubscriptionID
		})

	s.mockTokenGenerator.EXPECT().
		GenerateToken().
		Return(testToken)

	s.mockTokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.Value == testToken && token.SubscriptionID == testSubscriptionID
		})).
		Return(nil)

	s.mockNotificationService.EXPECT().
		SendConfirmationEmail(mock.Anything, testEmail, testCity, mock.Anything).
		Return(nil)

	// Make request
	requestBody := map[string]string{
		"email":     testEmail,
		"city":      testCity,
		"frequency": testFrequency,
	}

	resp, err := s.makeRequest("POST", "/api/v1/subscriptions", requestBody)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assert response
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Contains(response["message"], "successful")
}

func (s *SubscriptionServiceIntegrationSuite) TestSubscribe_ValidationError() {
	// Invalid email format
	requestBody := map[string]string{
		"email":     "invalid-email",
		"city":      testCity,
		"frequency": testFrequency,
	}

	resp, err := s.makeRequest("POST", "/api/v1/subscriptions", requestBody)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(http.StatusBadRequest, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Contains(response["error"], "invalid")
}

func (s *SubscriptionServiceIntegrationSuite) TestSubscribe_AlreadyExists() {
	// Setup mock: subscription already exists
	existingSub := &ports.SubscriptionData{
		ID:        testSubscriptionID,
		Email:     testEmail,
		City:      testCity,
		Frequency: testFrequency,
		Confirmed: true,
	}

	s.mockSubscriptionRepo.EXPECT().
		FindByEmail(mock.Anything, testEmail, testCity).
		Return(existingSub, nil)

	// Make request
	requestBody := map[string]string{
		"email":     testEmail,
		"city":      testCity,
		"frequency": testFrequency,
	}

	resp, err := s.makeRequest("POST", "/api/v1/subscriptions", requestBody)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Should return conflict status
	s.Equal(http.StatusConflict, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Contains(strings.ToLower(fmt.Sprintf("%v", response["error"])), "already")
}

func (s *SubscriptionServiceIntegrationSuite) TestConfirm_Success() {
	// Setup mocks
	tokenData := &ports.TokenData{
		Value:          testToken,
		SubscriptionID: testSubscriptionID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	subscriptionData := &ports.SubscriptionData{
		ID:        testSubscriptionID,
		Email:     testEmail,
		City:      testCity,
		Frequency: testFrequency,
		Confirmed: false,
	}

	s.mockTokenRepo.EXPECT().
		FindByToken(mock.Anything, testToken).
		Return(tokenData, nil)

	s.mockSubscriptionRepo.EXPECT().
		FindByID(mock.Anything, testSubscriptionID).
		Return(subscriptionData, nil)

	s.mockSubscriptionRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.ID == testSubscriptionID && sub.Confirmed
		})).
		Return(nil)

	s.mockTokenGenerator.EXPECT().
		GenerateToken().
		Return("unsubscribe-token")

	s.mockTokenRepo.EXPECT().
		Save(mock.Anything, mock.MatchedBy(func(token *ports.TokenData) bool {
			return token.Type == "unsubscribe" && token.SubscriptionID == testSubscriptionID
		})).
		Return(nil)

	s.mockNotificationService.EXPECT().
		SendWelcomeEmail(mock.Anything, testEmail, testCity, testFrequency, mock.Anything).
		Return(nil)

	s.mockTokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Make request
	resp, err := s.makeRequest("GET", fmt.Sprintf("/api/v1/confirm/%s", testToken), nil)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assert response
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Contains(response["message"], "confirmed")
}

func (s *SubscriptionServiceIntegrationSuite) TestUnsubscribe_Success() {
	// Setup mocks
	tokenData := &ports.TokenData{
		Value:          testToken,
		SubscriptionID: testSubscriptionID,
		Type:           "unsubscribe",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	subscriptionData := &ports.SubscriptionData{
		ID:        testSubscriptionID,
		Email:     testEmail,
		City:      testCity,
		Frequency: testFrequency,
		Confirmed: true,
	}

	s.mockTokenRepo.EXPECT().
		FindByToken(mock.Anything, testToken).
		Return(tokenData, nil)

	s.mockSubscriptionRepo.EXPECT().
		FindByID(mock.Anything, testSubscriptionID).
		Return(subscriptionData, nil)

	s.mockSubscriptionRepo.EXPECT().
		Delete(mock.Anything, subscriptionData).
		Return(nil)

	s.mockNotificationService.EXPECT().
		SendUnsubscribeConfirmationEmail(mock.Anything, testEmail, testCity).
		Return(nil)

	s.mockTokenRepo.EXPECT().
		Delete(mock.Anything, tokenData).
		Return(nil)

	// Make request
	resp, err := s.makeRequest("GET", fmt.Sprintf("/api/v1/unsubscribe/%s", testToken), nil)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assert response
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Contains(response["message"], "unsubscribe")
}

func (s *SubscriptionServiceIntegrationSuite) TestHealthCheck() {
	resp, err := s.makeRequest("GET", "/health", nil)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)
	s.Equal("healthy", response["status"])
}

func (s *SubscriptionServiceIntegrationSuite) TestConcurrentSubscriptionRequests() {
	// This tests concurrent safety of the subscription service
	requestChan := make(chan *http.Response, numConcurrentReqs)
	errorChan := make(chan error, numConcurrentReqs)

	// Setup common mock expectations for concurrent requests
	for i := 0; i < numConcurrentReqs; i++ {
		email := fmt.Sprintf("concurrent%d@example.com", i)

		s.mockSubscriptionRepo.EXPECT().
			FindByEmail(mock.Anything, email, testCity).
			Return(nil, shared.NewNotFoundError("not found"))

		s.mockSubscriptionRepo.EXPECT().
			Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
				return sub.Email == email && sub.City == testCity
			})).
			Return(nil).
			Run(func(ctx context.Context, sub *ports.SubscriptionData) {
				sub.ID = uint(i + 1)
			})

		s.mockTokenGenerator.EXPECT().
			GenerateToken().
			Return(fmt.Sprintf("token-%d", i))

		s.mockTokenRepo.EXPECT().
			Save(mock.Anything, mock.Anything).
			Return(nil)

		s.mockNotificationService.EXPECT().
			SendConfirmationEmail(mock.Anything, email, testCity, mock.Anything).
			Return(nil)
	}

	// Execute concurrent requests
	for i := 0; i < numConcurrentReqs; i++ {
		go func(index int) {
			requestBody := map[string]string{
				"email":     fmt.Sprintf("concurrent%d@example.com", index),
				"city":      testCity,
				"frequency": testFrequency,
			}

			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", requestBody)
			if err != nil {
				errorChan <- err
				return
			}
			requestChan <- resp
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numConcurrentReqs; i++ {
		select {
		case resp := <-requestChan:
			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			_ = resp.Body.Close()
		case err := <-errorChan:
			s.T().Errorf("Request failed: %v", err)
		case <-time.After(testTimeout):
			s.T().Errorf("Timeout waiting for concurrent request %d", i)
		}
	}

	s.Equal(numConcurrentReqs, successCount, "All concurrent requests should succeed")
}

func (s *SubscriptionServiceIntegrationSuite) TearDownTest() {
	// Reset mock expectations between tests
	s.setupMocks()
}
