package messaging

import "context"

type MessageBroker interface {
	Publish(ctx context.Context, topic string, message []byte) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	SubscribeWithGroup(ctx context.Context, topic, group string, handler MessageHandler) error
	Close() error
}

type MessageHandler func(ctx context.Context, message *Message) error

type Message struct {
	ID      string
	Topic   string
	Data    []byte
	Headers map[string]string
}

type Publisher interface {
	PublishEvent(ctx context.Context, event Event) error
	PublishCommand(ctx context.Context, command Command) error
}

type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

type Event interface {
	Topic() string
	Data() []byte
	ID() string
	Headers() map[string]string
}

type Command interface {
	Topic() string
	Data() []byte
	ID() string
	Headers() map[string]string
}
