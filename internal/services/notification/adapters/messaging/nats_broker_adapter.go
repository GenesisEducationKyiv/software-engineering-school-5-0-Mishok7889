package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
)

type NATSBrokerAdapter struct {
	conn   *nats.Conn
	logger ports.Logger
	subs   map[string]*nats.Subscription
	mu     sync.RWMutex
}

type NATSBrokerConfig struct {
	URL    string
	Logger ports.Logger
}

func NewNATSBrokerAdapter(config NATSBrokerConfig) (*NATSBrokerAdapter, error) {
	conn, err := nats.Connect(config.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS server: %w", err)
	}

	return &NATSBrokerAdapter{
		conn:   conn,
		logger: config.Logger,
		subs:   make(map[string]*nats.Subscription),
	}, nil
}

func (n *NATSBrokerAdapter) Publish(ctx context.Context, topic string, message []byte) error {
	n.logger.Debug("Publishing message", ports.F("topic", topic), ports.F("size", len(message)))

	// Create NATS message with headers support
	msg := &nats.Msg{
		Subject: topic,
		Data:    message,
		Header:  make(nats.Header),
	}

	// Try to extract headers from JSON message if it's an event/command
	n.extractAndSetHeaders(msg, message)

	if err := n.conn.PublishMsg(msg); err != nil {
		n.logger.Error("Failed to publish message", ports.F("error", err), ports.F("topic", topic))
		return fmt.Errorf("publish message to topic %s: %w", topic, err)
	}

	n.logger.Debug("Message published successfully", ports.F("topic", topic))
	return nil
}

func (n *NATSBrokerAdapter) Subscribe(ctx context.Context, topic string, handler messaging.MessageHandler) error {
	n.logger.Info("Subscribing to topic", ports.F("topic", topic))

	sub, err := n.conn.Subscribe(topic, func(msg *nats.Msg) {
		message := &messaging.Message{
			ID:      extractMessageID(msg),
			Topic:   msg.Subject,
			Data:    msg.Data,
			Headers: extractHeaders(msg),
		}

		if err := handler(ctx, message); err != nil {
			n.logger.Error("Message handler error",
				ports.F("error", err),
				ports.F("topic", topic),
				ports.F("message_id", message.ID))
		}
	})

	if err != nil {
		return fmt.Errorf("subscribe to topic %s: %w", topic, err)
	}

	n.mu.Lock()
	n.subs[topic] = sub
	n.mu.Unlock()

	n.logger.Info("Successfully subscribed to topic", ports.F("topic", topic))
	return nil
}

func (n *NATSBrokerAdapter) SubscribeWithGroup(ctx context.Context, topic, group string, handler messaging.MessageHandler) error {
	n.logger.Info("Subscribing to topic with group", ports.F("topic", topic), ports.F("group", group))

	sub, err := n.conn.QueueSubscribe(topic, group, func(msg *nats.Msg) {
		message := &messaging.Message{
			ID:      extractMessageID(msg),
			Topic:   msg.Subject,
			Data:    msg.Data,
			Headers: extractHeaders(msg),
		}

		if err := handler(ctx, message); err != nil {
			n.logger.Error("Message handler error",
				ports.F("error", err),
				ports.F("topic", topic),
				ports.F("group", group),
				ports.F("message_id", message.ID))
		}
	})

	if err != nil {
		return fmt.Errorf("subscribe to topic %s with group %s: %w", topic, group, err)
	}

	subscriptionKey := fmt.Sprintf("%s:%s", topic, group)
	n.mu.Lock()
	n.subs[subscriptionKey] = sub
	n.mu.Unlock()

	n.logger.Info("Successfully subscribed to topic with group",
		ports.F("topic", topic), ports.F("group", group))
	return nil
}

func (n *NATSBrokerAdapter) Close() error {
	n.logger.Info("Closing NATS broker connection")

	n.mu.Lock()
	defer n.mu.Unlock()

	for key, sub := range n.subs {
		if err := sub.Unsubscribe(); err != nil {
			n.logger.Warn("Error unsubscribing", ports.F("subscription", key), ports.F("error", err))
		}
	}

	n.conn.Close()
	n.logger.Info("NATS broker connection closed")
	return nil
}

func extractMessageID(msg *nats.Msg) string {
	if msg.Header != nil {
		if id := msg.Header.Get("message_id"); id != "" {
			return id
		}
		if id := msg.Header.Get("event_id"); id != "" {
			return id
		}
		if id := msg.Header.Get("command_id"); id != "" {
			return id
		}
	}
	return fmt.Sprintf("msg_%s", msg.Reply)
}

func extractHeaders(msg *nats.Msg) map[string]string {
	headers := make(map[string]string)
	if msg.Header != nil {
		for key, values := range msg.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}
	return headers
}

func (n *NATSBrokerAdapter) extractAndSetHeaders(msg *nats.Msg, data []byte) {
	// Try to parse JSON and extract common headers
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return // Not JSON or malformed, skip header extraction
	}

	// Extract common event/command headers
	if eventID, ok := jsonData["event_id"].(string); ok {
		msg.Header.Set("event_id", eventID)
	}
	if commandID, ok := jsonData["command_id"].(string); ok {
		msg.Header.Set("command_id", commandID)
	}
	if eventType, ok := jsonData["event_type"].(string); ok {
		msg.Header.Set("event_type", eventType)
	}
	if commandType, ok := jsonData["command_type"].(string); ok {
		msg.Header.Set("command_type", commandType)
	}
	if timestamp, ok := jsonData["timestamp"].(string); ok {
		msg.Header.Set("timestamp", timestamp)
	}
	if correlationID, ok := jsonData["correlation_id"].(string); ok {
		msg.Header.Set("correlation_id", correlationID)
	}
}
