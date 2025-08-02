package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type SubscriptionEventConsumer struct {
	emailProvider ports.EmailProvider
	logger        ports.Logger
}

type SubscriptionEventConsumerConfig struct {
	EmailProvider ports.EmailProvider
	Logger        ports.Logger
}

func NewSubscriptionEventConsumer(config SubscriptionEventConsumerConfig) *SubscriptionEventConsumer {
	return &SubscriptionEventConsumer{
		emailProvider: config.EmailProvider,
		logger:        config.Logger,
	}
}

func (c *SubscriptionEventConsumer) HandleMessage(ctx context.Context, message *messaging.Message) error {
	c.logger.Debug("Processing subscription event", ports.F("message_id", message.ID), ports.F("topic", message.Topic))

	switch message.Topic {
	case string(shared.EventTypeSubscriptionCreated):
		return c.handleSubscriptionCreated(ctx, message)
	case string(shared.EventTypeSubscriptionCancelled):
		return c.handleSubscriptionCancelled(ctx, message)
	default:
		c.logger.Warn("Unknown subscription event type", ports.F("topic", message.Topic))
		return nil
	}
}

func (c *SubscriptionEventConsumer) handleSubscriptionCreated(ctx context.Context, message *messaging.Message) error {
	var event shared.SubscriptionCreatedEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		c.logger.Error("Failed to unmarshal subscription created event", ports.F("error", err))
		return fmt.Errorf("unmarshal subscription created event: %w", err)
	}

	emailReq := shared.EmailRequest{
		To:      event.Email,
		Subject: "Welcome to Weather Updates!",
		Body:    c.buildWelcomeEmail(event),
	}

	if err := c.emailProvider.SendEmail(ctx, emailReq); err != nil {
		c.logger.Error("Failed to send welcome email",
			ports.F("error", err),
			ports.F("subscription_id", event.SubscriptionID),
			ports.F("email", event.Email))
		return fmt.Errorf("send welcome email to %s: %w", event.Email, err)
	}

	c.logger.Info("Welcome email sent successfully",
		ports.F("subscription_id", event.SubscriptionID),
		ports.F("email", event.Email))

	return nil
}

func (c *SubscriptionEventConsumer) handleSubscriptionCancelled(ctx context.Context, message *messaging.Message) error {
	var event shared.SubscriptionCancelledEvent
	if err := json.Unmarshal(message.Data, &event); err != nil {
		c.logger.Error("Failed to unmarshal subscription cancelled event", ports.F("error", err))
		return fmt.Errorf("unmarshal subscription cancelled event: %w", err)
	}

	emailReq := shared.EmailRequest{
		To:      event.Email,
		Subject: "Subscription Cancelled",
		Body:    c.buildCancellationEmail(event),
	}

	if err := c.emailProvider.SendEmail(ctx, emailReq); err != nil {
		c.logger.Error("Failed to send cancellation email",
			ports.F("error", err),
			ports.F("subscription_id", event.SubscriptionID),
			ports.F("email", event.Email))
		return fmt.Errorf("send cancellation email to %s: %w", event.Email, err)
	}

	c.logger.Info("Cancellation email sent successfully",
		ports.F("subscription_id", event.SubscriptionID),
		ports.F("email", event.Email))

	return nil
}

func (c *SubscriptionEventConsumer) buildWelcomeEmail(event shared.SubscriptionCreatedEvent) string {
	return fmt.Sprintf(`
Dear Weather Subscriber,

Welcome to Weather Updates!

Your subscription details:
- City: %s
- Frequency: %s updates
- Subscription ID: %s

You will receive weather updates for %s at %s intervals.

Thank you for subscribing!

Best regards,
Weather API Team
`, event.City, event.Frequency, event.SubscriptionID, event.City, event.Frequency)
}

func (c *SubscriptionEventConsumer) buildCancellationEmail(event shared.SubscriptionCancelledEvent) string {
	return fmt.Sprintf(`
Dear Weather Subscriber,

Your weather subscription (ID: %s) has been successfully cancelled.

You will no longer receive weather updates.

If this was done in error, please contact our support team.

Best regards,
Weather API Team
`, event.SubscriptionID)
}
