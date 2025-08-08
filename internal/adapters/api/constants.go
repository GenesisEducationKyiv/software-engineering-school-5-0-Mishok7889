package api

// Error message constants
const (
	InternalServerErrorMsg  = "Internal server error"
	ExternalServiceErrorMsg = "External service unavailable"
	InvalidRequestFormatMsg = "Invalid request format"
)

// HTTP response messages
const (
	SubscriptionSuccessMsg = "Subscription successful. Confirmation email sent."
	ConfirmationSuccessMsg = "Subscription confirmed successfully"
	UnsubscribeSuccessMsg  = "Unsubscribed successfully"
	HealthStatusOK         = "ok"
)

// Metrics names
const (
	APIErrorsTotalMetric = "api_errors_total"
)

// Metrics label keys
const (
	EndpointLabelKey = "endpoint"
	ErrorLabelKey    = "error"
)

// Metrics label values
const (
	WeatherEndpoint = "weather"
	UsecaseError    = "usecase"
)

// Health check component names
const (
	DatabaseComponent   = "database"
	WeatherAPIComponent = "weatherAPI"
	SMTPComponent       = "smtp"
	ConfigComponent     = "config"
	HealthyStatus       = "healthy"
	ConnectedField      = "connected"
	StatusField         = "status"
)

// Log message constants
const (
	MetricsEndpointCalledMsg  = "Metrics endpoint called"
	ErrorGettingMetricsMsg    = "Error getting metrics"
	DebugEndpointCalledMsg    = "Debug endpoint called"
	GettingWeatherForCityMsg  = "Getting weather for city"
	WeatherUseCaseErrorMsg    = "Weather use case error"
	WeatherResultMsg          = "Weather result"
	HandlingSubscriptionMsg   = "Handling subscription request"
	RequestBindingErrorMsg    = "Request binding error"
	SubscriptionRequestMsg    = "Subscription request received"
	SubscriptionErrorMsg      = "Subscription error"
	SubscriptionCreatedMsg    = "Subscription created successfully"
	ConfirmingSubscriptionMsg = "Confirming subscription"
	ConfirmationErrorMsg      = "Confirmation error"
	SubscriptionConfirmedMsg  = "Subscription confirmed successfully"
	UnsubscribingMsg          = "Unsubscribing"
	UnsubscribeErrorMsg       = "Unsubscribe error"
	UnsubscribedMsg           = "Unsubscribed successfully"
	StartingHTTPServerMsg     = "Starting HTTP server"
)

// Log field keys
const (
	ErrorField       = "error"
	CityField        = "city"
	EmailField       = "email"
	FrequencyField   = "frequency"
	TokenField       = "token"
	TemperatureField = "temperature"
	PortField        = "port"
)

// Validation error messages
const (
	WeatherUseCaseRequiredMsg      = "weather use case is required"
	SubscriptionUseCaseRequiredMsg = "subscription use case is required"
	MetricsCollectorRequiredMsg    = "metrics collector is required"
	SystemHealthCheckerRequiredMsg = "system health checker is required"
	LoggerRequiredMsg              = "logger is required"
)
