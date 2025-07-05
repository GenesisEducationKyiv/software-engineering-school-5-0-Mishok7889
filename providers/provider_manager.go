package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"weatherapi.app/config"
	"weatherapi.app/metrics"
	"weatherapi.app/models"
	"weatherapi.app/providers/cache"
)

type CacheType int

const (
	CacheTypeMemory CacheType = iota
	CacheTypeRedis
)

func (c CacheType) String() string {
	switch c {
	case CacheTypeMemory:
		return "memory"
	case CacheTypeRedis:
		return "redis"
	default:
		return "unknown"
	}
}

func CacheTypeFromString(s string) CacheType {
	switch s {
	case "memory":
		return CacheTypeMemory
	case "redis":
		return CacheTypeRedis
	default:
		return CacheTypeMemory
	}
}

type ProviderManagerOptions struct {
	Cache             Cache
	InstrumentedCache *InstrumentedCache
}

type ProviderManager struct {
	primaryChain      WeatherProviderChain
	cache             Cache
	instrumentedCache *InstrumentedCache
	logger            FileLogger
	configuration     *ProviderConfiguration
	cacheType         CacheType
}

type ProviderConfiguration struct {
	WeatherAPIKey         string
	WeatherAPIBaseURL     string
	OpenWeatherMapKey     string
	OpenWeatherMapBaseURL string
	AccuWeatherKey        string
	AccuWeatherBaseURL    string
	CacheTTL              time.Duration
	LogFilePath           string
	EnableLogging         bool
	ProviderOrder         []string
	CacheType             CacheType
	CacheConfig           *config.CacheConfig
}

func NewProviderManager(config *ProviderConfiguration, opts *ProviderManagerOptions) (*ProviderManager, error) {
	manager := &ProviderManager{
		configuration: config,
	}

	// Apply options if provided
	if opts != nil {
		manager.cache = opts.Cache
		manager.instrumentedCache = opts.InstrumentedCache
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
	if pm.instrumentedCache != nil {
		pm.cacheType = pm.configuration.CacheType
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

// Ensure ProviderManager implements both interfaces
var _ WeatherManager = (*ProviderManager)(nil)
var _ WeatherProviderMetrics = (*ProviderManager)(nil)

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

	baseURL := pm.configuration.OpenWeatherMapBaseURL
	if baseURL == "" {
		baseURL = "https://api.openweathermap.org/data/2.5"
	}

	var provider = NewOpenWeatherMapProvider(pm.configuration.OpenWeatherMapKey, baseURL)

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

	baseURL := pm.configuration.AccuWeatherBaseURL
	if baseURL == "" {
		baseURL = "http://dataservice.accuweather.com/currentconditions/v1"
	}

	var provider = NewAccuWeatherProvider(pm.configuration.AccuWeatherKey, baseURL)

	if pm.configuration.EnableLogging {
		provider = NewWeatherLoggerDecorator(provider, pm.logger, "AccuWeather")
	}

	return provider
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
	if pm.instrumentedCache != nil {
		return pm.getWeatherWithCache(city)
	}
	return pm.primaryChain.Handle(city)
}

func (pm *ProviderManager) getWeatherWithCache(city string) (*models.WeatherResponse, error) {
	cacheKey := pm.generateCacheKey(city)

	// Try cache first
	if cachedData, found := pm.instrumentedCache.Get(context.Background(), cacheKey); found {
		var weather models.WeatherResponse
		if err := json.Unmarshal(cachedData, &weather); err == nil {
			return &weather, nil
		}
	}

	// Cache miss - get from provider chain
	response, err := pm.primaryChain.Handle(city)
	if err != nil {
		return nil, err
	}

	// Cache the response
	if data, err := json.Marshal(response); err == nil {
		pm.instrumentedCache.Set(context.Background(), cacheKey, data, pm.configuration.CacheTTL)
	}

	return response, nil
}

func (pm *ProviderManager) generateCacheKey(city string) string {
	return fmt.Sprintf("weather:%s", strings.ToLower(strings.TrimSpace(city)))
}

func (pm *ProviderManager) GetProviderInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["cache_enabled"] = pm.instrumentedCache != nil
	if pm.instrumentedCache != nil {
		info["cache_type"] = pm.cacheType.String()
		info["cache_ttl"] = pm.configuration.CacheTTL.String()
	}
	info["logging_enabled"] = pm.configuration.EnableLogging
	info["provider_order"] = pm.configuration.ProviderOrder
	info["chain_name"] = pm.primaryChain.GetProviderName()

	return info
}

func (pm *ProviderManager) GetCacheMetrics() (metrics.CacheStats, error) {
	if pm.instrumentedCache == nil {
		return metrics.CacheStats{}, fmt.Errorf("cache not enabled")
	}
	return pm.instrumentedCache.GetMetrics().GetStats(), nil
}

func DefaultProviderConfiguration() *ProviderConfiguration {
	return &ProviderConfiguration{
		CacheTTL:      10 * time.Minute,
		LogFilePath:   "logs/weather_providers.log",
		EnableLogging: true,
		ProviderOrder: []string{"weatherapi", "openweathermap", "accuweather"},
		CacheType:     CacheTypeMemory,
		CacheConfig:   &config.CacheConfig{Type: CacheTypeMemory.String()},
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

func (b *ProviderManagerBuilder) WithOpenWeatherMapBaseURL(baseURL string) *ProviderManagerBuilder {
	b.config.OpenWeatherMapBaseURL = baseURL
	return b
}

func (b *ProviderManagerBuilder) WithAccuWeatherKey(key string) *ProviderManagerBuilder {
	b.config.AccuWeatherKey = key
	return b
}

func (b *ProviderManagerBuilder) WithAccuWeatherBaseURL(baseURL string) *ProviderManagerBuilder {
	b.config.AccuWeatherBaseURL = baseURL
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

func (b *ProviderManagerBuilder) WithLoggingEnabled(enabled bool) *ProviderManagerBuilder {
	b.config.EnableLogging = enabled
	return b
}

func (b *ProviderManagerBuilder) WithProviderOrder(order []string) *ProviderManagerBuilder {
	b.config.ProviderOrder = order
	return b
}

func (b *ProviderManagerBuilder) WithCacheType(cacheType CacheType) *ProviderManagerBuilder {
	b.config.CacheType = cacheType
	return b
}

func (b *ProviderManagerBuilder) WithCacheConfig(cacheConfig *config.CacheConfig) *ProviderManagerBuilder {
	b.config.CacheConfig = cacheConfig
	return b
}

func (b *ProviderManagerBuilder) Build() (*ProviderManager, error) {
	if err := b.validate(); err != nil {
		return nil, fmt.Errorf("provider manager configuration validation failed: %w", err)
	}

	// Create options
	var opts *ProviderManagerOptions

	if b.config.CacheConfig != nil {
		genericCache, err := b.createGenericCache()
		if err != nil {
			return nil, fmt.Errorf("create cache: %w", err)
		}
		instrumentedCache := NewInstrumentedCache(genericCache, b.config.CacheType.String())
		weatherCache := cache.NewWeatherCache(instrumentedCache)

		opts = &ProviderManagerOptions{
			Cache:             weatherCache,
			InstrumentedCache: instrumentedCache,
		}
	}

	return NewProviderManager(b.config, opts)
}

func (b *ProviderManagerBuilder) createGenericCache() (cache.GenericCache, error) {
	switch b.config.CacheType {
	case CacheTypeMemory:
		slog.Info("Creating memory cache")
		return cache.NewMemoryCache(), nil
	case CacheTypeRedis:
		slog.Info("Creating Redis cache", "addr", b.config.CacheConfig.Redis.Addr)
		redisConfig := &cache.RedisCacheConfig{
			Addr:         b.config.CacheConfig.Redis.Addr,
			Password:     b.config.CacheConfig.Redis.Password,
			DB:           b.config.CacheConfig.Redis.DB,
			DialTimeout:  time.Duration(b.config.CacheConfig.Redis.DialTimeout) * time.Second,
			ReadTimeout:  time.Duration(b.config.CacheConfig.Redis.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(b.config.CacheConfig.Redis.WriteTimeout) * time.Second,
		}
		return cache.NewRedisCache(redisConfig)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", b.config.CacheType)
	}
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
