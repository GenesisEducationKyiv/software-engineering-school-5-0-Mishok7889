package providers

import (
	"fmt"
	"time"

	"weatherapi.app/config"
	"weatherapi.app/models"
	"weatherapi.app/providers/cache"
)

type ProviderManager struct {
	primaryChain  WeatherProviderChain
	cache         CacheInterface
	logger        FileLogger
	configuration *ProviderConfiguration
}

type ProviderConfiguration struct {
	WeatherAPIKey     string
	WeatherAPIBaseURL string
	OpenWeatherMapKey string
	AccuWeatherKey    string
	CacheTTL          time.Duration
	LogFilePath       string
	EnableCache       bool
	EnableLogging     bool
	ProviderOrder     []string
}

func NewProviderManager(config *ProviderConfiguration) (*ProviderManager, error) {
	manager := &ProviderManager{
		configuration: config,
	}

	// Initialize components
	if err := manager.initializeComponents(); err != nil {
		return nil, fmt.Errorf("initialize provider manager: %w", err)
	}

	// Build the provider chain
	if err := manager.buildProviderChain(); err != nil {
		return nil, fmt.Errorf("build provider chain: %w", err)
	}

	return manager, nil
}

func (pm *ProviderManager) initializeComponents() error {
	if pm.configuration.EnableCache {
		pm.cache = cache.NewMemoryCache()
	}

	if pm.configuration.EnableLogging {
		logger, err := NewFileLogger(pm.configuration.LogFilePath)
		if err != nil {
			return fmt.Errorf("create logger: %w", err)
		}
		pm.logger = logger
	}

	return nil
}

func (pm *ProviderManager) buildProviderChain() error {
	providers := pm.createProviders()

	// Check if any providers are configured
	if len(providers) == 0 {
		// No providers configured - set chain to nil but don't return error
		// This allows provider manager to be created but will fail at GetWeather() level
		pm.primaryChain = nil
		return nil
	}

	chain := pm.buildChain(providers)
	if chain == nil {
		return fmt.Errorf("build provider chain")
	}

	if pm.configuration.EnableCache {
		chain = NewWeatherChainCacheProxy(chain, pm.cache, pm.configuration.CacheTTL)
	}

	pm.primaryChain = chain
	return nil
}

func (pm *ProviderManager) createProviders() map[string]WeatherProvider {
	providers := make(map[string]WeatherProvider)

	if pm.configuration.WeatherAPIKey != "" {
		baseURL := pm.configuration.WeatherAPIBaseURL
		if baseURL == "" {
			baseURL = "https://api.weatherapi.com/v1" // Default to production API
		}
		weatherConfig := &config.WeatherConfig{
			APIKey:  pm.configuration.WeatherAPIKey,
			BaseURL: baseURL,
		}
		var provider WeatherProvider = NewWeatherAPIProvider(weatherConfig)

		if pm.configuration.EnableLogging {
			provider = NewWeatherLoggerDecorator(provider, pm.logger, "WeatherAPI")
		}

		providers["weatherapi"] = provider
	}

	if pm.configuration.OpenWeatherMapKey != "" {
		var provider = NewOpenWeatherMapProvider(pm.configuration.OpenWeatherMapKey)

		if pm.configuration.EnableLogging {
			provider = NewWeatherLoggerDecorator(provider, pm.logger, "OpenWeatherMap")
		}

		providers["openweathermap"] = provider
	}

	if pm.configuration.AccuWeatherKey != "" {
		var provider = NewAccuWeatherProvider(pm.configuration.AccuWeatherKey)

		if pm.configuration.EnableLogging {
			provider = NewWeatherLoggerDecorator(provider, pm.logger, "AccuWeather")
		}

		providers["accuweather"] = provider
	}

	return providers
}

func (pm *ProviderManager) buildChain(providers map[string]WeatherProvider) WeatherProviderChain {
	builder := NewChainBuilder()

	for _, providerName := range pm.configuration.ProviderOrder {
		if provider, exists := providers[providerName]; exists {
			handler := pm.createHandler(providerName, provider)
			if handler != nil {
				builder.AddHandler(handler)
			}
		}
	}

	return builder.Build()
}

func (pm *ProviderManager) createHandler(providerName string, provider WeatherProvider) WeatherProviderChain {
	switch providerName {
	case "weatherapi":
		return NewWeatherAPIHandler(provider)
	case "openweathermap":
		return NewOpenWeatherMapHandler(provider)
	case "accuweather":
		return NewAccuWeatherHandler(provider)
	default:
		return nil
	}
}

func (pm *ProviderManager) GetWeather(city string) (*models.WeatherResponse, error) {
	if pm.primaryChain == nil {
		return nil, fmt.Errorf("no weather providers configured")
	}

	return pm.primaryChain.Handle(city)
}

func (pm *ProviderManager) GetProviderInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["cache_enabled"] = pm.configuration.EnableCache
	info["logging_enabled"] = pm.configuration.EnableLogging
	info["cache_ttl"] = pm.configuration.CacheTTL.String()
	info["provider_order"] = pm.configuration.ProviderOrder

	if pm.primaryChain != nil {
		info["chain_name"] = pm.primaryChain.GetProviderName()
	}

	return info
}

func DefaultProviderConfiguration() *ProviderConfiguration {
	return &ProviderConfiguration{
		CacheTTL:      10 * time.Minute,
		LogFilePath:   "logs/weather_providers.log",
		EnableCache:   true,
		EnableLogging: true,
		ProviderOrder: []string{"weatherapi", "openweathermap", "accuweather"},
	}
}

type ProviderManagerBuilder struct {
	config *ProviderConfiguration
}

func NewProviderManagerBuilder() *ProviderManagerBuilder {
	return &ProviderManagerBuilder{
		config: DefaultProviderConfiguration(),
	}
}

func (b *ProviderManagerBuilder) WithWeatherAPIKey(key string) *ProviderManagerBuilder {
	b.config.WeatherAPIKey = key
	return b
}

func (b *ProviderManagerBuilder) WithWeatherAPIBaseURL(baseURL string) *ProviderManagerBuilder {
	b.config.WeatherAPIBaseURL = baseURL
	return b
}

func (b *ProviderManagerBuilder) WithOpenWeatherMapKey(key string) *ProviderManagerBuilder {
	b.config.OpenWeatherMapKey = key
	return b
}

func (b *ProviderManagerBuilder) WithAccuWeatherKey(key string) *ProviderManagerBuilder {
	b.config.AccuWeatherKey = key
	return b
}

func (b *ProviderManagerBuilder) WithCacheTTL(ttl time.Duration) *ProviderManagerBuilder {
	b.config.CacheTTL = ttl
	return b
}

func (b *ProviderManagerBuilder) WithLogFilePath(path string) *ProviderManagerBuilder {
	b.config.LogFilePath = path
	return b
}

func (b *ProviderManagerBuilder) WithCacheEnabled(enabled bool) *ProviderManagerBuilder {
	b.config.EnableCache = enabled
	return b
}

func (b *ProviderManagerBuilder) WithLoggingEnabled(enabled bool) *ProviderManagerBuilder {
	b.config.EnableLogging = enabled
	return b
}

func (b *ProviderManagerBuilder) WithProviderOrder(order []string) *ProviderManagerBuilder {
	b.config.ProviderOrder = order
	return b
}

func (b *ProviderManagerBuilder) Build() (*ProviderManager, error) {
	return NewProviderManager(b.config)
}
