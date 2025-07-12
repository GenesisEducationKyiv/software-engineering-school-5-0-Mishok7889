package ports

// ApplicationPorts aggregates all ports for dependency injection
type ApplicationPorts struct {
	// Weather
	WeatherProvider WeatherProviderManager
	WeatherCache    WeatherCache
	WeatherMetrics  WeatherMetrics

	// Subscription
	SubscriptionRepository SubscriptionRepository
	TokenRepository        TokenRepository

	// Communication
	EmailProvider EmailProvider

	// Cache
	CacheMetrics CacheMetrics

	// Infrastructure
	ConfigProvider ConfigProvider
	Logger         Logger
	Database       interface{}
}
