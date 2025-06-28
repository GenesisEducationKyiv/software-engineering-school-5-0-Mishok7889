package providers

import (
	"fmt"
	"log/slog"
	"time"

	"weatherapi.app/models"
)

type WeatherCacheProxy struct {
	realProvider WeatherProvider
	cache        CacheInterface
	cacheTTL     time.Duration
}

func NewWeatherCacheProxy(realProvider WeatherProvider, cache CacheInterface, cacheTTL time.Duration) WeatherProvider {
	return &WeatherCacheProxy{
		realProvider: realProvider,
		cache:        cache,
		cacheTTL:     cacheTTL,
	}
}

func (p *WeatherCacheProxy) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	cacheKey := p.generateCacheKey(city)

	if cachedResponse, found := p.cache.Get(cacheKey); found {
		slog.Info("cache hit", "city", city)
		return cachedResponse, nil
	}

	slog.Info("cache miss", "city", city)

	response, err := p.realProvider.GetCurrentWeather(city)
	if err != nil {
		return nil, err
	}

	p.cache.Set(cacheKey, response, p.cacheTTL)

	return response, nil
}

func (p *WeatherCacheProxy) generateCacheKey(city string) string {
	return fmt.Sprintf("weather:%s", city)
}

type WeatherChainCacheProxy struct {
	realChain WeatherProviderChain
	cache     CacheInterface
	cacheTTL  time.Duration
}

// NewWeatherChainCacheProxy creates a caching proxy for chain of providers
func NewWeatherChainCacheProxy(realChain WeatherProviderChain, cache CacheInterface, cacheTTL time.Duration) WeatherProviderChain {
	return &WeatherChainCacheProxy{
		realChain: realChain,
		cache:     cache,
		cacheTTL:  cacheTTL,
	}
}

// Handle implements caching for the chain of responsibility
func (p *WeatherChainCacheProxy) Handle(city string) (*models.WeatherResponse, error) {
	cacheKey := p.generateCacheKey(city)

	if cachedResponse, found := p.cache.Get(cacheKey); found {
		slog.Info("chain cache hit", "city", city)
		return cachedResponse, nil
	}

	slog.Info("chain cache miss", "city", city)

	response, err := p.realChain.Handle(city)
	if err != nil {
		return nil, err
	}

	p.cache.Set(cacheKey, response, p.cacheTTL)

	return response, nil
}

// SetNext delegates to the real chain
func (p *WeatherChainCacheProxy) SetNext(handler WeatherProviderChain) {
	p.realChain.SetNext(handler)
}

// GetProviderName returns a descriptive name for the cached chain
func (p *WeatherChainCacheProxy) GetProviderName() string {
	return fmt.Sprintf("Cached(%s)", p.realChain.GetProviderName())
}

// generateCacheKey creates a consistent cache key for a city
func (p *WeatherChainCacheProxy) generateCacheKey(city string) string {
	return fmt.Sprintf("weather:%s", city)
}

type CacheStats struct {
	Hits   int64
	Misses int64
	Total  int64
}

type WeatherCacheProxyWithStats struct {
	*WeatherCacheProxy
	stats CacheStats
}

// NewWeatherCacheProxyWithStats creates a caching proxy with statistics
func NewWeatherCacheProxyWithStats(realProvider WeatherProvider, cache CacheInterface, cacheTTL time.Duration) *WeatherCacheProxyWithStats {
	return &WeatherCacheProxyWithStats{
		WeatherCacheProxy: &WeatherCacheProxy{
			realProvider: realProvider,
			cache:        cache,
			cacheTTL:     cacheTTL,
		},
	}
}

// GetCurrentWeather implements caching with statistics tracking
func (p *WeatherCacheProxyWithStats) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	p.stats.Total++

	cacheKey := p.generateCacheKey(city)

	if cachedResponse, found := p.cache.Get(cacheKey); found {
		p.stats.Hits++
		slog.Info("cache hit with stats", "city", city, "hit_rate_percent", float64(p.stats.Hits)/float64(p.stats.Total)*100)
		return cachedResponse, nil
	}

	p.stats.Misses++
	slog.Info("cache miss with stats", "city", city, "hit_rate_percent", float64(p.stats.Hits)/float64(p.stats.Total)*100)

	response, err := p.realProvider.GetCurrentWeather(city)
	if err != nil {
		return nil, err
	}

	p.cache.Set(cacheKey, response, p.cacheTTL)
	return response, nil
}

// GetStats returns current cache statistics
func (p *WeatherCacheProxyWithStats) GetStats() CacheStats {
	return p.stats
}
