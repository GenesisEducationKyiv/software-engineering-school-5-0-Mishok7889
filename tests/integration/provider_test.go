package integration

import (
	"time"

	"weatherapi.app/config"
	"weatherapi.app/providers"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestProviderManagerIntegration() {
	// Test that provider manager correctly handles configuration validation

	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Test 1: Verify provider manager creation fails with no providers configured (fail-fast)
	providerConfigEmpty := &providers.ProviderConfiguration{
		WeatherAPIKey:     "", // No API key
		WeatherAPIBaseURL: "",
		OpenWeatherMapKey: "",
		AccuWeatherKey:    "",
		CacheTTL:          5 * time.Minute,
		LogFilePath:       "test.log",
		EnableCache:       false,
		EnableLogging:     false,
		ProviderOrder:     []string{"weatherapi"},
		CacheType:         "memory",
		CacheConfig:       &config.CacheConfig{Type: "memory"},
	}

	// This should now fail due to fail-fast validation
	_, err = providers.NewProviderManager(providerConfigEmpty)
	s.Error(err, "Provider manager creation should fail with no providers configured")
	s.Contains(err.Error(), "no weather providers configured")

	// Test 2: Verify provider manager works correctly with valid configuration
	subscription := s.CreateTestSubscription("validprovider@example.com", "London", "daily", true)
	s.CreateTestToken(subscription.ID, "unsubscribe", 365*24*time.Hour)

	// This should succeed with valid provider configuration
	subscriptionService := s.getSubscriptionService() // Uses valid config

	// This should succeed and send weather update
	err = subscriptionService.SendWeatherUpdate("daily")
	s.NoError(err)

	// Wait for email to be sent
	time.Sleep(2 * time.Second)

	// Verify email was sent successfully
	s.AssertEmailSent("validprovider@example.com", "Weather Update for London")
}
