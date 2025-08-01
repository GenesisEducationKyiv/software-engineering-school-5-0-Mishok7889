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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/mocks"
	"weatherapi.app/internal/ports"
	subscriptionapi "weatherapi.app/internal/services/subscription/adapters/api"
)

const (
	testTimeout       = 30 * time.Second
	numConcurrentReqs = 10
	numPerfRequests   = 100
	maxAvgLatency     = 50 * time.Millisecond

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
	mockSubscriptionRepo *mocks.SubscriptionRepository
	mockTokenRepo        *mocks.TokenRepository
	mockTokenGenerator   *mocks.TokenGenerator
	mockEmailProvider    *mocks.EmailProvider
	mockEmailBuilder     *mocks.EmailBuilder
	mockConfig           *mocks.ConfigProvider
	mockLogger           *mocks.Logger
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
	s.mockEmailProvider = mocks.NewEmailProvider(s.T())
	s.mockEmailBuilder = mocks.NewEmailBuilder(s.T())
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
		SubscriptionRepo: s.mockSubscriptionRepo,
		TokenRepo:        s.mockTokenRepo,
		TokenGenerator:   s.mockTokenGenerator,
		EmailProvider:    s.mockEmailProvider,
		EmailBuilder:     s.mockEmailBuilder,
		Config:           s.mockConfig,
		Logger:           s.mockLogger,
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
func (s *SubscriptionServiceIntegrationSuite) TestSubscribe() {
	testCases := []struct {
		name               string
		request            map[string]interface{}
		expectError        bool
		expectedStatusCode int
		errorContains      string
		setupMocks         func()
	}{
		{
			name: "ValidSubscription_Success",
			request: map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": testFrequency,
			},
			expectError:        false,
			expectedStatusCode: http.StatusOK,
			setupMocks: func() {
				// Mock subscription check (not exists)
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()

				// Mock token generation
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()

				// Mock token save
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
					return tokenData.Value == testToken &&
						tokenData.Type == "confirmation"
				})).Return(nil).Once()

				// Mock subscription creation
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
					return sub.Email == testEmail &&
						sub.City == testCity &&
						sub.Frequency == subscription.FrequencyHourly.String()
				})).Run(func(ctx context.Context, sub *ports.SubscriptionData) {
					// Simulate database setting the ID (like auto-increment)
					sub.ID = testSubscriptionID
				}).Return(nil).Once()

				// Mock email sending
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{
						To:      testEmail,
						Subject: "Confirm subscription",
						Body:    "Please confirm",
						Format:  ports.FormatText,
					}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "InvalidEmail_ValidationError",
			request: map[string]interface{}{
				"email":     "invalid-email",
				"city":      testCity,
				"frequency": testFrequency,
			},
			expectError:        true,
			expectedStatusCode: http.StatusBadRequest,
			errorContains:      "invalid request format",
			setupMocks:         func() {}, // No mocks needed for validation errors
		},
		{
			name: "MissingCity_ValidationError",
			request: map[string]interface{}{
				"email":     testEmail,
				"frequency": testFrequency,
			},
			expectError:        true,
			expectedStatusCode: http.StatusBadRequest,
			errorContains:      "invalid request format",
			setupMocks:         func() {},
		},
		{
			name: "InvalidFrequency_ValidationError",
			request: map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": "invalid",
			},
			expectError:        true,
			expectedStatusCode: http.StatusBadRequest,
			errorContains:      "invalid request format",
			setupMocks:         func() {},
		},
		{
			name: "SubscriptionAlreadyExists_ConflictError",
			request: map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": testFrequency,
			},
			expectError:        true,
			expectedStatusCode: http.StatusConflict,
			errorContains:      "already subscribed",
			setupMocks: func() {
				existingSubscription := &ports.SubscriptionData{
					ID:        testSubscriptionID,
					Email:     testEmail,
					City:      testCity,
					Frequency: subscription.FrequencyHourly.String(),
					Confirmed: true, // Make it confirmed so it returns AlreadyExistsError
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
					Return(existingSubscription, nil).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", tc.request)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(s.T(), tc.expectedStatusCode, resp.StatusCode)

			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			s.Require().NoError(err)

			if tc.expectError {
				errorMsg, exists := responseBody["error"]
				s.Require().True(exists, "Expected error field in response")
				if tc.errorContains != "" {
					assert.Contains(s.T(), errorMsg.(string), tc.errorContains)
				}
			} else {
				message, exists := responseBody["message"]
				s.Require().True(exists, "Expected message field in response")
				assert.Contains(s.T(), message.(string), "successfully")
			}
		})
	}
}
func (s *SubscriptionServiceIntegrationSuite) TestConfirmSubscription() {
	testCases := []struct {
		name               string
		token              string
		expectError        bool
		expectedStatusCode int
		errorContains      string
		setupMocks         func()
	}{
		{
			name:               "ValidToken_Success",
			token:              testToken,
			expectError:        false,
			expectedStatusCode: http.StatusOK,
			setupMocks: func() {
				// Mock token validation
				tokenData := &ports.TokenData{
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(time.Hour),
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()

				// Mock subscription lookup - returns unconfirmed subscription
				subscriptionData := &ports.SubscriptionData{
					ID:        testSubscriptionID,
					Email:     testEmail,
					City:      testCity,
					Frequency: subscription.FrequencyHourly.String(),
					Confirmed: false,
				}
				s.mockSubscriptionRepo.EXPECT().FindByID(mock.Anything, testSubscriptionID).
					Return(subscriptionData, nil).Once()

				// Mock subscription update - use matcher to ignore timestamps
				s.mockSubscriptionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
					return sub.ID == testSubscriptionID &&
						sub.Email == testEmail &&
						sub.City == testCity &&
						sub.Frequency == subscription.FrequencyHourly.String() &&
						sub.Confirmed == true
				})).Return(nil).Once()

				// Mock token deletion
				s.mockTokenRepo.EXPECT().Delete(mock.Anything, tokenData).Return(nil).Once()

				// Mock unsubscribe token generation for welcome email
				s.mockTokenGenerator.EXPECT().GenerateToken().Return("unsubscribe-token-123").Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
					return tokenData.Value == "unsubscribe-token-123" &&
						tokenData.Type == "unsubscribe" &&
						tokenData.SubscriptionID == testSubscriptionID
				})).Return(nil).Once()

				// Mock welcome email sending
				s.mockEmailBuilder.EXPECT().BuildWelcomeEmail(testCity, testFrequency, "unsubscribe-token-123").
					Return(ports.EmailParams{
						To:      testEmail,
						Subject: "Welcome",
						Body:    "Welcome to weather updates",
						Format:  ports.FormatText,
					}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:               "InvalidToken_NotFound",
			token:              "invalid-token",
			expectError:        true,
			expectedStatusCode: http.StatusNotFound,
			errorContains:      "not found",
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, "invalid-token").
					Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
		{
			name:               "ExpiredToken_ValidationError",
			token:              testToken,
			expectError:        true,
			expectedStatusCode: http.StatusBadRequest,
			errorContains:      "expired",
			setupMocks: func() {
				expiredTokenData := &ports.TokenData{
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "confirmation",
					ExpiresAt:      time.Now().Add(-time.Hour), // Expired
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).
					Return(expiredTokenData, nil).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			path := fmt.Sprintf("/api/v1/confirm/%s", tc.token)
			resp, err := s.makeRequest("GET", path, nil)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(s.T(), tc.expectedStatusCode, resp.StatusCode)

			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			s.Require().NoError(err)

			if tc.expectError {
				errorMsg, exists := responseBody["error"]
				s.Require().True(exists, "Expected error field in response")
				if tc.errorContains != "" {
					assert.Contains(s.T(), errorMsg.(string), tc.errorContains)
				}
			} else {
				message, exists := responseBody["message"]
				s.Require().True(exists, "Expected message field in response")
				assert.Contains(s.T(), message.(string), "confirmed")
			}
		})
	}
}
func (s *SubscriptionServiceIntegrationSuite) TestUnsubscribe() {
	testCases := []struct {
		name               string
		token              string
		expectError        bool
		expectedStatusCode int
		errorContains      string
		setupMocks         func()
	}{
		{
			name:               "ValidToken_Success",
			token:              testToken,
			expectError:        false,
			expectedStatusCode: http.StatusOK,
			setupMocks: func() {
				// Mock token validation
				tokenData := &ports.TokenData{
					Value:          testToken,
					SubscriptionID: testSubscriptionID,
					Type:           "unsubscribe",
					ExpiresAt:      time.Now().Add(time.Hour),
					CreatedAt:      time.Now(),
				}
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, testToken).Return(tokenData, nil).Once()

				// Mock subscription deletion
				subscriptionData := &ports.SubscriptionData{
					ID:        testSubscriptionID,
					Email:     testEmail,
					City:      testCity,
					Frequency: subscription.FrequencyHourly.String(),
					Confirmed: true,
				}
				s.mockSubscriptionRepo.EXPECT().FindByID(mock.Anything, testSubscriptionID).
					Return(subscriptionData, nil).Once()
				s.mockSubscriptionRepo.EXPECT().Delete(mock.Anything, subscriptionData).Return(nil).Once()

				// Mock token deletion
				s.mockTokenRepo.EXPECT().Delete(mock.Anything, tokenData).Return(nil).Once()

				// Mock unsubscribe confirmation email
				unsubscribeEmailParams := ports.EmailParams{
					Subject: "Unsubscribe Confirmation",
					Body:    "You have been unsubscribed from weather updates",
					Format:  ports.FormatHTML,
				}
				s.mockEmailBuilder.EXPECT().BuildUnsubscribeEmail(testCity).Return(unsubscribeEmailParams, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
					return params.To == testEmail && params.Subject == unsubscribeEmailParams.Subject
				})).Return(nil).Once()
			},
		},
		{
			name:               "InvalidToken_NotFound",
			token:              "invalid-token",
			expectError:        true,
			expectedStatusCode: http.StatusNotFound,
			errorContains:      "not found",
			setupMocks: func() {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, "invalid-token").
					Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			path := fmt.Sprintf("/api/v1/unsubscribe/%s", tc.token)
			resp, err := s.makeRequest("GET", path, nil)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(s.T(), tc.expectedStatusCode, resp.StatusCode)

			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			s.Require().NoError(err)

			if tc.expectError {
				errorMsg, exists := responseBody["error"]
				s.Require().True(exists, "Expected error field in response")
				if tc.errorContains != "" {
					assert.Contains(s.T(), errorMsg.(string), tc.errorContains)
				}
			} else {
				message, exists := responseBody["message"]
				s.Require().True(exists, "Expected message field in response")
				assert.Contains(s.T(), message.(string), "unsubscribed")
			}
		})
	}
}
func (s *SubscriptionServiceIntegrationSuite) TestHealthCheck() {
	resp, err := s.makeRequest("GET", "/health", nil)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	s.Require().NoError(err)

	assert.Equal(s.T(), "healthy", responseBody["status"])
	assert.Equal(s.T(), "subscription-service", responseBody["service"])
}

func (s *SubscriptionServiceIntegrationSuite) TestSubscriptionLifecycleWorkflow() {
	// Setup mocks for complete workflow
	confirmationToken := "confirmation-token-123"
	unsubscribeToken := "unsubscribe-token-456"

	// Step 1: Subscribe - Setup mocks
	s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
		Return(nil, ports.NewNotFoundError("subscription not found")).Once()
	s.mockTokenGenerator.EXPECT().GenerateToken().Return(confirmationToken).Once()
	s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
		return tokenData.Value == confirmationToken && tokenData.Type == "confirmation"
	})).Return(nil).Once()
	s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
		return sub.Email == testEmail && sub.City == testCity
	})).RunAndReturn(func(ctx context.Context, sub *ports.SubscriptionData) error {
		sub.ID = testSubscriptionID // Simulate database assigning an ID
		return nil
	}).Once()
	s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
		Return(ports.EmailParams{To: testEmail, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
	s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()

	// Step 1: Subscribe
	subscribeReq := map[string]interface{}{
		"email":     testEmail,
		"city":      testCity,
		"frequency": testFrequency,
	}
	resp, err := s.makeRequest("POST", "/api/v1/subscriptions", subscribeReq)
	s.Require().NoError(err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	// Step 2: Confirm subscription - Setup mocks
	confirmTokenData := &ports.TokenData{
		Value:          confirmationToken,
		SubscriptionID: testSubscriptionID,
		Type:           "confirmation",
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, confirmationToken).Return(confirmTokenData, nil).Once()

	// Create fresh subscription data for confirmation (must be unconfirmed)
	unconfirmedSubscriptionData := &ports.SubscriptionData{
		ID:        testSubscriptionID,
		Email:     testEmail,
		City:      testCity,
		Frequency: subscription.FrequencyHourly.String(),
		Confirmed: false,
	}
	s.mockSubscriptionRepo.EXPECT().FindByID(mock.Anything, testSubscriptionID).Return(unconfirmedSubscriptionData, nil).Once()

	// Expect subscription update to confirmed
	s.mockSubscriptionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
		return sub.ID == testSubscriptionID && sub.Confirmed == true
	})).Return(nil).Once()

	// Expect confirmation token deletion
	s.mockTokenRepo.EXPECT().Delete(mock.Anything, confirmTokenData).Return(nil).Once()

	// Expect unsubscribe token generation for welcome email
	s.mockTokenGenerator.EXPECT().GenerateToken().Return(unsubscribeToken).Once()
	s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
		return tokenData.Value == unsubscribeToken &&
			tokenData.Type == "unsubscribe" &&
			tokenData.SubscriptionID == testSubscriptionID
	})).Return(nil).Once()

	// Expect welcome email sending
	s.mockEmailBuilder.EXPECT().BuildWelcomeEmail(testCity, testFrequency, unsubscribeToken).
		Return(ports.EmailParams{
			To:      testEmail,
			Subject: "Welcome",
			Body:    "Welcome to weather updates",
			Format:  ports.FormatText,
		}, nil).Once()
	s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
		return params.To == testEmail && params.Subject == "Welcome"
	})).Return(nil).Once()

	// Step 2: Confirm subscription
	confirmPath := fmt.Sprintf("/api/v1/confirm/%s", confirmationToken)
	resp, err = s.makeRequest("GET", confirmPath, nil)
	s.Require().NoError(err)

	// Add debugging for the confirmation response
	var confirmBody map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&confirmBody)
	if resp.StatusCode != http.StatusOK {
		s.T().Logf("[DEBUG] Confirmation failed - Status: %d, Body: %+v", resp.StatusCode, confirmBody)
	}
	_ = resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// Step 3: Unsubscribe - Setup mocks
	unsubscribeTokenData := &ports.TokenData{
		Value:          unsubscribeToken,
		SubscriptionID: testSubscriptionID,
		Type:           "unsubscribe",
		ExpiresAt:      time.Now().Add(time.Hour),
		CreatedAt:      time.Now(),
	}
	s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, unsubscribeToken).Return(unsubscribeTokenData, nil).Once()

	// Update subscription data to be confirmed for unsubscribe step
	confirmedSubscriptionData := &ports.SubscriptionData{
		ID:        testSubscriptionID,
		Email:     testEmail,
		City:      testCity,
		Frequency: subscription.FrequencyHourly.String(),
		Confirmed: true,
	}
	s.mockSubscriptionRepo.EXPECT().FindByID(mock.Anything, testSubscriptionID).Return(confirmedSubscriptionData, nil).Once()
	s.mockSubscriptionRepo.EXPECT().Delete(mock.Anything, confirmedSubscriptionData).Return(nil).Once()
	s.mockTokenRepo.EXPECT().Delete(mock.Anything, unsubscribeTokenData).Return(nil).Once()

	// Mock unsubscribe confirmation email
	unsubscribeEmailParams := ports.EmailParams{
		Subject: "Unsubscribe Confirmation",
		Body:    "You have been unsubscribed from weather updates",
		Format:  ports.FormatHTML,
	}
	s.mockEmailBuilder.EXPECT().BuildUnsubscribeEmail(testCity).Return(unsubscribeEmailParams, nil).Once()
	s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.MatchedBy(func(params ports.EmailParams) bool {
		return params.To == testEmail && params.Subject == unsubscribeEmailParams.Subject
	})).Return(nil).Once()

	// Step 3: Unsubscribe
	unsubscribePath := fmt.Sprintf("/api/v1/unsubscribe/%s", unsubscribeToken)
	resp, err = s.makeRequest("GET", unsubscribePath, nil)
	s.Require().NoError(err)

	// Add debugging for the unsubscribe response
	var unsubscribeBody map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&unsubscribeBody)
	if resp.StatusCode != http.StatusOK {
		s.T().Logf("[DEBUG] Unsubscribe failed - Status: %d, Body: %+v", resp.StatusCode, unsubscribeBody)
	}
	_ = resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}
func (s *SubscriptionServiceIntegrationSuite) TestConcurrentSubscriptions() {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Setup mocks for concurrent requests
	for i := 0; i < numConcurrentReqs; i++ {
		email := fmt.Sprintf("user%d@test.com", i)
		token := fmt.Sprintf("token_%d", i)

		s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, email, testCity).
			Return(nil, ports.NewNotFoundError("subscription not found")).Once()
		s.mockTokenGenerator.EXPECT().GenerateToken().Return(token).Once()
		s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(tokenData *ports.TokenData) bool {
			return tokenData.Value == token && tokenData.Type == "confirmation"
		})).Return(nil).Once()
		s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
			return sub.Email == email && sub.City == testCity
		})).RunAndReturn(func(ctx context.Context, sub *ports.SubscriptionData) error {
			sub.ID = uint(i + 1) // Simulate database assigning unique IDs
			return nil
		}).Once()
		s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
			Return(ports.EmailParams{To: email, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
		s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
	}

	results := make(chan error, numConcurrentReqs)

	for i := 0; i < numConcurrentReqs; i++ {
		go func(id int) {
			req := map[string]interface{}{
				"email":     fmt.Sprintf("user%d@test.com", id),
				"city":      testCity,
				"frequency": testFrequency,
			}
			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", req)
			if err != nil {
				results <- err
				return
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
				return
			}
			results <- nil
		}(i)
	}

	for i := 0; i < numConcurrentReqs; i++ {
		select {
		case err := <-results:
			s.Require().NoError(err)
		case <-ctx.Done():
			s.Fail("Concurrent requests timeout")
		}
	}
}

func (s *SubscriptionServiceIntegrationSuite) TestSubscriptionPerformance() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	// Setup mocks for performance test
	for i := 0; i < numPerfRequests; i++ {
		email := fmt.Sprintf("perf%d@test.com", i)
		token := fmt.Sprintf("perf_token_%d", i)

		s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, email, testCity).
			Return(nil, ports.NewNotFoundError("subscription not found")).Once()
		s.mockTokenGenerator.EXPECT().GenerateToken().Return(token).Once()
		s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
		s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, sub *ports.SubscriptionData) error {
				sub.ID = uint(i + 1) // Simulate database assigning unique IDs
				return nil
			}).Once()
		s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
			Return(ports.EmailParams{To: email, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
		s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
	}

	// Performance test loop
	start := time.Now()

	for i := 0; i < numPerfRequests; i++ {
		req := map[string]interface{}{
			"email":     fmt.Sprintf("perf%d@test.com", i),
			"city":      testCity,
			"frequency": testFrequency,
		}
		resp, err := s.makeRequest("POST", "/api/v1/subscriptions", req)
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	}

	duration := time.Since(start)
	avgDuration := duration / numPerfRequests

	s.T().Logf("Performance: %d subscriptions in %v (avg: %v per request)",
		numPerfRequests, duration, avgDuration)

	assert.True(s.T(), avgDuration < maxAvgLatency,
		"Average subscription request should be under 50ms")
}
func (s *SubscriptionServiceIntegrationSuite) TestEmailValidation() {
	testCases := []struct {
		name          string
		email         string
		expectError   bool
		errorContains string
	}{
		{
			name:        "ValidEmail_Standard",
			email:       "user@example.com",
			expectError: false,
		},
		{
			name:        "ValidEmail_WithDots",
			email:       "user.name@example.com",
			expectError: false,
		},
		{
			name:        "ValidEmail_WithNumbers",
			email:       "user123@example.com",
			expectError: false,
		},
		{
			name:          "InvalidEmail_NoAt",
			email:         "userexample.com",
			expectError:   true,
			errorContains: "invalid request format",
		},
		{
			name:          "InvalidEmail_NoDomain",
			email:         "user@",
			expectError:   true,
			errorContains: "invalid request format",
		},
		{
			name:          "InvalidEmail_NoLocal",
			email:         "@example.com",
			expectError:   true,
			errorContains: "invalid request format",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks only for valid emails
			if !tc.expectError {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, tc.email, testCity).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, sub *ports.SubscriptionData) error {
						sub.ID = testSubscriptionID // Simulate database assigning an ID
						return nil
					}).Once()
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{To: tc.email, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			}

			req := map[string]interface{}{
				"email":     tc.email,
				"city":      testCity,
				"frequency": testFrequency,
			}
			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", req)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			if tc.expectError {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
				var responseBody map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				s.Require().NoError(err)
				if tc.errorContains != "" {
					errorMsg := responseBody["error"].(string)
					assert.Contains(s.T(), errorMsg, tc.errorContains)
				}
			} else {
				assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func (s *SubscriptionServiceIntegrationSuite) TestFrequencyValidation() {
	testCases := []struct {
		name        string
		frequency   string
		expectError bool
	}{
		{
			name:        "ValidFrequency_Hourly",
			frequency:   "hourly",
			expectError: false,
		},
		{
			name:        "ValidFrequency_Daily",
			frequency:   "daily",
			expectError: false,
		},
		{
			name:        "InvalidFrequency_Weekly",
			frequency:   "weekly",
			expectError: true,
		},
		{
			name:        "InvalidFrequency_Empty",
			frequency:   "",
			expectError: true,
		},
		{
			name:        "InvalidFrequency_Invalid",
			frequency:   "invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks only for valid frequencies
			if !tc.expectError {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, sub *ports.SubscriptionData) error {
						sub.ID = testSubscriptionID // Simulate database assigning an ID
						return nil
					}).Once()
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{To: testEmail, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			}

			req := map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": tc.frequency,
			}
			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", req)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			if tc.expectError {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
			} else {
				assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func (s *SubscriptionServiceIntegrationSuite) TestErrorHandling() {
	testCases := []struct {
		name               string
		request            map[string]interface{}
		expectedStatusCode int
		setupMocks         func()
	}{
		{
			name: "DatabaseError_SubscriptionSave",
			request: map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": testFrequency,
			},
			expectedStatusCode: http.StatusInternalServerError,
			setupMocks: func() {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).
					Return(fmt.Errorf("database connection failed")).Once()
			},
		},
		{
			name: "EmailError_SendFailed",
			request: map[string]interface{}{
				"email":     testEmail,
				"city":      testCity,
				"frequency": testFrequency,
			},
			expectedStatusCode: http.StatusInternalServerError,
			setupMocks: func() {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, testCity).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(sub *ports.SubscriptionData) bool {
					return sub.Email == testEmail && sub.City == testCity
				})).RunAndReturn(func(ctx context.Context, sub *ports.SubscriptionData) error {
					sub.ID = testSubscriptionID // Simulate database assigning an ID
					return nil
				}).Once()
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{To: testEmail, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).
					Return(fmt.Errorf("SMTP server unavailable")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks()

			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", tc.request)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(s.T(), tc.expectedStatusCode, resp.StatusCode)

			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			s.Require().NoError(err)

			errorMsg, exists := responseBody["error"]
			s.Require().True(exists, "Expected error field in response")
			s.Require().NotEmpty(errorMsg, "Expected non-empty error message")
		})
	}
}
func (s *SubscriptionServiceIntegrationSuite) TestMalformedRequests() {
	testCases := []struct {
		name               string
		requestBody        string
		contentType        string
		expectedStatusCode int
	}{
		{
			name:               "InvalidJSON",
			requestBody:        `{"email": "test@example.com", "city": "London", "frequency":}`,
			contentType:        "application/json",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "EmptyBody",
			requestBody:        "",
			contentType:        "application/json",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "InvalidContentType",
			requestBody:        `email=test@example.com&city=London&frequency=hourly`,
			contentType:        "application/x-www-form-urlencoded",
			expectedStatusCode: http.StatusOK, // Gin should handle form data
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for successful case (form data)
			if tc.expectedStatusCode == http.StatusOK {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, "test@example.com", "London").
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, sub *ports.SubscriptionData) error {
						sub.ID = testSubscriptionID // Simulate database assigning an ID
						return nil
					}).Once()
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{To: "test@example.com", Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			}

			req := httptest.NewRequest("POST", "/api/v1/subscriptions", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", tc.contentType)

			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, req)

			assert.Equal(s.T(), tc.expectedStatusCode, recorder.Code)
		})
	}
}

func (s *SubscriptionServiceIntegrationSuite) TestCityValidation() {
	testCases := []struct {
		name          string
		city          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "ValidCity_Standard",
			city:        "London",
			expectError: false,
		},
		{
			name:        "ValidCity_WithSpaces",
			city:        "New York",
			expectError: false,
		},
		{
			name:        "ValidCity_WithHyphen",
			city:        "São Paulo",
			expectError: false,
		},
		{
			name:          "InvalidCity_Empty",
			city:          "",
			expectError:   true,
			errorContains: "invalid request format",
		},
		{
			name:          "InvalidCity_OnlySpaces",
			city:          "   ",
			expectError:   true,
			errorContains: "city is required",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks only for valid cities
			if !tc.expectError {
				s.mockSubscriptionRepo.EXPECT().FindByEmail(mock.Anything, testEmail, tc.city).
					Return(nil, ports.NewNotFoundError("subscription not found")).Once()
				s.mockTokenGenerator.EXPECT().GenerateToken().Return(testToken).Once()
				s.mockTokenRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
				s.mockSubscriptionRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
					func(ctx context.Context, sub *ports.SubscriptionData) error {
						sub.ID = testSubscriptionID // Simulate database assigning an ID
						return nil
					}).Once()
				s.mockEmailBuilder.EXPECT().BuildConfirmationEmail(mock.Anything, mock.Anything).
					Return(ports.EmailParams{To: testEmail, Subject: "Confirm", Body: "Please confirm", Format: ports.FormatText}, nil).Once()
				s.mockEmailProvider.EXPECT().SendEmail(mock.Anything, mock.Anything).Return(nil).Once()
			}

			req := map[string]interface{}{
				"email":     testEmail,
				"city":      tc.city,
				"frequency": testFrequency,
			}
			resp, err := s.makeRequest("POST", "/api/v1/subscriptions", req)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			if tc.expectError {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
				var responseBody map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&responseBody)
				s.Require().NoError(err)
				if tc.errorContains != "" {
					errorMsg := responseBody["error"].(string)
					assert.Contains(s.T(), errorMsg, tc.errorContains)
				}
			} else {
				assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func (s *SubscriptionServiceIntegrationSuite) TestTokenEdgeCases() {
	testCases := []struct {
		name               string
		token              string
		endpoint           string
		expectedStatusCode int
		setupMocks         func(token string)
	}{
		{
			name:               "VeryLongToken",
			token:              strings.Repeat("a", 500),
			endpoint:           "confirm",
			expectedStatusCode: http.StatusNotFound,
			setupMocks: func(token string) {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, token).
					Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
		{
			name:               "TokenWithSpecialChars",
			token:              "token-with_special.chars123",
			endpoint:           "confirm",
			expectedStatusCode: http.StatusNotFound,
			setupMocks: func(token string) {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, token).
					Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
		{
			name:               "TokenWithUnicode",
			token:              "token_αβγ_测试",
			endpoint:           "unsubscribe",
			expectedStatusCode: http.StatusNotFound,
			setupMocks: func(token string) {
				s.mockTokenRepo.EXPECT().FindByToken(mock.Anything, token).
					Return(nil, ports.NewNotFoundError("token not found")).Once()
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup mocks for this test case
			tc.setupMocks(tc.token)

			path := fmt.Sprintf("/api/v1/%s/%s", tc.endpoint, tc.token)
			resp, err := s.makeRequest("GET", path, nil)
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(s.T(), tc.expectedStatusCode, resp.StatusCode)
		})
	}
}
