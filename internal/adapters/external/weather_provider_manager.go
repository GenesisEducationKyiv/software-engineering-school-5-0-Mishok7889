package external

import (
	"context"
	"fmt"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// WeatherProviderManagerAdapter implements Chain of Responsibility pattern for weather providers
// This adapter manages multiple weather providers and implements automatic failover
type WeatherProviderManagerAdapter struct {
	providers []ports.WeatherProvider
	logger    ports.Logger
}

// ProviderManagerConfig holds configuration for creating the provider manager
type ProviderManagerConfig struct {
	WeatherAPIKey     string
	WeatherAPIBaseURL string
	OpenWeatherKey    string
	OpenWeatherURL    string
	AccuWeatherKey    string
	AccuWeatherURL    string
	ProviderOrder     []string
	Logger            ports.Logger
}

// NewWeatherProviderManagerAdapter creates a new weather provider manager with Chain of Responsibility
func NewWeatherProviderManagerAdapter(config ProviderManagerConfig) ports.WeatherProviderManager {
	manager := &WeatherProviderManagerAdapter{
		providers: []ports.WeatherProvider{},
		logger:    config.Logger,
	}

	// Create providers in configured order
	providerMap := manager.createProviderMap(config)

	for _, providerName := range config.ProviderOrder {
		if provider, exists := providerMap[providerName]; exists {
			manager.providers = append(manager.providers, provider)
		}
	}

	// If no order specified, add available providers in default order
	if len(manager.providers) == 0 {
		for _, provider := range providerMap {
			manager.providers = append(manager.providers, provider)
		}
	}

	return manager
}

func (m *WeatherProviderManagerAdapter) createProviderMap(config ProviderManagerConfig) map[string]ports.WeatherProvider {
	providers := make(map[string]ports.WeatherProvider)

	// WeatherAPI provider
	if config.WeatherAPIKey != "" {
		providers["weatherapi"] = NewWeatherAPIProviderAdapter(WeatherAPIProviderParams{
			APIKey:  config.WeatherAPIKey,
			BaseURL: config.WeatherAPIBaseURL,
			Logger:  m.logger,
		})
		if m.logger != nil {
			m.logger.Debug("Created WeatherAPI provider", ports.F("provider", "weatherapi"))
		}
	}

	// OpenWeatherMap provider
	if config.OpenWeatherKey != "" {
		providers["openweathermap"] = NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
			APIKey:  config.OpenWeatherKey,
			BaseURL: config.OpenWeatherURL,
			Logger:  m.logger,
		})
		if m.logger != nil {
			m.logger.Debug("Created OpenWeatherMap provider", ports.F("provider", "openweathermap"))
		}
	}

	// AccuWeather provider
	if config.AccuWeatherKey != "" {
		providers["accuweather"] = NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
			APIKey:  config.AccuWeatherKey,
			BaseURL: config.AccuWeatherURL,
			Logger:  m.logger,
		})
		if m.logger != nil {
			m.logger.Debug("Created AccuWeather provider", ports.F("provider", "accuweather"))
		}
	}

	return providers
}

// GetWeather implements Chain of Responsibility - tries each provider until one succeeds
func (m *WeatherProviderManagerAdapter) GetWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if len(m.providers) == 0 {
		return nil, errors.NewExternalAPIError("no weather providers configured", nil)
	}

	var lastErr error

	// Chain of Responsibility: try each provider in order
	for i, provider := range m.providers {
		providerName := provider.GetProviderName()

		if m.logger != nil {
			m.logger.Debug("Trying weather provider",
				ports.F("provider", providerName),
				ports.F("attempt", i+1),
				ports.F("city", city))
		}

		weather, err := provider.GetCurrentWeather(ctx, city)
		if err == nil {
			if m.logger != nil {
				m.logger.Debug("Weather provider succeeded",
					ports.F("provider", providerName),
					ports.F("city", city),
					ports.F("temperature", weather.Temperature))
			}
			return weather, nil
		}

		lastErr = err
		if m.logger != nil {
			m.logger.Warn("Weather provider failed, trying next",
				ports.F("provider", providerName),
				ports.F("error", err.Error()),
				ports.F("city", city))
		}
	}

	// All providers failed
	if m.logger != nil {
		m.logger.Error("All weather providers failed",
			ports.F("city", city),
			ports.F("providers_tried", len(m.providers)),
			ports.F("last_error", lastErr.Error()))
	}

	return nil, fmt.Errorf("all weather providers failed (tried %d providers): %w", len(m.providers), lastErr)
}

// GetProviderInfo returns information about configured providers
func (m *WeatherProviderManagerAdapter) GetProviderInfo() map[string]interface{} {
	providerNames := make([]string, len(m.providers))
	for i, provider := range m.providers {
		providerNames[i] = provider.GetProviderName()
	}

	return map[string]interface{}{
		"total_providers":  len(m.providers),
		"provider_order":   providerNames,
		"chain_enabled":    true,
		"fallback_enabled": len(m.providers) > 1,
	}
}
