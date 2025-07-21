package notification

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
)

type weatherUpdateData struct {
	City             string
	Temperature      float64
	TemperatureUnit  string
	Humidity         float64
	HumidityUnit     string
	Description      string
	LastUpdated      string
	Frequency        string
	SubscriptionCity string
	UnsubscribeURL   string
}

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

	if err := ValidateDeps(deps); err != nil {
		return nil, err
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

func ValidateDeps(deps UseCaseDependencies) error {
	if deps.WeatherService == nil {
		return shared.NewValidationError("weather service is required")
	}
	if deps.SubscriptionService == nil {
		return shared.NewValidationError("subscription service is required")
	}
	if deps.EmailProvider == nil {
		return shared.NewValidationError("email provider is required")
	}
	if deps.TokenRepo == nil {
		return shared.NewValidationError("token repository is required")
	}
	if deps.SubscriptionRepo == nil {
		return shared.NewValidationError("subscription repository is required")
	}
	if deps.Config == nil {
		return shared.NewValidationError("config is required")
	}
	if deps.Logger == nil {
		return shared.NewValidationError("logger is required")
	}

	return nil
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

	emailBody, err := uc.buildWeatherUpdateEmailBody(sub, currentWeather)
	if err != nil {
		return fmt.Errorf("build weather update email body: %w", err)
	}

	emailParams := ports.EmailParams{
		To:      sub.Email,
		Subject: fmt.Sprintf("Weather Update for %s", currentWeather.City),
		Body:    emailBody,
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

func (uc *UseCase) buildWeatherUpdateEmailBody(sub *ports.SubscriptionServiceData, weather *ports.WeatherServiceData) (string, error) {
	baseURL := uc.config.GetAppConfig().BaseURL
	unsubscribeURL := ""
	if sub.UnsubscribeToken != "" {
		unsubscribeURL = fmt.Sprintf("%s/api/unsubscribe/%s", baseURL, sub.UnsubscribeToken)
	}

	data := weatherUpdateData{
		City:             weather.City,
		Temperature:      weather.Temperature,
		TemperatureUnit:  "°C",
		Humidity:         weather.Humidity,
		HumidityUnit:     "%",
		Description:      weather.Description,
		LastUpdated:      weather.Timestamp.Format("2006-01-02 15:04:05 MST"),
		Frequency:        sub.Frequency,
		SubscriptionCity: sub.City,
		UnsubscribeURL:   unsubscribeURL,
	}

	tmpl, err := template.New("weatherUpdate").Parse(WeatherUpdateTemplate)
	if err != nil {
		return "", fmt.Errorf("parse weather update template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute weather update template: %w", err)
	}

	return buf.String(), nil
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
