package integration

import (
	"context"
	"time"

	"weatherapi.app/internal/core/notification"
	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/tests/integration/helpers"
)

func (s *IntegrationTestSuite) TestWeatherProviderIntegration() {
	// Test that weather providers work correctly through the hexagonal architecture

	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Test 1: Verify weather service works with valid configuration
	// This tests the complete chain: HTTP -> Weather Handler -> Weather Use Case -> Weather Provider -> External API
	ctx := context.Background()

	// Get weather through the application's weather use case (not directly through provider)
	weatherUseCase := s.application.GetWeatherUseCase()
	s.Require().NotNil(weatherUseCase, "Weather use case should be available")

	// Test weather retrieval
	weatherRequest := weather.WeatherRequest{City: "London"}
	weatherData, err := weatherUseCase.GetWeather(ctx, weatherRequest)
	s.NoError(err, "Weather retrieval should succeed")
	s.NotNil(weatherData, "Weather data should not be nil")
	s.Equal(15.0, weatherData.Temperature)
	s.Equal(76.0, weatherData.Humidity)
	s.Equal("Partly cloudy", weatherData.Description)

	// Test 2: Verify provider metrics are available
	providerInfo := weatherUseCase.GetProviderInfo(ctx)
	s.NotNil(providerInfo, "Provider info should be available")

	// Verify provider info contains expected fields
	s.Contains(providerInfo, "cache_enabled")
	s.Contains(providerInfo, "provider_order")

	// Test 3: Verify cache metrics if caching is enabled
	cacheMetrics, err := weatherUseCase.GetCacheMetrics(ctx)
	if err == nil {
		// If cache is enabled, verify metrics structure
		s.GreaterOrEqual(cacheMetrics.TotalOps, int64(0))
		s.GreaterOrEqual(cacheMetrics.HitRatio, 0.0)
		s.LessOrEqual(cacheMetrics.HitRatio, 1.0)
	}
}

func (s *IntegrationTestSuite) TestWeatherProviderFailover() {
	// Test provider failover behavior through the application
	// Note: In integration tests, we're using a mock server, so we can't easily test real failover
	// But we can test that the system handles provider errors gracefully

	ctx := context.Background()
	weatherUseCase := s.application.GetWeatherUseCase()

	// Test 1: Valid city should work
	weatherRequest := weather.WeatherRequest{City: "London"}
	weatherData, err := weatherUseCase.GetWeather(ctx, weatherRequest)
	s.NoError(err)
	s.NotNil(weatherData)

	// Test 2: Invalid city should return appropriate error
	invalidRequest := weather.WeatherRequest{City: "NonExistentCity"}
	weatherData, err = weatherUseCase.GetWeather(ctx, invalidRequest)
	if err != nil {
		// Error is expected for non-existent cities
		s.Nil(weatherData)
		s.T().Logf("Expected error for non-existent city: %v", err)
	}

	// Test 3: Empty city should return validation error
	emptyRequest := weather.WeatherRequest{City: ""}
	weatherData, err = weatherUseCase.GetWeather(ctx, emptyRequest)
	s.Error(err, "Empty city should return validation error")
	s.Nil(weatherData)
}

func (s *IntegrationTestSuite) TestWeatherProviderCaching() {
	// Test caching behavior through the weather use case
	ctx := context.Background()
	weatherUseCase := s.application.GetWeatherUseCase()

	city := "London"
	weatherRequest := weather.WeatherRequest{City: city}

	// First request - should hit provider
	weather1, err := weatherUseCase.GetWeather(ctx, weatherRequest)
	s.NoError(err)
	s.NotNil(weather1)

	// Get initial cache metrics
	initialMetrics, err := weatherUseCase.GetCacheMetrics(ctx)
	if err != nil {
		s.T().Logf("Cache metrics not available: %v", err)
		return // Skip cache testing if cache is not enabled
	}

	// Second request - should potentially hit cache (if caching is enabled)
	weather2, err := weatherUseCase.GetWeather(ctx, weatherRequest)
	s.NoError(err)
	s.NotNil(weather2)

	// Weather data should be the same
	s.Equal(weather1.Temperature, weather2.Temperature)
	s.Equal(weather1.Humidity, weather2.Humidity)
	s.Equal(weather1.Description, weather2.Description)

	// Get final cache metrics
	finalMetrics, err := weatherUseCase.GetCacheMetrics(ctx)
	if err == nil {
		// Cache operations should have increased
		s.GreaterOrEqual(finalMetrics.TotalOps, initialMetrics.TotalOps)
		s.T().Logf("Cache metrics - Initial: %+v, Final: %+v", initialMetrics, finalMetrics)
	}
}

func (s *IntegrationTestSuite) TestWeatherProviderWithSubscription() {
	// Test integration between weather provider and subscription service
	err := helpers.ClearEmails()
	s.Require().NoError(err)

	// Create a confirmed subscription
	testSubscription := s.CreateTestSubscription("weatherprovider@example.com", "London", "daily", true)
	s.CreateTestToken(testSubscription.ID, "unsubscribe", 365*24*time.Hour)

	// Get notification use case
	notificationUseCase := s.application.GetNotificationUseCase()
	s.Require().NotNil(notificationUseCase, "Notification use case should be available")

	// Send weather update (this should use the weather provider internally)
	ctx := context.Background()
	dailyParams := notification.SendWeatherUpdateParams{
		Frequency: subscription.FrequencyDaily,
	}
	err = notificationUseCase.SendWeatherUpdates(ctx, dailyParams)
	s.NoError(err, "Weather update should succeed")

	// Wait for email to be sent
	s.Require().Eventually(func() bool {
		return helpers.CheckEmailSent("weatherprovider@example.com", "Weather Update for London")
	}, 5*time.Second, 200*time.Millisecond)

	// Verify email was sent successfully
	s.AssertEmailSent("weatherprovider@example.com", "Weather Update for London")
}
