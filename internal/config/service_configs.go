package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Service-specific configuration loading functions
// Each service only validates what it needs

// LoadWeatherServiceConfig loads and validates config for Weather Service
func LoadWeatherServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Set default port if not specified
	if config.Services.Weather.Port == 0 {
		config.Services.Weather.Port = defaultWeatherServicePort
	}

	// Validate only what Weather Service needs
	if err := config.Services.Weather.Validate(); err != nil {
		return nil, err
	}
	if err := config.Weather.Validate(); err != nil {
		return nil, err
	}
	if err := config.Cache.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadUserServiceConfig loads and validates config for User Service
func LoadUserServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Set default port if not specified
	if config.Services.User.Port == 0 {
		config.Services.User.Port = defaultUserServicePort
	}

	// Validate only what User Service needs
	if err := config.Services.User.Validate(); err != nil {
		return nil, err
	}
	if err := config.UserDB.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadSubscriptionServiceConfig loads and validates config for Subscription Service
func LoadSubscriptionServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Set default ports for all services subscription needs to connect to
	if config.Services.Subscription.Port == 0 {
		config.Services.Subscription.Port = defaultSubscriptionServicePort
	}
	if config.Services.User.Port == 0 {
		config.Services.User.Port = defaultUserServicePort
	}

	// Validate only what Subscription Service needs
	if err := config.Services.Subscription.Validate(); err != nil {
		return nil, err
	}
	if err := config.Services.User.Validate(); err != nil {
		return nil, err
	}
	if err := config.SubscriptionDB.Validate(); err != nil {
		return nil, err
	}
	if err := config.Email.Validate(); err != nil {
		return nil, err
	}
	if err := config.MessageBroker.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadNotificationServiceConfig loads and validates config for Notification Service
func LoadNotificationServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Set default ports for all services notification needs to connect to
	if config.Services.Notification.Port == 0 {
		config.Services.Notification.Port = defaultNotificationServicePort
	}
	if config.Services.User.Port == 0 {
		config.Services.User.Port = defaultUserServicePort
	}
	if config.Services.Weather.Port == 0 {
		config.Services.Weather.Port = defaultWeatherServicePort
	}
	if config.Services.Subscription.Port == 0 {
		config.Services.Subscription.Port = defaultSubscriptionServicePort
	}

	// Validate only what Notification Service needs
	if err := config.Services.Notification.Validate(); err != nil {
		return nil, err
	}
	if err := config.Services.User.Validate(); err != nil {
		return nil, err
	}
	if err := config.Services.Weather.Validate(); err != nil {
		return nil, err
	}
	if err := config.Services.Subscription.Validate(); err != nil {
		return nil, err
	}
	if err := config.Email.Validate(); err != nil {
		return nil, err
	}
	if err := config.MessageBroker.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadBaseConfig loads config without validation
func loadBaseConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}
	return &config, nil
}
