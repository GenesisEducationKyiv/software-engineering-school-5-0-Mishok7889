package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"weatherapi.app/models"
	"weatherapi.app/tests/integration/helpers"
)

const (
	// Subscribe test constants
	subscriptionSuccessful      = "Subscription successful"
	confirmEmailSubject         = "Confirm your weather subscription"
	emailAlreadySubscribedError = "email already subscribed"
	invalidRequestFormatError   = "invalid request format"
)

func (s *IntegrationTestSuite) TestSubscribe_Success() {
	_ = helpers.ClearEmails()

	formData := "email=test@example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Contains(response["message"], subscriptionSuccessful)

	subscription := s.AssertSubscriptionExists("test@example.com", "London")
	s.Equal("daily", subscription.Frequency)
	s.False(subscription.Confirmed)

	s.AssertTokenExists(subscription.ID, tokenTypeConfirmation)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", confirmEmailSubject)
}

func (s *IntegrationTestSuite) TestSubscribe_UpdateExisting() {
	_ = helpers.ClearEmails()

	existingSubscription := s.CreateTestSubscription("test@example.com", "London", "hourly", false)

	formData := "email=test@example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	var updatedSubscription models.Subscription
	err := s.db.First(&updatedSubscription, existingSubscription.ID).Error
	s.NoError(err)
	s.Equal("daily", updatedSubscription.Frequency)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", confirmEmailSubject)
}

func (s *IntegrationTestSuite) TestSubscribe_AlreadyConfirmed() {
	s.CreateTestSubscription("test@example.com", "London", "daily", true)

	formData := "email=test@example.com&city=London&frequency=hourly"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusConflict, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(emailAlreadySubscribedError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestSubscribe_MissingEmail() {
	formData := "city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(invalidRequestFormatError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestSubscribe_InvalidEmail() {
	formData := "email=invalid-email&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(invalidRequestFormatError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestSubscribe_MissingCity() {
	formData := "email=test@example.com&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(invalidRequestFormatError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestSubscribe_InvalidFrequency() {
	formData := "email=test@example.com&city=London&frequency=weekly"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal(invalidRequestFormatError, errorResponse.Error)
}

func (s *IntegrationTestSuite) TestSubscribe_JSONFormat() {
	_ = helpers.ClearEmails()

	jsonData := `{"email":"test@example.com","city":"London","frequency":"daily"}`
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	subscription := s.AssertSubscriptionExists("test@example.com", "London")
	s.Equal("daily", subscription.Frequency)
	s.False(subscription.Confirmed)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("test@example.com", confirmEmailSubject)
}
