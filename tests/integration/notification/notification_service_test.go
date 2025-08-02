package notification

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/core/shared"
	"weatherapi.app/internal/ports"
	"weatherapi.app/internal/ports/messaging"
	"weatherapi.app/tests/integration/helpers"
)

type NotificationServiceIntegrationSuite struct {
	suite.Suite
	config       *config.Config
	broker       messaging.MessageBroker
	testBroker   *helpers.TestBroker
	logger       ports.Logger
	eventCapture *helpers.EventCapture
}

func (s *NotificationServiceIntegrationSuite) SetupSuite() {
	s.logger = &infrastructure.SlogLoggerAdapter{}

	// Setup event capture
	s.eventCapture = helpers.NewEventCapture()

	// Setup embedded NATS
	s.testBroker = helpers.NewTestBroker(s.T(), s.logger)
	var natsURL string
	s.broker, natsURL = s.testBroker.Setup()

	// Create test configuration
	s.config = helpers.CreateTestConfig(natsURL)
}

func (s *NotificationServiceIntegrationSuite) SetupTest() {
	// Create fresh event capture for each test to ensure isolation
	s.eventCapture = helpers.NewEventCapture()
}

func (s *NotificationServiceIntegrationSuite) TearDownSuite() {
	if s.testBroker != nil {
		s.testBroker.Cleanup()
	}
}

func (s *NotificationServiceIntegrationSuite) TestWeatherEventProcessing_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture to monitor message processing
	err := s.broker.Subscribe(ctx, string(shared.EventTypeWeatherDataFetched), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create and publish test event
	weatherEvent := shared.NewWeatherDataFetchedEvent("London", 22.5, 65.0, "Sunny")
	err = s.broker.Publish(ctx, weatherEvent.Topic(), weatherEvent.Data())
	require.NoError(t, err)

	// Wait for message to be captured
	messages := s.eventCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify message content
	msg := messages[0]
	assert.Equal(t, string(shared.EventTypeWeatherDataFetched), msg.Topic)
	assert.Contains(t, string(msg.Data), "London")
	assert.Contains(t, string(msg.Data), "22.5")
	assert.Contains(t, string(msg.Data), "Sunny")

	// Verify headers are present
	assert.NotEmpty(t, msg.Headers)
	if eventID, exists := msg.Headers["event_id"]; exists {
		assert.NotEmpty(t, eventID)
	}
	if eventType, exists := msg.Headers["event_type"]; exists {
		assert.Equal(t, string(shared.EventTypeWeatherDataFetched), eventType)
	}
}

func (s *NotificationServiceIntegrationSuite) TestSubscriptionEventProcessing_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture
	err := s.broker.Subscribe(ctx, string(shared.EventTypeSubscriptionCreated), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create and publish test event
	subscriptionEvent := shared.NewSubscriptionCreatedEvent(
		"sub-123",
		"user-456",
		"London",
		"daily",
		"test@example.com",
	)

	err = s.broker.Publish(ctx, subscriptionEvent.Topic(), subscriptionEvent.Data())
	require.NoError(t, err)

	// Wait for message to be captured
	messages := s.eventCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify message content
	msg := messages[0]
	assert.Equal(t, string(shared.EventTypeSubscriptionCreated), msg.Topic)
	assert.Contains(t, string(msg.Data), "test@example.com")
	assert.Contains(t, string(msg.Data), "sub-123")
	assert.Contains(t, string(msg.Data), "daily")

	// Verify headers
	assert.NotEmpty(t, msg.Headers)
	if eventType, exists := msg.Headers["event_type"]; exists {
		assert.Equal(t, string(shared.EventTypeSubscriptionCreated), eventType)
	}
}

func (s *NotificationServiceIntegrationSuite) TestCommandProcessing_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture
	err := s.broker.Subscribe(ctx, string(shared.CommandTypeSendWeatherUpdates), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create and publish test command
	command := shared.NewSendWeatherUpdatesCommand("daily")
	err = s.broker.Publish(ctx, command.Topic(), command.Data())
	require.NoError(t, err)

	// Wait for message to be captured
	messages := s.eventCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify command content
	msg := messages[0]
	assert.Equal(t, string(shared.CommandTypeSendWeatherUpdates), msg.Topic)
	assert.Contains(t, string(msg.Data), "daily")

	// Verify headers
	assert.NotEmpty(t, msg.Headers)
	if commandType, exists := msg.Headers["command_type"]; exists {
		assert.Equal(t, string(shared.CommandTypeSendWeatherUpdates), commandType)
	}
}

func (s *NotificationServiceIntegrationSuite) TestEventProcessingWithNoSubscriptions_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture
	err := s.broker.Subscribe(ctx, string(shared.EventTypeWeatherDataFetched), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create test event for city with no subscriptions
	weatherEvent := shared.NewWeatherDataFetchedEvent("UnknownCity", 0, 0, "Test")
	err = s.broker.Publish(ctx, weatherEvent.Topic(), weatherEvent.Data())
	require.NoError(t, err)

	// Wait for message to be captured
	messages := s.eventCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify that event was received correctly
	msg := messages[0]
	assert.Equal(t, string(shared.EventTypeWeatherDataFetched), msg.Topic)
	assert.Contains(t, string(msg.Data), "UnknownCity")

	// Verify headers
	assert.NotEmpty(t, msg.Headers)
}

func (s *NotificationServiceIntegrationSuite) TestMultipleEventProcessing_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture
	err := s.broker.Subscribe(ctx, string(shared.EventTypeWeatherDataFetched), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create multiple events
	events := []*shared.WeatherDataFetchedEvent{
		shared.NewWeatherDataFetchedEvent("London", 22.5, 65.0, "Sunny"),
		shared.NewWeatherDataFetchedEvent("Paris", 18.0, 70.0, "Cloudy"),
		shared.NewWeatherDataFetchedEvent("Berlin", 15.0, 80.0, "Rainy"),
	}

	// Publish all events
	for _, event := range events {
		err := s.broker.Publish(ctx, event.Topic(), event.Data())
		require.NoError(t, err)
	}

	// Wait for all messages to be captured
	messages := s.eventCapture.WaitForMessages(3, 3*time.Second)
	require.Len(t, messages, 3)

	// Verify that all events were received
	cities := make(map[string]bool)
	for _, msg := range messages {
		assert.Equal(t, string(shared.EventTypeWeatherDataFetched), msg.Topic)
		// Extract city from message data
		if strings.Contains(string(msg.Data), "London") {
			cities["London"] = true
		} else if strings.Contains(string(msg.Data), "Paris") {
			cities["Paris"] = true
		} else if strings.Contains(string(msg.Data), "Berlin") {
			cities["Berlin"] = true
		}
		// Verify headers exist
		assert.NotEmpty(t, msg.Headers)
	}

	assert.Len(t, cities, 3, "All three cities should be present")
	assert.True(t, cities["London"], "London should be present")
	assert.True(t, cities["Paris"], "Paris should be present")
	assert.True(t, cities["Berlin"], "Berlin should be present")
}

func (s *NotificationServiceIntegrationSuite) TestMessageBrokerConnectivity_Success() {
	t := s.T()
	ctx := context.Background()

	// Test broker connectivity by publishing and consuming a simple message
	testMessage := "test-message"
	testTopic := "test-topic"

	// Setup capture for test topic
	testCapture := helpers.NewEventCapture()
	err := s.broker.Subscribe(ctx, testTopic, testCapture.CaptureMessage)
	require.NoError(t, err)

	// Publish test message
	err = s.broker.Publish(ctx, testTopic, []byte(testMessage))
	require.NoError(t, err)

	// Wait for message
	messages := testCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify message content
	assert.Equal(t, testTopic, messages[0].Topic)
	assert.Equal(t, testMessage, string(messages[0].Data))
}

func (s *NotificationServiceIntegrationSuite) TestEventDataValidation_Success() {
	t := s.T()
	ctx := context.Background()

	// Setup event capture
	err := s.broker.Subscribe(ctx, string(shared.EventTypeWeatherDataFetched), s.eventCapture.CaptureMessage)
	require.NoError(t, err)

	// Create event with various data types
	weatherEvent := shared.NewWeatherDataFetchedEvent("Test City", 25.5, 80.0, "Test Description")
	err = s.broker.Publish(ctx, weatherEvent.Topic(), weatherEvent.Data())
	require.NoError(t, err)

	// Wait for message
	messages := s.eventCapture.WaitForMessages(1, 3*time.Second)
	require.Len(t, messages, 1)

	// Verify event data integrity
	msg := messages[0]
	assert.Equal(t, string(shared.EventTypeWeatherDataFetched), msg.Topic)
	assert.Contains(t, string(msg.Data), "Test City")
	assert.Contains(t, string(msg.Data), "25.5")
	assert.Contains(t, string(msg.Data), "80")
	assert.Contains(t, string(msg.Data), "Test Description")

	// Verify headers
	assert.NotEmpty(t, msg.Headers)
	if eventID, exists := msg.Headers["event_id"]; exists {
		assert.NotEmpty(t, eventID)
	}
}

func TestNotificationServiceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceIntegrationSuite))
}
