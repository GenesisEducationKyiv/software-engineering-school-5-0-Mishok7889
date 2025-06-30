package providers

import (
	"fmt"
	"log/slog"
	"time"

	"weatherapi.app/metrics"
	"weatherapi.app/models"
)

type InstrumentedCacheProxy struct {
	realProvider WeatherProvider
	cache        CacheInterface
	cacheTTL     time.Duration
	metrics      *metrics.CacheMetrics
}

func NewInstrumentedCacheProxy(realProvider WeatherProvider, cache CacheInterface, cacheTTL time.Duration, cacheType string) WeatherProvider {
	return &InstrumentedCacheProxy{
		realProvider: realProvider,
		cache:        cache,
		cacheTTL:     cacheTTL,
		metrics:      metrics.NewCacheMetrics(cacheType),
	}
}

func (p *InstrumentedCacheProxy) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	start := time.Now()
	cacheKey := p.generateCacheKey(city)

	cachedResponse, found := p.cache.Get(cacheKey)
	getLatency := time.Since(start).Seconds()
	p.metrics.RecordLatency("get", getLatency)

	if found {
		p.metrics.RecordHit()
		slog.Debug("cache hit", "city", city, "latency_ms", getLatency*1000)
		return cachedResponse, nil
	}

	p.metrics.RecordMiss()
	slog.Debug("cache miss", "city", city, "latency_ms", getLatency*1000)

	providerStart := time.Now()
	response, err := p.realProvider.GetCurrentWeather(city)
	providerLatency := time.Since(providerStart).Seconds()

	if err != nil {
		return nil, err
	}

	setStart := time.Now()
	p.cache.Set(cacheKey, response, p.cacheTTL)
	setLatency := time.Since(setStart).Seconds()
	p.metrics.RecordLatency("set", setLatency)

	slog.Debug("cache set", "city", city, "provider_latency_ms", providerLatency*1000, "set_latency_ms", setLatency*1000)
	return response, nil
}

func (p *InstrumentedCacheProxy) generateCacheKey(city string) string {
	return fmt.Sprintf("weather:%s", city)
}

func (p *InstrumentedCacheProxy) GetMetrics() *metrics.CacheMetrics {
	return p.metrics
}

type InstrumentedChainCacheProxy struct {
	realChain WeatherProviderChain
	cache     CacheInterface
	cacheTTL  time.Duration
	metrics   *metrics.CacheMetrics
}

func NewInstrumentedChainCacheProxy(realChain WeatherProviderChain, cache CacheInterface, cacheTTL time.Duration, cacheType string) WeatherProviderChain {
	return &InstrumentedChainCacheProxy{
		realChain: realChain,
		cache:     cache,
		cacheTTL:  cacheTTL,
		metrics:   metrics.NewCacheMetrics(cacheType),
	}
}

func (p *InstrumentedChainCacheProxy) Handle(city string) (*models.WeatherResponse, error) {
	start := time.Now()
	cacheKey := p.generateCacheKey(city)

	cachedResponse, found := p.cache.Get(cacheKey)
	getLatency := time.Since(start).Seconds()
	p.metrics.RecordLatency("get", getLatency)

	if found {
		p.metrics.RecordHit()
		slog.Debug("chain cache hit", "city", city, "latency_ms", getLatency*1000)
		return cachedResponse, nil
	}

	p.metrics.RecordMiss()
	slog.Debug("chain cache miss", "city", city, "latency_ms", getLatency*1000)

	chainStart := time.Now()
	response, err := p.realChain.Handle(city)
	chainLatency := time.Since(chainStart).Seconds()

	if err != nil {
		return nil, err
	}

	setStart := time.Now()
	p.cache.Set(cacheKey, response, p.cacheTTL)
	setLatency := time.Since(setStart).Seconds()
	p.metrics.RecordLatency("set", setLatency)

	slog.Debug("chain cache set", "city", city, "chain_latency_ms", chainLatency*1000, "set_latency_ms", setLatency*1000)
	return response, nil
}

func (p *InstrumentedChainCacheProxy) SetNext(handler WeatherProviderChain) {
	p.realChain.SetNext(handler)
}

func (p *InstrumentedChainCacheProxy) GetProviderName() string {
	return fmt.Sprintf("InstrumentedCache(%s)", p.realChain.GetProviderName())
}

func (p *InstrumentedChainCacheProxy) generateCacheKey(city string) string {
	return fmt.Sprintf("weather:%s", city)
}

func (p *InstrumentedChainCacheProxy) GetMetrics() *metrics.CacheMetrics {
	return p.metrics
}
