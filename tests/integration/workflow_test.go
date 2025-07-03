package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"weatherapi.app/models"
	"weatherapi.app/providers"
	"weatherapi.app/repository"
	"weatherapi.app/service"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestCompleteSubscriptionWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	formData := "email=workflow@example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	subscription := s.AssertSubscriptionExists("workflow@example.com", "London")
	s.False(subscription.Confirmed)

	confirmToken := s.AssertTokenExists(subscription.ID, "confirmation")

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("workflow@example.com", "Confirm your weather subscription")

	req = httptest.NewRequest("GET", "/api/confirm/"+confirmToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var confirmedSubscription models.Subscription
	err = s.db.First(&confirmedSubscription, subscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)

	unsubscribeToken := s.AssertTokenExists(subscription.ID, "unsubscribe")

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("workflow@example.com", "Welcome to Weather Updates")

	req = httptest.NewRequest("GET", "/api/unsubscribe/"+unsubscribeToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var deletedSubscription models.Subscription
	err = s.db.First(&deletedSubscription, subscription.ID).Error
	s.Error(err)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("workflow@example.com", "unsubscribed from weather updates")
}

func (s *IntegrationTestSuite) TestMultipleSubscriptionsWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscriptions := []struct {
		email     string
		city      string
		frequency string
	}{
		{"user1@example.com", "London", "daily"},
		{"user2@example.com", "Paris", "hourly"},
		{"user3@example.com", "Berlin", "daily"},
	}

	for _, sub := range subscriptions {
		formData := "email=" + sub.email + "&city=" + sub.city + "&frequency=" + sub.frequency
		req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)

		subscription := s.AssertSubscriptionExists(sub.email, sub.city)
		s.Equal(sub.frequency, subscription.Frequency)
		s.False(subscription.Confirmed)

		confirmToken := s.AssertTokenExists(subscription.ID, "confirmation")

		req = httptest.NewRequest("GET", "/api/confirm/"+confirmToken.Token, nil)
		w = httptest.NewRecorder()

		s.router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
	}

	time.Sleep(3 * time.Second)

	for _, sub := range subscriptions {
		s.AssertEmailSent(sub.email, "Confirm your weather subscription")
		s.AssertEmailSent(sub.email, "Welcome to Weather Updates")
	}
}

func (s *IntegrationTestSuite) TestWeatherUpdateWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscription1 := s.CreateTestSubscription("daily@example.com", "London", "daily", true)
	subscription2 := s.CreateTestSubscription("hourly@example.com", "Paris", "hourly", true)

	s.CreateTestToken(subscription1.ID, "unsubscribe", 365*24*time.Hour)
	s.CreateTestToken(subscription2.ID, "unsubscribe", 365*24*time.Hour)

	subscriptionService := s.getSubscriptionService()

	err = subscriptionService.SendWeatherUpdate("daily")
	s.NoError(err)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("daily@example.com", "Weather Update for London")

	err = subscriptionService.SendWeatherUpdate("hourly")
	s.NoError(err)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("hourly@example.com", "Weather Update for Paris")
}

func (s *IntegrationTestSuite) TestSubscriptionUpdateWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	formData := "email=update@example.com&city=London&frequency=hourly"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	subscription := s.AssertSubscriptionExists("update@example.com", "London")
	s.Equal("hourly", subscription.Frequency)

	formData = "email=update@example.com&city=London&frequency=daily"
	req = httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var updatedSubscription models.Subscription
	err = s.db.First(&updatedSubscription, subscription.ID).Error
	s.NoError(err)
	s.Equal("daily", updatedSubscription.Frequency)

	time.Sleep(2 * time.Second)
	s.AssertEmailSent("update@example.com", "Confirm your weather subscription")
}

func (s *IntegrationTestSuite) TestErrorRecoveryWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	subscription := s.CreateTestSubscription("error@example.com", "London", "daily", false)
	expiredToken := s.CreateTestToken(subscription.ID, "confirmation", -1*time.Hour)

	req := httptest.NewRequest("GET", "/api/confirm/"+expiredToken.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse models.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Equal("token not found or expired", errorResponse.Error)

	formData := "email=error@example.com&city=London&frequency=daily"
	req = httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	newToken := s.AssertTokenExists(subscription.ID, "confirmation")
	s.NotEqual(expiredToken.Token, newToken.Token)

	req = httptest.NewRequest("GET", "/api/confirm/"+newToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var confirmedSubscription models.Subscription
	err = s.db.First(&confirmedSubscription, subscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)
}

func (s *IntegrationTestSuite) getSubscriptionService() service.SubscriptionServiceInterface {
	// Create provider manager instead of individual provider
	providerConfig := &providers.ProviderConfiguration{
		WeatherAPIKey:     s.config.Weather.APIKey,
		WeatherAPIBaseURL: s.config.Weather.BaseURL, // Use mock API URL
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false, // Disable cache for testing
		EnableLogging:     false, // Disable logging for testing
		ProviderOrder:     []string{"weatherapi"},
	}

	providerManager, err := providers.NewProviderManager(providerConfig, nil)
	s.Require().NoError(err)

	emailProvider := providers.NewSMTPEmailProvider(&s.config.Email)

	weatherService := service.NewWeatherService(providerManager)
	emailService := service.NewEmailService(emailProvider)

	subscriptionRepo := repository.NewSubscriptionRepository(s.db)
	tokenRepo := repository.NewTokenRepository(s.db)

	return service.NewSubscriptionService(
		s.db,
		subscriptionRepo,
		tokenRepo,
		emailService,
		weatherService,
		s.config,
	)
}
