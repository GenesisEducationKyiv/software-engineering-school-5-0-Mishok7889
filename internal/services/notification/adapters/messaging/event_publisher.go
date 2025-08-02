package messaging

import (
	"context"
	"fmt"

	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type EventPublisher struct {
	broker messaging.MessageBroker
	logger ports.Logger
}

type EventPublisherConfig struct {
	Broker messaging.MessageBroker
	Logger ports.Logger
}

func NewEventPublisher(config EventPublisherConfig) *EventPublisher {
	return &EventPublisher{
		broker: config.Broker,
		logger: config.Logger,
	}
}

func (p *EventPublisher) PublishEvent(ctx context.Context, event messaging.Event) error {
	p.logger.Debug("Publishing event",
		ports.F("event_id", event.ID()),
		ports.F("topic", event.Topic()))

	if err := p.broker.Publish(ctx, event.Topic(), event.Data()); err != nil {
		p.logger.Error("Failed to publish event",
			ports.F("error", err),
			ports.F("event_id", event.ID()),
			ports.F("topic", event.Topic()))
		return fmt.Errorf("publish event %s to topic %s: %w", event.ID(), event.Topic(), err)
	}

	p.logger.Info("Event published successfully",
		ports.F("event_id", event.ID()),
		ports.F("topic", event.Topic()))

	return nil
}

func (p *EventPublisher) PublishCommand(ctx context.Context, command messaging.Command) error {
	p.logger.Debug("Publishing command",
		ports.F("command_id", command.ID()),
		ports.F("topic", command.Topic()))

	if err := p.broker.Publish(ctx, command.Topic(), command.Data()); err != nil {
		p.logger.Error("Failed to publish command",
			ports.F("error", err),
			ports.F("command_id", command.ID()),
			ports.F("topic", command.Topic()))
		return fmt.Errorf("publish command %s to topic %s: %w", command.ID(), command.Topic(), err)
	}

	p.logger.Info("Command published successfully",
		ports.F("command_id", command.ID()),
		ports.F("topic", command.Topic()))

	return nil
}
