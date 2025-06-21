package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"weatherapi.app/models"
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

	var deletedSubscription models.Subscription
	err = s.db.First(&deletedSubscription, subscription.ID).Error
	s.Error(err)

	var deletedToken models.Token
	err = s.db.First(&deletedToken, token.ID).Error
	s.Error(err)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", "unsubscribed from weather updates")
}

func (s *IntegrationTestSuite) TestUnsubscribe_InvalidToken() {
	req := httptest.NewRequest("GET", "/api/unsubscribe/invalid-token", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("token not found or expired", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestUnsubscribe_ExpiredToken() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", -1*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("token not found or expired", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestUnsubscribe_WrongTokenType() {
	subscription := s.CreateTestSubscription("test@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "confirmation", 24*time.Hour)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("invalid token type", errorResponse.Error)
}

func (s *IntegrationTestSuite) TestUnsubscribe_SubscriptionNotFound() {
	// Create a subscription first, then delete it to create an orphaned token scenario
	subscription := s.CreateTestSubscription("orphan@example.com", "London", "daily", true)
	token := s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	// Now delete the subscription, leaving the token orphaned
	err := s.db.Delete(&models.Subscription{}, subscription.ID).Error
	s.NoError(err)

	req := httptest.NewRequest("GET", "/api/unsubscribe/"+token.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusInternalServerError, w.Code)

	var errorResponse models.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("Internal server error", errorResponse.Error)
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

		var deletedSubscription models.Subscription
		err := s.db.First(&deletedSubscription, subscription.ID).Error
		s.Error(err)

		time.Sleep(1 * time.Second)
		s.AssertEmailSent("test@example.com", "unsubscribed from weather updates")

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

	var deletedSubscription models.Subscription
	err := s.db.First(&deletedSubscription, subscription.ID).Error
	s.Error(err)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", "unsubscribed from weather updates")
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

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", "unsubscribed from weather updates")
}
