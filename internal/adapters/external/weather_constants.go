package external

import "time"

// Provider type for weather provider identification
type Provider string

// Weather provider constants
const (
	WeatherAPI     Provider = "weatherapi"
	OpenWeatherMap Provider = "openweathermap"
	AccuWeather    Provider = "accuweather"
)

// HTTP client timeout constants
const (
	DefaultHTTPTimeout = 10 * time.Second
)

// Default base URLs
const (
	DefaultWeatherAPIURL     = "http://api.weatherapi.com/v1"
	DefaultOpenWeatherMapURL = "https://api.openweathermap.org/data/2.5"
	DefaultAccuWeatherURL    = "http://dataservice.accuweather.com/currentconditions/v1"
)

// Validation error messages
const (
	EmptyCityValidationMsg   = "city cannot be empty"
	EmptyAPIKeyValidationMsg = "API key cannot be empty"
	MissingAPIKeyMsg         = "API key not configured"
	NoProvidersConfiguredMsg = "no weather providers configured"
	CityNotFoundMsg          = "city not found"
)

// HTTP response messages
const (
	WeatherAPICallFailedMsg       = "failed to call WeatherAPI"
	OpenWeatherMapCallFailedMsg   = "failed to call OpenWeatherMap"
	AccuWeatherCallFailedMsg      = "AccuWeather API key not configured"
	WeatherAPIDecodeFailedMsg     = "failed to decode WeatherAPI response"
	OpenWeatherMapDecodeFailedMsg = "failed to decode OpenWeatherMap response"
)

// Log messages
const (
	CreatedProviderMsg          = "Created weather provider"
	TryingProviderMsg           = "Trying weather provider"
	ProviderSucceededMsg        = "Weather provider succeeded"
	ProviderFailedMsg           = "Weather provider failed, trying next"
	AllProvidersFailedMsg       = "All weather providers failed"
	ProviderValidationFailedMsg = "Provider validation failed"
)

// Log field keys
const (
	ProviderField       = "provider"
	AttemptField        = "attempt"
	CityField           = "city"
	TemperatureField    = "temperature"
	ErrorField          = "error"
	ProvidersTriedField = "providers_tried"
	LastErrorField      = "last_error"
)

// Default values
const (
	DefaultWeatherDescription = "N/A"
	DefaultTemperature        = 22.5
	DefaultHumidity           = 65.0
	DefaultMockDescription    = "Partly cloudy"
)

// Provider info field names
const (
	TotalProvidersField  = "total_providers"
	ProviderOrderField   = "provider_order"
	ChainEnabledField    = "chain_enabled"
	FallbackEnabledField = "fallback_enabled"
)
