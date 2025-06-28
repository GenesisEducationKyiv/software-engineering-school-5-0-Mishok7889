package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/stretchr/testify/require"
	"weatherapi.app/models"
	"weatherapi.app/tests/integration/helpers"
)

const (
	// Confirmation test constants
	subscriptionConfirmed = "Subscription confirmed"
	tokenTypeUnsubscribe  = "unsubscribe"
	tokenTypeConfirmation = "confirmation"
	welcomeEmailSubject   = "Welcome to Weather Updates"
	tokenNotFoundError    = "token not found or expired"
	invalidTokenTypeError = "invalid token type"
	subscriptionNotFoundError = "subscription not found"
)

func (s *IntegrationTestSuite) TestConfirmSubscription_Success() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", false)
	token := s.CreateTestToken(subscription.ID, tokenTypeConfirmation, 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Contains(response["message"], subscriptionConfirmed)

	var confirmedSubscription models.Subscription
	err = s.db.First(&confirmedSubscription, subscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)

	var deletedToken models.Token
	err = s.db.First(&deletedToken, token.ID).Error
	s.Error(err)

	s.AssertTokenExists(subscription.ID, tokenTypeUnsubscribe)

	require.Eventually(s.T(), func() bool {
		return helpers.CheckEmailSent("test@example.com", welcomeEmailSubject)
	}, 10*time.Second, 500*time.Millisecond)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_InvalidToken() {
	req := httptest.NewRequest("GET", "/api/confirm/invalid-token", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(tokenNotFoundError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_ExpiredToken() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", false)
	token := s.CreateTestToken(subscription.ID, tokenTypeConfirmation, -1*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(tokenNotFoundError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_WrongTokenType() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", false)
	token := s.CreateTestToken(subscription.ID, tokenTypeUnsubscribe, 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(invalidTokenTypeError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_SubscriptionNotFound() {
	// Create a subscription first, then delete it to create an orphaned token scenario
	subscription := s.CreateTestSubscription("orphan@example.com", "London", "daily", false)
	token := s.CreateTestToken(subscription.ID, tokenTypeConfirmation, 24*time.Hour)

	// Now delete the subscription, leaving the token orphaned
	err := s.db.Delete(&models.Subscription{}, subscription.ID).Error
	s.NoError(err)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)

	var errorResponse models.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(subscriptionNotFoundError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_AlreadyConfirmed() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, tokenTypeConfirmation, 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Contains(response["message"], subscriptionConfirmed)

	require.Eventually(s.T(), func() bool {
		return helpers.CheckEmailSent("test@example.com", welcomeEmailSubject)
	}, 10*time.Second, 500*time.Millisecond)
}

func (s *IntegrationTestSuite) TestConfirmSubscription_HourlyFrequency() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscription := s.CreateTestSubscription("test@example.com", "Paris", "hourly", false)
	token := s.CreateTestToken(subscription.ID, tokenTypeConfirmation, 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var confirmedSubscription models.Subscription
	err = s.db.First(&confirmedSubscription, subscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)
	s.Equal("hourly", confirmedSubscription.Frequency)

	require.Eventually(s.T(), func() bool {
		return helpers.CheckEmailSent("test@example.com", welcomeEmailSubject)
	}, 10*time.Second, 500*time.Millisecond)
}
