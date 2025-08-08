package consumers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeatherEventConsumer_New(t *testing.T) {
	// Simple test to check if the consumer can be created
	consumer := NewWeatherEventConsumer(WeatherEventConsumerConfig{})
	assert.NotNil(t, consumer)
}

func TestSubscriptionEventConsumer_New(t *testing.T) {
	// Simple test to check if the consumer can be created
	consumer := NewSubscriptionEventConsumer(SubscriptionEventConsumerConfig{})
	assert.NotNil(t, consumer)
}

func TestCommandConsumer_New(t *testing.T) {
	// Simple test to check if the consumer can be created
	consumer := NewCommandConsumer(CommandConsumerConfig{})
	assert.NotNil(t, consumer)
}
