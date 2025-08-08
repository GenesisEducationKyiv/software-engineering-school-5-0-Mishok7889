package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type CommandConsumer struct {
	subscriptionService ports.SubscriptionService
	weatherService      ports.WeatherService
	emailProvider       ports.EmailProvider
	logger              ports.Logger
}

type CommandConsumerConfig struct {
	SubscriptionService ports.SubscriptionService
	WeatherService      ports.WeatherService
	EmailProvider       ports.EmailProvider
	Logger              ports.Logger
}

func NewCommandConsumer(config CommandConsumerConfig) *CommandConsumer {
	return &CommandConsumer{
		subscriptionService: config.SubscriptionService,
		weatherService:      config.WeatherService,
		emailProvider:       config.EmailProvider,
		logger:              config.Logger,
	}
}

func (c *CommandConsumer) HandleMessage(ctx context.Context, message *messaging.Message) error {
	c.logger.Debug("Processing command", ports.F("message_id", message.ID), ports.F("topic", message.Topic))

	switch message.Topic {
	case string(shared.CommandTypeSendWeatherUpdates):
		return c.handleSendWeatherUpdates(ctx, message)
	default:
		c.logger.Warn("Unknown command type", ports.F("topic", message.Topic))
		return nil
	}
}

func (c *CommandConsumer) handleSendWeatherUpdates(ctx context.Context, message *messaging.Message) error {
	var command shared.SendWeatherUpdatesCommand
	if err := json.Unmarshal(message.Data, &command); err != nil {
		c.logger.Error("Failed to unmarshal send weather updates command", ports.F("error", err))
		return fmt.Errorf("unmarshal send weather updates command: %w", err)
	}

	frequency := shared.FrequencyFromString(command.Frequency)
	if frequency != shared.FrequencyHourly && frequency != shared.FrequencyDaily {
		c.logger.Error("Invalid frequency in command", ports.F("frequency", command.Frequency))
		return fmt.Errorf("invalid frequency: %s", command.Frequency)
	}

	subscriptions, err := c.subscriptionService.GetActiveSubscriptionsByFrequency(ctx, frequency)
	if err != nil {
		c.logger.Error("Failed to get active subscriptions by frequency",
			ports.F("error", err), ports.F("frequency", command.Frequency))
		return fmt.Errorf("get active subscriptions for frequency %s: %w", command.Frequency, err)
	}

	if len(subscriptions) == 0 {
		c.logger.Info("No active subscriptions found for frequency",
			ports.F("frequency", command.Frequency))
		return nil
	}

	c.logger.Info("Processing weather updates for subscriptions",
		ports.F("frequency", command.Frequency),
		ports.F("subscription_count", len(subscriptions)))

	cityWeatherMap := make(map[string]*shared.WeatherData)

	for _, subscription := range subscriptions {
		weatherData, exists := cityWeatherMap[subscription.City]
		if !exists {
			var err error
			weatherData, err = c.weatherService.GetWeatherByCity(ctx, subscription.City)
			if err != nil {
				c.logger.Error("Failed to get weather data",
					ports.F("error", err),
					ports.F("city", subscription.City),
					ports.F("subscription_id", subscription.ID))
				continue
			}
			cityWeatherMap[subscription.City] = weatherData
		}

		emailReq := shared.EmailRequest{
			To:      subscription.Email,
			Subject: fmt.Sprintf("%s Weather Update for %s", c.capitalizeFrequency(command.Frequency), subscription.City),
			Body:    c.buildScheduledWeatherEmail(*weatherData, subscription, command.Frequency),
		}

		if err := c.emailProvider.SendEmail(ctx, emailReq); err != nil {
			c.logger.Error("Failed to send scheduled weather email",
				ports.F("error", err),
				ports.F("subscription_id", subscription.ID),
				ports.F("email", subscription.Email))
			continue
		}

		c.logger.Debug("Scheduled weather email sent successfully",
			ports.F("subscription_id", subscription.ID),
			ports.F("city", subscription.City),
			ports.F("email", subscription.Email))
	}

	c.logger.Info("Completed processing weather updates command",
		ports.F("frequency", command.Frequency),
		ports.F("subscriptions_processed", len(subscriptions)))

	return nil
}

func (c *CommandConsumer) buildScheduledWeatherEmail(weather shared.WeatherData, subscription shared.Subscription, frequency string) string {
	return fmt.Sprintf(`
Dear Weather Subscriber,

Your %s weather update for %s:

Temperature: %.1f°C
Humidity: %.1f%%
Description: %s
Last Updated: %s

This is your scheduled %s weather update.

Best regards,
Weather API Team
`, frequency, weather.City, weather.Temperature, weather.Humidity, weather.Description,
		weather.LastUpdated.Format("2006-01-02 15:04:05"), frequency)
}

func (c *CommandConsumer) capitalizeFrequency(frequency string) string {
	if len(frequency) == 0 {
		return frequency
	}
	return string(frequency[0]-32) + frequency[1:]
}
