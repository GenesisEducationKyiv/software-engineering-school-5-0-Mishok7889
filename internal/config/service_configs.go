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

	// Validate only what Weather Service needs
	if err := config.Services.Weather.Validate(); err != nil {
		return nil, fmt.Errorf("weather service config: %w", err)
	}
	if err := config.Weather.Validate(); err != nil {
		return nil, fmt.Errorf("weather provider config: %w", err)
	}
	if err := config.Cache.Validate(); err != nil {
		return nil, fmt.Errorf("cache config: %w", err)
	}

	return config, nil
}

// LoadUserServiceConfig loads and validates config for User Service
func LoadUserServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Validate only what User Service needs
	if err := config.Services.User.Validate(); err != nil {
		return nil, fmt.Errorf("user service config: %w", err)
	}
	if err := config.UserDB.Validate(); err != nil {
		return nil, fmt.Errorf("user database config: %w", err)
	}

	return config, nil
}

// LoadSubscriptionServiceConfig loads and validates config for Subscription Service
func LoadSubscriptionServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Validate only what Subscription Service needs
	if err := config.Services.Subscription.Validate(); err != nil {
		return nil, fmt.Errorf("subscription service config: %w", err)
	}
	if err := config.Services.User.Validate(); err != nil {
		return nil, fmt.Errorf("user service client config: %w", err)
	}
	if err := config.SubscriptionDB.Validate(); err != nil {
		return nil, fmt.Errorf("subscription database config: %w", err)
	}
	if err := config.Email.Validate(); err != nil {
		return nil, fmt.Errorf("email config: %w", err)
	}
	if err := config.MessageBroker.Validate(); err != nil {
		return nil, fmt.Errorf("message broker config: %w", err)
	}

	return config, nil
}

// LoadNotificationServiceConfig loads and validates config for Notification Service
func LoadNotificationServiceConfig() (*Config, error) {
	config, err := loadBaseConfig()
	if err != nil {
		return nil, err
	}

	// Debug: print what was actually parsed for notification service
	fmt.Printf("DEBUG: Notification config parsed - Port: %d, Host: %s\n", config.Services.Notification.Port, config.Services.Notification.Host)
	fmt.Printf("DEBUG: Weather client config - Port: %d, Host: %s\n", config.Services.Weather.Port, config.Services.Weather.Host)
	fmt.Printf("DEBUG: User client config - Port: %d, Host: %s\n", config.Services.User.Port, config.Services.User.Host)
	fmt.Printf("DEBUG: Subscription client config - Port: %d, Host: %s\n", config.Services.Subscription.Port, config.Services.Subscription.Host)

	// Validate only what Notification Service needs
	if err := config.Services.Notification.Validate(); err != nil {
		return nil, fmt.Errorf("notification service config: %w", err)
	}
	if err := config.Services.User.Validate(); err != nil {
		return nil, fmt.Errorf("user service client config: %w", err)
	}
	if err := config.Services.Weather.Validate(); err != nil {
		return nil, fmt.Errorf("weather service client config: %w", err)
	}
	if err := config.Services.Subscription.Validate(); err != nil {
		return nil, fmt.Errorf("subscription service client config: %w", err)
	}
	if err := config.Email.Validate(); err != nil {
		return nil, fmt.Errorf("email config: %w", err)
	}
	if err := config.MessageBroker.Validate(); err != nil {
		return nil, fmt.Errorf("message broker config: %w", err)
	}

	return config, nil
}

// loadBaseConfig loads config without validation
func loadBaseConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing environment variables: %w", err)
	}
	return &config, nil
}
