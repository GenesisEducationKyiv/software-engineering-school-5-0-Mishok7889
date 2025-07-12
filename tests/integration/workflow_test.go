package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"weatherapi.app/internal/adapters/database"
	"weatherapi.app/internal/core/notification"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestCompleteSubscriptionWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Step 1: Subscribe
	formData := "email=workflow@example.com&city=London&frequency=daily"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	testSubscription := s.AssertSubscriptionExists("workflow@example.com", "London")
	s.False(testSubscription.Confirmed)

	confirmToken := s.AssertTokenExists(testSubscription.ID, "confirmation")

	// Wait for confirmation email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("workflow@example.com", "Confirm your weather subscription")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("workflow@example.com", "Confirm your weather subscription")

	// Step 2: Confirm subscription
	req = httptest.NewRequest("GET", "/api/confirm/"+confirmToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var confirmedSubscription database.SubscriptionModel
	err = s.db.First(&confirmedSubscription, testSubscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)

	unsubscribeToken := s.AssertTokenExists(testSubscription.ID, "unsubscribe")

	// Wait for welcome email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("workflow@example.com", "Welcome to Weather Updates")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("workflow@example.com", "Welcome to Weather Updates")

	// Step 3: Unsubscribe
	req = httptest.NewRequest("GET", "/api/unsubscribe/"+unsubscribeToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var deletedSubscription database.SubscriptionModel
	err = s.db.First(&deletedSubscription, testSubscription.ID).Error
	s.Error(err)

	// Wait for unsubscribe email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("workflow@example.com", "unsubscribed")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("workflow@example.com", "unsubscribed")
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

	// Subscribe all users
	for _, sub := range subscriptions {
		formData := "email=" + sub.email + "&city=" + sub.city + "&frequency=" + sub.frequency
		req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)

		testSubscription := s.AssertSubscriptionExists(sub.email, sub.city)
		s.Equal(sub.frequency, testSubscription.Frequency)
		s.False(testSubscription.Confirmed)
	}

	// Confirm all subscriptions
	for _, sub := range subscriptions {
		testSubscription := s.AssertSubscriptionExists(sub.email, sub.city)
		confirmToken := s.AssertTokenExists(testSubscription.ID, "confirmation")

		req := httptest.NewRequest("GET", "/api/confirm/"+confirmToken.Token, nil)
		w := httptest.NewRecorder()

		s.router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
	}

	// Wait for all emails to be sent
	s.Require().Eventually(func() bool {
		for _, sub := range subscriptions {
			if !helpers.CheckEmailSent(sub.email, "Confirm your weather subscription") {
				return false
			}
			if !helpers.CheckEmailSent(sub.email, "Welcome to Weather Updates") {
				return false
			}
		}
		return true
	}, 10*time.Second, 500*time.Millisecond)

	// Verify all emails were sent
	for _, sub := range subscriptions {
		s.AssertEmailSent(sub.email, "Confirm your weather subscription")
		s.AssertEmailSent(sub.email, "Welcome to Weather Updates")
	}
}

func (s *IntegrationTestSuite) TestWeatherUpdateWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Create confirmed subscriptions
	subscription1 := s.CreateTestSubscription("daily@example.com", "London", "daily", true)
	subscription2 := s.CreateTestSubscription("hourly@example.com", "Paris", "hourly", true)

	s.CreateTestToken(subscription1.ID, "unsubscribe", 365*24*time.Hour)
	s.CreateTestToken(subscription2.ID, "unsubscribe", 365*24*time.Hour)

	// Send weather updates using the notification use case
	ctx := context.Background()
	notificationUseCase := s.application.GetNotificationUseCase()
	s.Require().NotNil(notificationUseCase)

	// Send daily updates
	dailyParams := notification.SendWeatherUpdateParams{
		Frequency: subscription.FrequencyDaily,
	}
	err = notificationUseCase.SendWeatherUpdates(ctx, dailyParams)
	s.NoError(err)

	// Wait for daily weather update email
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("daily@example.com", "Weather Update for London")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("daily@example.com", "Weather Update for London")

	// Clear emails and send hourly updates
	_ = helpers.ClearEmails()
	hourlyParams := notification.SendWeatherUpdateParams{
		Frequency: subscription.FrequencyHourly,
	}
	err = notificationUseCase.SendWeatherUpdates(ctx, hourlyParams)
	s.NoError(err)

	// Wait for hourly weather update email
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("hourly@example.com", "Weather Update for Paris")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("hourly@example.com", "Weather Update for Paris")
}

func (s *IntegrationTestSuite) TestSubscriptionUpdateWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Initial subscription with hourly frequency
	formData := "email=update@example.com&city=London&frequency=hourly"
	req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	testSubscription := s.AssertSubscriptionExists("update@example.com", "London")
	s.Equal("hourly", testSubscription.Frequency)

	// Update subscription to daily frequency
	formData = "email=update@example.com&city=London&frequency=daily"
	req = httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var updatedSubscription database.SubscriptionModel
	err = s.db.First(&updatedSubscription, testSubscription.ID).Error
	s.NoError(err)
	s.Equal("daily", updatedSubscription.Frequency)

	// Wait for confirmation email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("update@example.com", "Confirm your weather subscription")
	}, 5*time.Second, 200*time.Millisecond)

	s.AssertEmailSent("update@example.com", "Confirm your weather subscription")
}

func (s *IntegrationTestSuite) TestErrorRecoveryWorkflow() {
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Create subscription with expired token
	testSubscription := s.CreateTestSubscription("error@example.com", "London", "daily", false)
	expiredToken := s.CreateTestToken(testSubscription.ID, "confirmation", -1*time.Hour)

	// Try to confirm with expired token
	req := httptest.NewRequest("GET", "/api/confirm/"+expiredToken.Token, nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusBadRequest, w.Code)

	var errorResponse ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	s.NoError(err)
	s.Contains(errorResponse.Error, "expired")

	// Resubscribe to get new token
	formData := "email=error@example.com&city=London&frequency=daily"
	req = httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	// Get new token
	newToken := s.AssertTokenExists(testSubscription.ID, "confirmation")
	s.NotEqual(expiredToken.Token, newToken.Token)

	// Confirm with new token
	req = httptest.NewRequest("GET", "/api/confirm/"+newToken.Token, nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var confirmedSubscription database.SubscriptionModel
	err = s.db.First(&confirmedSubscription, testSubscription.ID).Error
	s.NoError(err)
	s.True(confirmedSubscription.Confirmed)
}

func (s *IntegrationTestSuite) TestConcurrentSubscriptionWorkflow() {
	// Test concurrent subscriptions to ensure thread safety
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	emails := []string{
		"concurrent1@example.com",
		"concurrent2@example.com",
		"concurrent3@example.com",
		"concurrent4@example.com",
		"concurrent5@example.com",
	}

	// Subscribe all users concurrently
	results := make(chan error, len(emails))

	for _, email := range emails {
		go func(email string) {
			formData := "email=" + email + "&city=London&frequency=daily"
			req := httptest.NewRequest("POST", "/api/subscribe", strings.NewReader(formData))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				results <- fmt.Errorf("subscription failed for %s: status %d", email, w.Code)
				return
			}
			results <- nil
		}(email)
	}

	// Wait for all subscriptions to complete
	for i := 0; i < len(emails); i++ {
		err := <-results
		s.NoError(err)
	}

	// Verify all subscriptions were created
	for _, email := range emails {
		subscription := s.AssertSubscriptionExists(email, "London")
		s.Equal("daily", subscription.Frequency)
		s.False(subscription.Confirmed)
	}

	// Wait for all confirmation emails to be sent
	s.Require().Eventually(func() bool {
		for _, email := range emails {
			if !helpers.CheckEmailSent(email, "Confirm your weather subscription") {
				return false
			}
		}
		return true
	}, 10*time.Second, 500*time.Millisecond)

	// Verify confirmation emails were sent
	for _, email := range emails {
		s.AssertEmailSent(email, "Confirm your weather subscription")
	}
}
