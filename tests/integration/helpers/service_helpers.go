package helpers

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
	messagingadapter "weatherapi.app/internal/services/notification/adapters/messaging"
)

type TestBroker struct {
	t          *testing.T
	natsServer *server.Server
	natsClient *nats.Conn
	broker     messaging.MessageBroker
	logger     ports.Logger
	cleanupFns []func()
	mu         sync.Mutex
}

func NewTestBroker(t *testing.T, logger ports.Logger) *TestBroker {
	return &TestBroker{
		t:          t,
		logger:     logger,
		cleanupFns: make([]func(), 0),
	}
}

func (tb *TestBroker) Setup() (messaging.MessageBroker, string) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	natsPort := tb.getFreePort()
	natsURL := fmt.Sprintf("nats://127.0.0.1:%d", natsPort)

	opts := &server.Options{
		Host: "127.0.0.1",
		Port: natsPort,
	}

	natsServer, err := server.NewServer(opts)
	require.NoError(tb.t, err)

	go natsServer.Start()

	if !natsServer.ReadyForConnections(5 * time.Second) {
		tb.t.Fatal("NATS server failed to start")
	}

	tb.natsServer = natsServer
	tb.addCleanup(func() {
		natsServer.Shutdown()
	})

	client, err := nats.Connect(natsURL)
	require.NoError(tb.t, err)

	tb.natsClient = client
	tb.addCleanup(func() {
		client.Close()
	})

	broker, err := messagingadapter.NewNATSBrokerAdapter(messagingadapter.NATSBrokerConfig{
		URL:    natsURL,
		Logger: tb.logger,
	})
	require.NoError(tb.t, err)

	tb.broker = broker
	tb.addCleanup(func() {
		if err := broker.Close(); err != nil {
			tb.t.Logf("Failed to close broker: %v", err)
		}
	})

	return broker, natsURL
}

func (tb *TestBroker) Cleanup() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	for i := len(tb.cleanupFns) - 1; i >= 0; i-- {
		tb.cleanupFns[i]()
	}
	tb.cleanupFns = nil
}

func (tb *TestBroker) addCleanup(fn func()) {
	tb.cleanupFns = append(tb.cleanupFns, fn)
}

func (tb *TestBroker) getFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(tb.t, err)

	l, err := net.ListenTCP("tcp", addr)
	require.NoError(tb.t, err)
	defer func() {
		if err := l.Close(); err != nil {
			tb.t.Logf("Failed to close TCP listener: %v", err)
		}
	}()

	return l.Addr().(*net.TCPAddr).Port
}

type EventCapture struct {
	mu     sync.RWMutex
	events []messaging.Message
	done   chan struct{}
}

func NewEventCapture() *EventCapture {
	return &EventCapture{
		events: make([]messaging.Message, 0),
		done:   make(chan struct{}),
	}
}

func (ec *EventCapture) CaptureMessage(ctx context.Context, message *messaging.Message) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.events = append(ec.events, *message)
	select {
	case ec.done <- struct{}{}:
	default:
	}
	return nil
}

func (ec *EventCapture) WaitForMessages(count int, timeout time.Duration) []messaging.Message {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ec.mu.RLock()
		currentCount := len(ec.events)
		ec.mu.RUnlock()

		if currentCount >= count {
			ec.mu.RLock()
			result := make([]messaging.Message, len(ec.events))
			copy(result, ec.events)
			ec.mu.RUnlock()
			return result
		}

		select {
		case <-ec.done:
			continue
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}

	ec.mu.RLock()
	result := make([]messaging.Message, len(ec.events))
	copy(result, ec.events)
	ec.mu.RUnlock()
	return result
}

func (ec *EventCapture) GetMessages() []messaging.Message {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]messaging.Message, len(ec.events))
	copy(result, ec.events)
	return result
}

func (ec *EventCapture) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.events = ec.events[:0]
}

func CreateTestConfig(natsURL string) *config.Config {
	return &config.Config{
		Services: config.ServicesConfig{
			Notification: config.NotificationServiceConfig{
				Host: "localhost",
				Port: 0, // Will be assigned dynamically
			},
			User: config.UserServiceConfig{
				Host: "localhost",
				Port: 8082,
			},
			Weather: config.WeatherServiceConfig{
				Host: "localhost",
				Port: 8081,
			},
			Subscription: config.SubscriptionServiceConfig{
				Host: "localhost",
				Port: 8083,
			},
		},
		MessageBroker: config.MessageBrokerConfig{
			URL: natsURL,
		},
		Email: config.EmailConfig{
			SMTPHost:     "localhost",
			SMTPPort:     1025,
			SMTPUsername: "test",
			SMTPPassword: "test",
			FromName:     "Weather API",
			FromAddress:  "noreply@weatherapi.app",
		},
	}
}

func WaitForCondition(condition func() bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("condition not met within timeout %v", timeout)
}
