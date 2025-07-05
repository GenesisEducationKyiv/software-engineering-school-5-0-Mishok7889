package providers

import (
	"time"

	"weatherapi.app/models"
)

type WeatherLoggerDecorator struct {
	wrappedProvider WeatherProvider
	logger          FileLogger
	providerName    string
}

func NewWeatherLoggerDecorator(provider WeatherProvider, logger FileLogger, providerName string) WeatherProvider {
	return &WeatherLoggerDecorator{
		wrappedProvider: provider,
		logger:          logger,
		providerName:    providerName,
	}
}

func (d *WeatherLoggerDecorator) GetCurrentWeather(city string) (*models.WeatherResponse, error) {
	d.logger.LogRequest(d.providerName, city)
	startTime := time.Now()

	response, err := d.wrappedProvider.GetCurrentWeather(city)
	duration := time.Since(startTime)

	if err != nil {
		d.logger.LogError(d.providerName, city, err, duration)
		return nil, err
	}

	d.logger.LogResponse(d.providerName, city, response, duration)
	return response, nil
}

type WeatherChainLoggerDecorator struct {
	wrappedChain WeatherProviderChain
	logger       FileLogger
}

// NewWeatherChainLoggerDecorator creates a logging decorator for chains
func NewWeatherChainLoggerDecorator(chain WeatherProviderChain, logger FileLogger) WeatherProviderChain {
	return &WeatherChainLoggerDecorator{
		wrappedChain: chain,
		logger:       logger,
	}
}

func (d *WeatherChainLoggerDecorator) Handle(city string) (*models.WeatherResponse, error) {
	d.logger.LogRequest("WeatherChain", city)
	startTime := time.Now()

	response, err := d.wrappedChain.Handle(city)
	duration := time.Since(startTime)

	if err != nil {
		d.logger.LogError("WeatherChain", city, err, duration)
		return nil, err
	}

	d.logger.LogResponse("WeatherChain", city, response, duration)
	return response, nil
}

// SetNext delegates to the wrapped chain
func (d *WeatherChainLoggerDecorator) SetNext(handler WeatherProviderChain) {
	d.wrappedChain.SetNext(handler)
}

// GetProviderName returns the name of the wrapped chain with decoration info
func (d *WeatherChainLoggerDecorator) GetProviderName() string {
	return "Logged(" + d.wrappedChain.GetProviderName() + ")"
}

type MultiProviderLoggerDecorator struct {
	wrappedChain WeatherProviderChain
	logger       FileLogger
}

// NewMultiProviderLoggerDecorator creates a decorator that logs each provider attempt
func NewMultiProviderLoggerDecorator(chain WeatherProviderChain, logger FileLogger) WeatherProviderChain {
	return &MultiProviderLoggerDecorator{
		wrappedChain: chain,
		logger:       logger,
	}
}

// Handle logs each provider attempt individually
func (d *MultiProviderLoggerDecorator) Handle(city string) (*models.WeatherResponse, error) {
	return d.handleWithLogging(d.wrappedChain, city)
}

func (d *MultiProviderLoggerDecorator) handleWithLogging(handler WeatherProviderChain, city string) (*models.WeatherResponse, error) {
	if handler == nil {
		return nil, nil
	}

	providerName := handler.GetProviderName()
	d.logger.LogRequest(providerName, city)
	startTime := time.Now()

	response, err := d.tryCurrentHandler(handler, city)
	duration := time.Since(startTime)

	if err != nil {
		d.logger.LogError(providerName, city, err, duration)
		return handler.Handle(city)
	}

	d.logger.LogResponse(providerName, city, response, duration)
	return response, nil
}

func (d *MultiProviderLoggerDecorator) tryCurrentHandler(handler WeatherProviderChain, city string) (*models.WeatherResponse, error) {
	return handler.Handle(city)
}

// SetNext delegates to the wrapped chain
func (d *MultiProviderLoggerDecorator) SetNext(handler WeatherProviderChain) {
	d.wrappedChain.SetNext(handler)
}

// GetProviderName returns the name with multi-logging info
func (d *MultiProviderLoggerDecorator) GetProviderName() string {
	return "MultiLogged(" + d.wrappedChain.GetProviderName() + ")"
}

type CompositeWeatherDecorator struct {
	decorators   []func(WeatherProvider) WeatherProvider
	baseProvider WeatherProvider
}

// NewCompositeWeatherDecorator creates a composite decorator
func NewCompositeWeatherDecorator(baseProvider WeatherProvider) *CompositeWeatherDecorator {
	return &CompositeWeatherDecorator{
		baseProvider: baseProvider,
		decorators:   make([]func(WeatherProvider) WeatherProvider, 0),
	}
}

func (c *CompositeWeatherDecorator) AddDecorator(decorator func(WeatherProvider) WeatherProvider) *CompositeWeatherDecorator {
	c.decorators = append(c.decorators, decorator)
	return c
}

func (c *CompositeWeatherDecorator) Build() WeatherProvider {
	result := c.baseProvider

	for _, decorator := range c.decorators {
		result = decorator(result)
	}

	return result
}
