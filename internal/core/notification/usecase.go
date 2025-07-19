package notification

import (
	"context"
	"fmt"
	"time"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type UseCase struct {
	weatherService      ports.WeatherService
	subscriptionService ports.SubscriptionService
	emailProvider       ports.EmailProvider
	tokenRepo           ports.TokenRepository
	subscriptionRepo    ports.SubscriptionRepository
	config              ports.ConfigProvider
	logger              ports.Logger
}

type UseCaseDependencies struct {
	WeatherService      ports.WeatherService
	SubscriptionService ports.SubscriptionService
	EmailProvider       ports.EmailProvider
	TokenRepo           ports.TokenRepository
	SubscriptionRepo    ports.SubscriptionRepository
	Config              ports.ConfigProvider
	Logger              ports.Logger
}

type SendWeatherUpdateParams struct {
	Frequency string
}

func NewUseCase(deps UseCaseDependencies) (*UseCase, error) {
	if deps.WeatherService == nil {
		return nil, shared.NewValidationError("weather service is required")
	}
	if deps.SubscriptionService == nil {
		return nil, shared.NewValidationError("subscription service is required")
	}
	if deps.EmailProvider == nil {
		return nil, shared.NewValidationError("email provider is required")
	}
	if deps.TokenRepo == nil {
		return nil, shared.NewValidationError("token repository is required")
	}
	if deps.SubscriptionRepo == nil {
		return nil, shared.NewValidationError("subscription repository is required")
	}
	if deps.Config == nil {
		return nil, shared.NewValidationError("config is required")
	}
	if deps.Logger == nil {
		return nil, shared.NewValidationError("logger is required")
	}

	return &UseCase{
		weatherService:      deps.WeatherService,
		subscriptionService: deps.SubscriptionService,
		emailProvider:       deps.EmailProvider,
		tokenRepo:           deps.TokenRepo,
		subscriptionRepo:    deps.SubscriptionRepo,
		config:              deps.Config,
		logger:              deps.Logger,
	}, nil
}

func (uc *UseCase) SendWeatherUpdates(ctx context.Context, params SendWeatherUpdateParams) error {
	if params.Frequency == "" {
		return shared.NewValidationError("frequency is required")
	}

	uc.logger.Info("Starting weather update notifications", ports.F("frequency", params.Frequency))

	subscriptions, err := uc.subscriptionService.GetConfirmedSubscriptions(ctx, params.Frequency)
	if err != nil {
		return fmt.Errorf("get subscriptions for frequency %s: %w", params.Frequency, err)
	}

	if len(subscriptions) == 0 {
		uc.logger.Debug("No subscriptions found for frequency", ports.F("frequency", params.Frequency))
		return nil
	}

	uc.logger.Info("Processing weather updates",
		ports.F("frequency", params.Frequency),
		ports.F("count", len(subscriptions)))

	successCount := 0
	errorCount := 0

	for _, sub := range subscriptions {
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
		ports.F("total", len(subscriptions)),
		ports.F("success", successCount),
		ports.F("errors", errorCount))

	if errorCount > 0 {
		return fmt.Errorf("failed to send %d out of %d weather updates", errorCount, len(subscriptions))
	}

	return nil
}

func (uc *UseCase) sendWeatherUpdateToSubscription(ctx context.Context, sub *ports.SubscriptionServiceData) error {
	currentWeather, err := uc.weatherService.GetWeather(ctx, sub.City)
	if err != nil {
		return fmt.Errorf("get weather for city %s: %w", sub.City, err)
	}

	emailParams := ports.EmailParams{
		To:      sub.Email,
		Subject: fmt.Sprintf("Weather Update for %s", currentWeather.City),
		Body:    uc.buildWeatherUpdateEmailBody(sub, currentWeather),
		Format:  ports.FormatHTML,
	}

	if err := uc.emailProvider.SendEmail(ctx, emailParams); err != nil {
		return fmt.Errorf("send weather update email: %w", err)
	}

	uc.logger.Debug("Weather update sent successfully",
		ports.F("email", sub.Email),
		ports.F("city", currentWeather.City),
		ports.F("temperature", currentWeather.Temperature))

	return nil
}

func (uc *UseCase) buildWeatherUpdateEmailBody(sub *ports.SubscriptionServiceData, weather *ports.WeatherServiceData) string {
	baseURL := uc.config.GetAppConfig().BaseURL
	unsubscribeURL := ""
	if sub.UnsubscribeToken != "" {
		unsubscribeURL = fmt.Sprintf("%s/api/unsubscribe/%s", baseURL, sub.UnsubscribeToken)
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
		weather.City,
		weather.Temperature, temperatureUnit,
		weather.Humidity, humidityUnit,
		weather.Description,
		weather.Timestamp.Format("2006-01-02 15:04:05 MST"),
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
	hourlyCount, err := uc.subscriptionRepo.CountByFrequency(ctx, "hourly")
	if err != nil {
		return NotificationStats{}, fmt.Errorf("count hourly subscriptions: %w", err)
	}

	dailyCount, err := uc.subscriptionRepo.CountByFrequency(ctx, "daily")
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
