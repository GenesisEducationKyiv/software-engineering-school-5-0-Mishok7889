package integration

import (
	"time"

	"weatherapi.app/providers"
	"weatherapi.app/repository"
	"weatherapi.app/service"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestProviderManagerIntegration() {
	// Test that provider manager correctly handles edge cases

	// Test 1: Verify provider manager can be created with no providers
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Create a subscription service with no providers configured
	subscription := s.CreateTestSubscription("noproviders@example.com", "London", "daily", true)
	s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	// This should demonstrate that the email sending fails gracefully
	// when no weather providers are configured
	subscriptionService := s.getSubscriptionServiceWithNoProviders()

	// This should succeed at the service level but fail to get weather data
	err = subscriptionService.SendWeatherUpdate("daily")
	s.NoError(err) // SendWeatherUpdate itself succeeds, individual weather lookups fail

	// Wait a bit to ensure no email was sent
	time.Sleep(2 * time.Second)

	// Verify no email was sent (since weather lookup failed)
	// We can't easily check that no email was sent, but we can check the logs
}

func (s *IntegrationTestSuite) getSubscriptionServiceWithNoProviders() service.SubscriptionServiceInterface {
	// Create provider manager with NO providers configured (all empty API keys)
	providerConfig := &providers.ProviderConfiguration{
		WeatherAPIKey:     "", // No API key
		WeatherAPIBaseURL: "",
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false,
		EnableLogging:     false,
		ProviderOrder:     []string{"weatherapi"},
	}

	providerManager, err := providers.NewProviderManager(providerConfig)
	s.Require().NoError(err) // This should NOT fail anymore

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
