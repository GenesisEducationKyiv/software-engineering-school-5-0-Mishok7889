package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type WeatherEventConsumer struct {
	subscriptionService ports.SubscriptionService
	emailProvider       ports.EmailProvider
	logger              ports.Logger
}

type WeatherEventConsumerConfig struct {
	SubscriptionService ports.SubscriptionService
	EmailProvider       ports.EmailProvider
	Logger              ports.Logger
}

func NewWeatherEventConsumer(config WeatherEventConsumerConfig) *WeatherEventConsumer {
	return &WeatherEventConsumer{
		subscriptionService: config.SubscriptionService,
		emailProvider:       config.EmailProvider,
		logger:              config.Logger,
	}
}

func (c *WeatherEventConsumer) HandleMessage(ctx context.Context, message *messaging.Message) error {
	c.logger.Debug("Processing weather event", ports.F("message_id", message.ID), ports.F("topic", message.Topic))

	var event shared.WeatherDataFetchedEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		c.logger.Error("Failed to unmarshal weather event", ports.F("error", err), ports.F("message_id", message.ID))
		return fmt.Errorf("unmarshal weather event: %w", err)
	}

	subscriptions, err := c.subscriptionService.GetActiveSubscriptionsByCity(ctx, event.City)
	if err != nil {
		c.logger.Error("Failed to get active subscriptions", ports.F("error", err), ports.F("city", event.City))
		return fmt.Errorf("get active subscriptions for city %s: %w", event.City, err)
	}

	if len(subscriptions) == 0 {
		c.logger.Debug("No active subscriptions found for city", ports.F("city", event.City))
		return nil
	}

	for _, subscription := range subscriptions {
		emailReq := shared.EmailRequest{
			To:      subscription.Email,
			Subject: fmt.Sprintf("Weather Update for %s", event.City),
			Body:    c.buildWeatherUpdateEmail(event, subscription),
		}

		if err := c.emailProvider.SendEmail(ctx, emailReq); err != nil {
			c.logger.Error("Failed to send weather update email",
				ports.F("error", err),
				ports.F("subscription_id", subscription.ID),
				ports.F("email", subscription.Email))
			return fmt.Errorf("send weather update email to %s: %w", subscription.Email, err)
		}

		c.logger.Info("Weather update email sent successfully",
			ports.F("subscription_id", subscription.ID),
			ports.F("city", event.City),
			ports.F("email", subscription.Email))
	}

	return nil
}

func (c *WeatherEventConsumer) buildWeatherUpdateEmail(event shared.WeatherDataFetchedEvent, subscription shared.Subscription) string {
	return fmt.Sprintf(`
Dear Weather Subscriber,

Here's the latest weather update for %s:

Temperature: %.1f°C
Humidity: %.1f%%
Description: %s

This update was triggered by a weather data change.

Best regards,
Weather API Team
`, event.City, event.Temperature, event.Humidity, event.Description)
}
