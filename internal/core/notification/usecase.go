package notification

import (
	"context"
	"fmt"
	"time"

	"weatherapi.app/internal/core/subscription"
	"weatherapi.app/internal/core/weather"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

type UseCase struct {
	subscriptionRepo ports.SubscriptionRepository
	tokenRepo        ports.TokenRepository
	emailProvider    ports.EmailProvider
	weatherUseCase   *weather.UseCase
	config           ports.ConfigProvider
	logger           ports.Logger
}

type UseCaseDependencies struct {
	SubscriptionRepo ports.SubscriptionRepository
	TokenRepo        ports.TokenRepository
	EmailProvider    ports.EmailProvider
	WeatherUseCase   *weather.UseCase
	Config           ports.ConfigProvider
	Logger           ports.Logger
}

type SendWeatherUpdateParams struct {
	Frequency subscription.Frequency
}

type SendSingleUpdateParams struct {
	Subscription *subscription.Subscription
	Weather      *weather.Weather
}

func NewUseCase(deps UseCaseDependencies) (*UseCase, error) {
	if deps.SubscriptionRepo == nil {
		return nil, errors.NewValidationError("subscription repository is required")
	}
	if deps.TokenRepo == nil {
		return nil, errors.NewValidationError("token repository is required")
	}
	if deps.EmailProvider == nil {
		return nil, errors.NewValidationError("email provider is required")
	}
	if deps.WeatherUseCase == nil {
		return nil, errors.NewValidationError("weather use case is required")
	}
	if deps.Config == nil {
		return nil, errors.NewValidationError("config is required")
	}
	if deps.Logger == nil {
		return nil, errors.NewValidationError("logger is required")
	}

	return &UseCase{
		subscriptionRepo: deps.SubscriptionRepo,
		tokenRepo:        deps.TokenRepo,
		emailProvider:    deps.EmailProvider,
		weatherUseCase:   deps.WeatherUseCase,
		config:           deps.Config,
		logger:           deps.Logger,
	}, nil
}

func (uc *UseCase) SendWeatherUpdates(ctx context.Context, params SendWeatherUpdateParams) error {
	if !params.Frequency.IsValid() {
		return errors.NewValidationError("invalid frequency")
	}

	uc.logger.Info("Starting weather update notifications", ports.F("frequency", params.Frequency))

	subscriptionsData, err := uc.subscriptionRepo.GetConfirmedByFrequency(ctx, params.Frequency.String())
	if err != nil {
		return fmt.Errorf("get subscriptions for frequency %s: %w", params.Frequency, err)
	}

	if len(subscriptionsData) == 0 {
		uc.logger.Debug("No subscriptions found for frequency", ports.F("frequency", params.Frequency))
		return nil
	}

	uc.logger.Info("Processing weather updates",
		ports.F("frequency", params.Frequency),
		ports.F("count", len(subscriptionsData)))

	successCount := 0
	errorCount := 0

	for _, subData := range subscriptionsData {
		sub := uc.convertFromPortsSubscription(subData)
		if err := uc.sendWeatherUpdateToSubscription(ctx, sub); err != nil {
			uc.logger.Error("Failed to send weather update",
				ports.F("error", err),
				ports.F("email", sub.Email),
				ports.F("city", sub.City))
			errorCount++
		} else {
			successCount++
		}
	}

	uc.logger.Info("Weather update notifications completed",
		ports.F("frequency", params.Frequency),
		ports.F("total", len(subscriptionsData)),
		ports.F("success", successCount),
		ports.F("errors", errorCount))

	if errorCount > 0 {
		return fmt.Errorf("failed to send %d out of %d weather updates", errorCount, len(subscriptionsData))
	}

	return nil
}

func (uc *UseCase) sendWeatherUpdateToSubscription(ctx context.Context, sub *subscription.Subscription) error {
	weatherRequest := weather.WeatherRequest{City: sub.City}
	currentWeather, err := uc.weatherUseCase.GetWeather(ctx, weatherRequest)
	if err != nil {
		return fmt.Errorf("get weather for city %s: %w", sub.City, err)
	}

	updateParams := SendSingleUpdateParams{
		Subscription: sub,
		Weather:      currentWeather,
	}

	if err := uc.SendSingleWeatherUpdate(ctx, updateParams); err != nil {
		return fmt.Errorf("send weather update email: %w", err)
	}

	return nil
}

func (uc *UseCase) SendSingleWeatherUpdate(ctx context.Context, params SendSingleUpdateParams) error {
	if params.Subscription == nil {
		return errors.NewValidationError("subscription is required")
	}
	if params.Weather == nil {
		return errors.NewValidationError("weather is required")
	}

	unsubscribeToken, err := uc.getOrCreateUnsubscribeToken(ctx, params.Subscription.ID)
	if err != nil {
		uc.logger.Warn("Failed to get unsubscribe token", ports.F("error", err))
	}

	emailParams := ports.EmailParams{
		To:      params.Subscription.Email,
		Subject: fmt.Sprintf("Weather Update for %s", params.Weather.City),
		Body:    uc.buildWeatherUpdateEmailBody(params.Subscription, params.Weather, unsubscribeToken),
		IsHTML:  true,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send weather update email: %w", err)
	}

	uc.logger.Debug("Weather update sent successfully",
		ports.F("email", params.Subscription.Email),
		ports.F("city", params.Weather.City),
		ports.F("temperature", params.Weather.Temperature))

	return nil
}

func (uc *UseCase) getOrCreateUnsubscribeToken(ctx context.Context, subscriptionID uint) (string, error) {
	existingToken, err := uc.tokenRepo.FindBySubscriptionIDAndType(ctx, subscriptionID, subscription.TokenTypeUnsubscribe.String())
	if err == nil && existingToken != nil && !time.Now().After(existingToken.ExpiresAt) {
		return existingToken.Value, nil
	}

	newToken, err := uc.tokenRepo.CreateUnsubscribeToken(ctx, subscriptionID, 365*24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("create unsubscribe token: %w", err)
	}

	return newToken.Value, nil
}

func (uc *UseCase) convertFromPortsSubscription(data *ports.SubscriptionData) *subscription.Subscription {
	return &subscription.Subscription{
		ID:        data.ID,
		Email:     data.Email,
		City:      data.City,
		Frequency: subscription.FrequencyFromString(data.Frequency),
		Confirmed: data.Confirmed,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}
}

func (uc *UseCase) buildWeatherUpdateEmailBody(sub *subscription.Subscription, weatherData *weather.Weather, unsubscribeToken string) string {
	baseURL := uc.config.GetAppConfig().BaseURL
	unsubscribeURL := ""
	if unsubscribeToken != "" {
		unsubscribeURL = fmt.Sprintf("%s/api/unsubscribe/%s", baseURL, unsubscribeToken)
	}

	temperatureUnit := "°C"
	humidityUnit := "%"

	emailBody := fmt.Sprintf(`
		<h2>Weather Update for %s</h2>
		<div style="background-color: #f5f5f5; padding: 20px; border-radius: 8px; margin: 20px 0;">
			<h3 style="color: #333; margin-top: 0;">Current Weather</h3>
			<p style="font-size: 18px; margin: 10px 0;">
				<strong>Temperature:</strong> %.1f%s
			</p>
			<p style="font-size: 16px; margin: 10px 0;">
				<strong>Humidity:</strong> %.1f%s
			</p>
			<p style="font-size: 16px; margin: 10px 0;">
				<strong>Description:</strong> %s
			</p>
			<p style="font-size: 14px; color: #666; margin: 10px 0;">
				<strong>Last Updated:</strong> %s
			</p>
		</div>
		<hr style="border: none; border-top: 1px solid #ddd; margin: 20px 0;">
		<p style="font-size: 12px; color: #888;">
			You are receiving this because you subscribed to <strong>%s</strong> weather updates for <strong>%s</strong>.
		</p>`,
		weatherData.City,
		weatherData.Temperature, temperatureUnit,
		weatherData.Humidity, humidityUnit,
		weatherData.Description,
		weatherData.Timestamp.Format("2006-01-02 15:04:05 MST"),
		sub.Frequency,
		sub.City)

	if unsubscribeURL != "" {
		emailBody += fmt.Sprintf(`
		<p style="font-size: 12px; color: #888;">
			To unsubscribe from these updates, <a href="%s" style="color: #0066cc;">click here</a>.
		</p>`, unsubscribeURL)
	}

	return emailBody
}

func (uc *UseCase) CleanupExpiredTokens(ctx context.Context) error {
	uc.logger.Debug("Starting cleanup of expired tokens")

	deletedCount, err := uc.tokenRepo.DeleteExpiredTokens(ctx)
	if err != nil {
		return fmt.Errorf("cleanup expired tokens: %w", err)
	}

	if deletedCount > 0 {
		uc.logger.Info("Cleaned up expired tokens", ports.F("count", deletedCount))
	} else {
		uc.logger.Debug("No expired tokens to cleanup")
	}

	return nil
}

func (uc *UseCase) GetNotificationStats(ctx context.Context) (NotificationStats, error) {
	hourlyCount, err := uc.subscriptionRepo.CountByFrequency(ctx, subscription.FrequencyHourly.String())
	if err != nil {
		return NotificationStats{}, fmt.Errorf("count hourly subscriptions: %w", err)
	}

	dailyCount, err := uc.subscriptionRepo.CountByFrequency(ctx, subscription.FrequencyDaily.String())
	if err != nil {
		return NotificationStats{}, fmt.Errorf("count daily subscriptions: %w", err)
	}

	totalCount, err := uc.subscriptionRepo.CountConfirmed(ctx)
	if err != nil {
		return NotificationStats{}, fmt.Errorf("count total confirmed subscriptions: %w", err)
	}

	stats := NotificationStats{
		TotalSubscriptions:  int(totalCount),
		HourlySubscriptions: int(hourlyCount),
		DailySubscriptions:  int(dailyCount),
		LastUpdated:         time.Now(),
	}

	return stats, nil
}
