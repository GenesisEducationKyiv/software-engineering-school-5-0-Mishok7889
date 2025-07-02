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

	// Fail fast if no providers are configured
	if len(providers) == 0 {
		return fmt.Errorf("no weather providers configured - at least one API key must be provided (WEATHER_API_KEY, OPENWEATHERMAP_API_KEY, or ACCUWEATHER_API_KEY)")
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

	if weatherProvider := pm.createWeatherAPIProvider(); weatherProvider != nil {
		providers["weatherapi"] = weatherProvider
	}

	if openWeatherProvider := pm.createOpenWeatherProvider(); openWeatherProvider != nil {
		providers["openweathermap"] = openWeatherProvider
	}

	if accuWeatherProvider := pm.createAccuWeatherProvider(); accuWeatherProvider != nil {
		providers["accuweather"] = accuWeatherProvider
	}

	return providers
}

// createWeatherAPIProvider creates and configures WeatherAPI provider if API key is provided
func (pm *ProviderManager) createWeatherAPIProvider() WeatherProvider {
	if pm.configuration.WeatherAPIKey == "" {
		return nil
	}

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

	return provider
}

// createOpenWeatherProvider creates and configures OpenWeatherMap provider if API key is provided
func (pm *ProviderManager) createOpenWeatherProvider() WeatherProvider {
	if pm.configuration.OpenWeatherMapKey == "" {
		return nil
	}

	var provider = NewOpenWeatherMapProvider(pm.configuration.OpenWeatherMapKey)

	if pm.configuration.EnableLogging {
		provider = NewWeatherLoggerDecorator(provider, pm.logger, "OpenWeatherMap")
	}

	return provider
}

// createAccuWeatherProvider creates and configures AccuWeather provider if API key is provided
func (pm *ProviderManager) createAccuWeatherProvider() WeatherProvider {
	if pm.configuration.AccuWeatherKey == "" {
		return nil
	}

	var provider = NewAccuWeatherProvider(pm.configuration.AccuWeatherKey)

	if pm.configuration.EnableLogging {
		provider = NewWeatherLoggerDecorator(provider, pm.logger, "AccuWeather")
	}

	return provider
}

func (pm *ProviderManager) buildChain(providers map[string]WeatherProvider) WeatherProviderChain {
	builder := NewChainBuilder()

	for _, providerName := range pm.configuration.ProviderOrder {
		provider, exists := providers[providerName]
		if !exists {
			continue
		}

		handler := pm.createHandler(providerName, provider)
		if handler != nil {
			builder.AddHandler(handler)
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
	return pm.primaryChain.Handle(city)
}

func (pm *ProviderManager) GetProviderInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["cache_enabled"] = pm.configuration.EnableCache
	info["logging_enabled"] = pm.configuration.EnableLogging
	info["cache_ttl"] = pm.configuration.CacheTTL.String()
	info["provider_order"] = pm.configuration.ProviderOrder
	info["chain_name"] = pm.primaryChain.GetProviderName()

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
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("provider manager configuration validation failed: %w", err)
	}
	return NewProviderManager(b.config)
}

// validate ensures the builder configuration is valid before building
func (b *ProviderManagerBuilder) validate() error {
	// At least one weather provider must be configured
	if b.config.WeatherAPIKey == "" && b.config.OpenWeatherMapKey == "" && b.config.AccuWeatherKey == "" {
		return fmt.Errorf("at least one weather provider API key must be configured")
	}

	// Validate WeatherAPI configuration if provided
	if b.config.WeatherAPIKey != "" && b.config.WeatherAPIBaseURL == "" {
		return fmt.Errorf("WeatherAPI base URL is required when API key is provided")
	}

	// Validate cache TTL
	if b.config.CacheTTL <= 0 {
		return fmt.Errorf("cache TTL must be positive")
	}

	// Validate log file path if logging is enabled
	if b.config.EnableLogging && b.config.LogFilePath == "" {
		return fmt.Errorf("log file path is required when logging is enabled")
	}

	// Validate provider order contains valid providers
	validProviders := map[string]bool{
		"weatherapi":     true,
		"openweathermap": true,
		"accuweather":    true,
	}

	for _, provider := range b.config.ProviderOrder {
		if !validProviders[provider] {
			return fmt.Errorf("invalid weather provider in order: %s", provider)
		}
	}

	return nil
}
