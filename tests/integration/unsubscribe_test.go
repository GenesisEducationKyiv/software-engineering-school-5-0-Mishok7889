package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"weatherapi.app/internal/adapters/database"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestUnsubscribe_Success() {
	_ = helpers.ClearEmails()

	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Contains(response["message"], "Unsubscribed successfully")

	var deletedSubscription database.SubscriptionModel
	err = s.db.First(&deletedSubscription, subscription.ID).Error
	s.Error(err)

	var deletedToken database.TokenModel
	err = s.db.First(&deletedToken, token.ID).Error
	s.Error(err)

	// Wait for unsubscribe email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("test@example.com", "unsubscribed")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("test@example.com", "unsubscribed")
}

func (s *IntegrationTestSuite) TestUnsubscribe_InvalidToken() {
	req := httptest.NewRequest("GET", "/api/unsubscribe/invalid-token", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Contains(errorResponse.Error, "invalid unsubscribe token")
}

func (s *IntegrationTestSuite) TestUnsubscribe_ExpiredToken() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", -1*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Contains(errorResponse.Error, "invalid unsubscribe token")
}

func (s *IntegrationTestSuite) TestUnsubscribe_WrongTokenType() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "confirmation", 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Contains(errorResponse.Error, "invalid token type")
}

func (s *IntegrationTestSuite) TestUnsubscribe_SubscriptionNotFound() {
	// Create a subscription first, then delete it to create an orphaned token scenario
	subscription := s.CreateTestSubscription("orphan@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	// Now delete the subscription, leaving the token orphaned
	err := s.db.Delete(&database.SubscriptionModel{}, subscription.ID).Error
	s.NoError(err)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusNotFound, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Contains(errorResponse.Error, "subscription not found")
}

func (s *IntegrationTestSuite) TestUnsubscribe_DifferentCities() {
	_ = helpers.ClearEmails()

	cities := []string{"London", "Paris", "Berlin"}

	for _, city := range cities {
		subscription := s.CreateTestSubscription("test@example.com", city, "daily", true)
		token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

		req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)

		var deletedSubscription database.SubscriptionModel
		err := s.db.First(&deletedSubscription, subscription.ID).Error
		s.Error(err)

		// Wait for unsubscribe email to be sent
		s.Require().Eventually(func() bool {
			return helpers.CheckEmailSent("test@example.com", "unsubscribed")
		}, 5*time.Second, 200*time.Millisecond)

		s.AssertEmailSent("test@example.com", "unsubscribed")

		_ = helpers.ClearEmails()
	}
}

func (s *IntegrationTestSuite) TestUnsubscribe_HourlyFrequency() {
	_ = helpers.ClearEmails()

	subscription := s.CreateTestSubscription("test@example.com", "London", "hourly", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var deletedSubscription database.SubscriptionModel
	err := s.db.First(&deletedSubscription, subscription.ID).Error
	s.Error(err)

	// Wait for unsubscribe email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("test@example.com", "unsubscribed")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("test@example.com", "unsubscribed")
}

func (s *IntegrationTestSuite) TestUnsubscribe_TokenValidForLongTime() {
	_ = helpers.ClearEmails()

	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	s.True(token.ExpiresAt.After(time.Now().Add(300 * 24 * time.Hour)))

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	// Wait for unsubscribe email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("test@example.com", "unsubscribed")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("test@example.com", "unsubscribed")
}
