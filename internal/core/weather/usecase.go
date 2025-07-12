package weather

import (
	"context"
	"fmt"

	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

type UseCase struct {
	weatherProvider ports.WeatherProviderManager
	cache           ports.WeatherCache
	config          ports.ConfigProvider
	logger          ports.Logger
	metrics         ports.WeatherMetrics
}

type UseCaseDependencies struct {
	WeatherProvider ports.WeatherProviderManager
	Cache           ports.WeatherCache
	Config          ports.ConfigProvider
	Logger          ports.Logger
	Metrics         ports.WeatherMetrics
}

func NewUseCase(deps UseCaseDependencies) (*UseCase, error) {
	if deps.WeatherProvider == nil {
		return nil, errors.NewValidationError("weather provider is required")
	}
	if deps.Cache == nil {
		return nil, errors.NewValidationError("cache is required")
	}
	if deps.Config == nil {
		return nil, errors.NewValidationError("config is required")
	}
	if deps.Logger == nil {
		return nil, errors.NewValidationError("logger is required")
	}
	if deps.Metrics == nil {
		return nil, errors.NewValidationError("metrics is required")
	}

	return &UseCase{
		weatherProvider: deps.WeatherProvider,
		cache:           deps.Cache,
		config:          deps.Config,
		logger:          deps.Logger,
		metrics:         deps.Metrics,
	}, nil
}

func (uc *UseCase) GetWeather(ctx context.Context, request WeatherRequest) (*Weather, error) {
	if err := request.IsValid(); err != nil {
		return nil, errors.NewValidationError("invalid weather request: " + err.Error())
	}

	request.NormalizeCity()
	normalizedCity := request.City
	uc.logger.Debug("Getting weather for city", ports.F("city", normalizedCity))

	weather, err := uc.getWeatherWithCache(ctx, normalizedCity)
	if err != nil {
		uc.logger.Error("Failed to get weather",
			ports.F("city", normalizedCity),
			ports.F("error", err))
		return nil, fmt.Errorf("get weather for city %s: %w", normalizedCity, err)
	}

	uc.logger.Debug("Weather retrieved successfully",
		ports.F("city", normalizedCity),
		ports.F("temperature", weather.Temperature))
	return weather, nil
}

func (uc *UseCase) getWeatherWithCache(ctx context.Context, city string) (*Weather, error) {
	if !uc.config.GetWeatherConfig().EnableCache {
		return uc.getWeatherFromProvider(ctx, city)
	}

	cacheKey := fmt.Sprintf("weather:%s", city)
	cachedWeather, err := uc.cache.Get(ctx, cacheKey)
	if err == nil && cachedWeather != nil {
		uc.logger.Debug("Weather found in cache", ports.F("city", city))
		return uc.convertFromPortsWeather(cachedWeather), nil
	}

	weather, err := uc.getWeatherFromProvider(ctx, city)
	if err != nil {
		return nil, err
	}

	cacheTTL := uc.config.GetWeatherConfig().CacheTTL
	portsWeather := uc.convertToPortsWeather(weather)
	if cacheErr := uc.cache.Set(ctx, cacheKey, portsWeather, cacheTTL); cacheErr != nil {
		uc.logger.Warn("Failed to cache weather data",
			ports.F("city", city),
			ports.F("error", cacheErr))
	}

	return weather, nil
}

func (uc *UseCase) getWeatherFromProvider(ctx context.Context, city string) (*Weather, error) {
	providerWeather, err := uc.weatherProvider.GetWeather(ctx, city)
	if err != nil {
		// Preserve NotFoundError from providers
		if errors.IsNotFoundError(err) {
			return nil, err
		}
		return nil, errors.NewExternalAPIError("weather provider failed", err)
	}

	domainWeather := uc.convertFromPortsWeather(providerWeather)
	if err := domainWeather.IsValid(); err != nil {
		return nil, errors.NewValidationError("invalid weather data from provider: " + err.Error())
	}

	return domainWeather, nil
}

func (uc *UseCase) convertToPortsWeather(weather *Weather) *ports.WeatherData {
	return &ports.WeatherData{
		Temperature: weather.Temperature,
		Humidity:    weather.Humidity,
		Description: weather.Description,
		City:        weather.City,
		Timestamp:   weather.Timestamp,
	}
}

func (uc *UseCase) convertFromPortsWeather(weatherData *ports.WeatherData) *Weather {
	return &Weather{
		Temperature: weatherData.Temperature,
		Humidity:    weatherData.Humidity,
		Description: weatherData.Description,
		City:        weatherData.City,
		Timestamp:   weatherData.Timestamp,
	}
}

func (uc *UseCase) GetProviderInfo(ctx context.Context) map[string]interface{} {
	return uc.metrics.GetProviderInfo()
}

func (uc *UseCase) GetCacheMetrics(ctx context.Context) (ports.CacheStats, error) {
	metrics, err := uc.metrics.GetCacheMetrics()
	if err != nil {
		return ports.CacheStats{}, fmt.Errorf("get cache metrics: %w", err)
	}
	return metrics, nil
}
