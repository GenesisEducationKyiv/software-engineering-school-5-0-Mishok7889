package providers

import (
	"fmt"
	"log/slog"

	"weatherapi.app/models"
)

type BaseWeatherHandler struct {
	next         WeatherProviderChain
	provider     WeatherProvider
	providerName string
}

func NewBaseWeatherHandler(provider WeatherProvider, providerName string) *BaseWeatherHandler {
	return &BaseWeatherHandler{
		provider:     provider,
		providerName: providerName,
	}
}

func (h *BaseWeatherHandler) Handle(city string) (*models.WeatherResponse, error) {
	if h.provider != nil {
		response, err := h.provider.GetCurrentWeather(city)
		if err == nil {
			return response, nil
		}

		slog.Info("provider failed", "provider", h.providerName, "city", city, "error", err)

		// If this is the last handler in the chain and no next handler, return the actual error
		if h.next == nil {
			return nil, err
		}
	}

	if h.next != nil {
		return h.next.Handle(city)
	}

	return nil, fmt.Errorf("all weather providers failed for city: %s", city)
}

func (h *BaseWeatherHandler) SetNext(handler WeatherProviderChain) {
	h.next = handler
}

func (h *BaseWeatherHandler) GetProviderName() string {
	return h.providerName
}

type WeatherAPIHandler struct {
	*BaseWeatherHandler
}

func NewWeatherAPIHandler(provider WeatherProvider) WeatherProviderChain {
	baseHandler := NewBaseWeatherHandler(provider, "WeatherAPI")
	return &WeatherAPIHandler{
		BaseWeatherHandler: baseHandler,
	}
}

type OpenWeatherMapHandler struct {
	*BaseWeatherHandler
}

func NewOpenWeatherMapHandler(provider WeatherProvider) WeatherProviderChain {
	baseHandler := NewBaseWeatherHandler(provider, "OpenWeatherMap")
	return &OpenWeatherMapHandler{
		BaseWeatherHandler: baseHandler,
	}
}

type AccuWeatherHandler struct {
	*BaseWeatherHandler
}

func NewAccuWeatherHandler(provider WeatherProvider) WeatherProviderChain {
	baseHandler := NewBaseWeatherHandler(provider, "AccuWeather")
	return &AccuWeatherHandler{
		BaseWeatherHandler: baseHandler,
	}
}

type ChainBuilder struct {
	handlers []WeatherProviderChain
}

func NewChainBuilder() *ChainBuilder {
	return &ChainBuilder{
		handlers: make([]WeatherProviderChain, 0),
	}
}

func (cb *ChainBuilder) AddHandler(handler WeatherProviderChain) *ChainBuilder {
	cb.handlers = append(cb.handlers, handler)
	return cb
}

func (cb *ChainBuilder) Build() WeatherProviderChain {
	if len(cb.handlers) == 0 {
		return nil
	}

	for i := 0; i < len(cb.handlers)-1; i++ {
		cb.handlers[i].SetNext(cb.handlers[i+1])
	}

	return cb.handlers[0]
}
