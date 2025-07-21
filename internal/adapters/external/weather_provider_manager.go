package external

import (
	"context"
	"fmt"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
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
func NewWeatherProviderManagerAdapter(config ProviderManagerConfig) (ports.WeatherProviderManager, error) {
	manager := &WeatherProviderManagerAdapter{
		providers: []ports.WeatherProvider{},
		logger:    config.Logger,
	}

	// Create providers in configured order
	providerMap, err := manager.createProviderMap(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider map: %w", err)
	}

	for _, providerName := range config.ProviderOrder {
		if provider, exists := providerMap[Provider(providerName)]; exists {
			manager.providers = append(manager.providers, provider)
		}
	}

	// If no order specified, add available providers in default order
	if len(manager.providers) == 0 {
		for _, provider := range providerMap {
			manager.providers = append(manager.providers, provider)
		}
	}

	return manager, nil
}

func (m *WeatherProviderManagerAdapter) createProviderMap(config ProviderManagerConfig) (map[Provider]ports.WeatherProvider, error) {
	providers := make(map[Provider]ports.WeatherProvider)

	// WeatherAPI provider
	if config.WeatherAPIKey != "" {
		provider, err := NewWeatherAPIProviderAdapter(WeatherAPIProviderParams{
			APIKey:  config.WeatherAPIKey,
			BaseURL: config.WeatherAPIBaseURL,
			Logger:  m.logger,
		})
		if err != nil {
			if m.logger != nil {
				m.logger.Error(ProviderValidationFailedMsg, logField(ProviderField, WeatherAPI), logField(ErrorField, err))
			}
			return nil, fmt.Errorf("failed to create WeatherAPI provider: %w", err)
		}
		providers[WeatherAPI] = provider
		if m.logger != nil {
			m.logger.Debug(CreatedProviderMsg, logField(ProviderField, WeatherAPI))
		}
	}

	// OpenWeatherMap provider
	if config.OpenWeatherKey != "" {
		provider, err := NewOpenWeatherMapProviderAdapter(OpenWeatherMapProviderParams{
			APIKey:  config.OpenWeatherKey,
			BaseURL: config.OpenWeatherURL,
			Logger:  m.logger,
		})
		if err != nil {
			if m.logger != nil {
				m.logger.Error(ProviderValidationFailedMsg, logField(ProviderField, OpenWeatherMap), logField(ErrorField, err))
			}
			return nil, fmt.Errorf("failed to create OpenWeatherMap provider: %w", err)
		}
		providers[OpenWeatherMap] = provider
		if m.logger != nil {
			m.logger.Debug(CreatedProviderMsg, logField(ProviderField, OpenWeatherMap))
		}
	}

	// AccuWeather provider
	if config.AccuWeatherKey != "" {
		provider, err := NewAccuWeatherProviderAdapter(AccuWeatherProviderParams{
			APIKey:  config.AccuWeatherKey,
			BaseURL: config.AccuWeatherURL,
			Logger:  m.logger,
		})
		if err != nil {
			if m.logger != nil {
				m.logger.Error(ProviderValidationFailedMsg, logField(ProviderField, AccuWeather), logField(ErrorField, err))
			}
			return nil, fmt.Errorf("failed to create AccuWeather provider: %w", err)
		}
		providers[AccuWeather] = provider
		if m.logger != nil {
			m.logger.Debug(CreatedProviderMsg, logField(ProviderField, AccuWeather))
		}
	}

	return providers, nil
}

// GetWeather implements Chain of Responsibility - tries each provider until one succeeds
func (m *WeatherProviderManagerAdapter) GetWeather(ctx context.Context, city string) (*ports.WeatherData, error) {
	if len(m.providers) == 0 {
		return nil, infrastructure.NewExternalAPIError(NoProvidersConfiguredMsg, nil)
	}

	var lastErr error
	var notFoundErrors []error

	for i, provider := range m.providers {
		providerName := provider.GetProviderName()

		if m.logger != nil {
			m.logger.Debug(TryingProviderMsg,
				logField(ProviderField, providerName),
				logField(AttemptField, i+1),
				logField(CityField, city))
		}

		weather, err := provider.GetCurrentWeather(ctx, city)
		if err == nil {
			if m.logger != nil {
				m.logger.Debug(ProviderSucceededMsg,
					logField(ProviderField, providerName),
					logField(CityField, city),
					logField(TemperatureField, weather.Temperature))
			}
			return weather, nil
		}

		if ports.IsNotFoundError(err) {
			notFoundErrors = append(notFoundErrors, err)
		}

		lastErr = err
		if m.logger != nil {
			m.logger.Warn(ProviderFailedMsg,
				logField(ProviderField, providerName),
				logField(ErrorField, err.Error()),
				logField(CityField, city))
		}
	}

	if m.logger != nil {
		m.logger.Error(AllProvidersFailedMsg,
			logField(CityField, city),
			logField(ProvidersTriedField, len(m.providers)),
			logField(LastErrorField, lastErr.Error()))
	}

	if len(notFoundErrors) == len(m.providers) {
		return nil, ports.NewNotFoundError(CityNotFoundMsg)
	}

	return nil, fmt.Errorf("all weather providers failed (tried %d providers): %w", len(m.providers), lastErr)
}

// GetProviderInfo returns information about configured providers
func (m *WeatherProviderManagerAdapter) GetProviderInfo() ports.ProviderInfo {
	providerNames := make([]string, len(m.providers))
	for i, provider := range m.providers {
		providerNames[i] = provider.GetProviderName()
	}

	return NewProviderInfo(providerNames)
}
