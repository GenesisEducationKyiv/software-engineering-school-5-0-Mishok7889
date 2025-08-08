package ports

// ApplicationPorts aggregates all ports grouped by bounded context for dependency injection
type ApplicationPorts struct {
	Weather        WeatherPorts
	Subscription   SubscriptionPorts
	Notification   NotificationPorts
	Infrastructure InfrastructurePorts
}

// WeatherPorts groups weather-related ports
type WeatherPorts struct {
	Provider WeatherProviderManager
	Cache    WeatherCache
	Metrics  WeatherMetrics
	Service  WeatherService
}

// SubscriptionPorts groups subscription-related ports
type SubscriptionPorts struct {
	Repository SubscriptionRepository
	Service    SubscriptionService
}

// NotificationPorts groups notification-related ports
type NotificationPorts struct {
	EmailProvider EmailProvider
	Service       NotificationService
}

// InfrastructurePorts groups infrastructure-related ports
type InfrastructurePorts struct {
	ConfigProvider ConfigProvider
	Logger         Logger
	Database       interface{}
	TokenRepo      TokenRepository
	TokenGenerator TokenGenerator
	CacheMetrics   CacheMetrics
}
