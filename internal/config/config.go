package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

const (
	maxRedisDB         = 15
	maxCacheTTLMinutes = 1440
	maxDailyInterval   = 10080
	maxPortNumber      = 65535
)

// Config represents the application configuration structure
type Config struct {
	Server         ServerConfig `split_words:"true"`
	Services       ServicesConfig
	Database       DatabaseConfig             `split_words:"true"`
	UserDB         UserDatabaseConfig         `split_words:"true"`
	SubscriptionDB SubscriptionDatabaseConfig `split_words:"true"`
	Weather        WeatherConfig              `split_words:"true"`
	Email          EmailConfig                `split_words:"true"`
	Scheduler      SchedulerConfig            `split_words:"true"`
	Cache          CacheConfig                `split_words:"true"`
	MessageBroker  MessageBrokerConfig        `split_words:"true"`
	AppBaseURL     string                     `envconfig:"APP_URL"`
}

type ServerConfig struct {
	Port int `envconfig:"SERVER_PORT"`
}

type ServicesConfig struct {
	Gateway      GatewayServiceConfig
	Weather      WeatherServiceConfig
	User         UserServiceConfig
	Subscription SubscriptionServiceConfig
	Notification NotificationServiceConfig
}

type GatewayServiceConfig struct {
	Port int    `envconfig:"GATEWAY_SERVICE_PORT"`
	Host string `envconfig:"GATEWAY_SERVICE_HOST"`
}

type WeatherServiceConfig struct {
	Port int    `envconfig:"WEATHER_SERVICE_PORT"`
	Host string `envconfig:"WEATHER_SERVICE_HOST"`
}

type UserServiceConfig struct {
	Port int    `envconfig:"USER_SERVICE_PORT"`
	Host string `envconfig:"USER_SERVICE_HOST"`
}

type SubscriptionServiceConfig struct {
	Port int    `envconfig:"SUBSCRIPTION_SERVICE_PORT"`
	Host string `envconfig:"SUBSCRIPTION_SERVICE_HOST"`
}

type NotificationServiceConfig struct {
	Port int    `envconfig:"NOTIFICATION_SERVICE_PORT"`
	Host string `envconfig:"NOTIFICATION_SERVICE_HOST"`
}

// ServiceConfig interface for compatibility with existing code
type ServiceConfig interface {
	GetPort() int
	GetHost() string
	Validate() error
}

func (g GatewayServiceConfig) GetPort() int    { return g.Port }
func (g GatewayServiceConfig) GetHost() string { return g.Host }
func (g GatewayServiceConfig) Validate() error {
	if g.Port < 1 || g.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if g.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

func (w WeatherServiceConfig) GetPort() int    { return w.Port }
func (w WeatherServiceConfig) GetHost() string { return w.Host }
func (w WeatherServiceConfig) Validate() error {
	if w.Port < 1 || w.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if w.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

func (u UserServiceConfig) GetPort() int    { return u.Port }
func (u UserServiceConfig) GetHost() string { return u.Host }
func (u UserServiceConfig) Validate() error {
	if u.Port < 1 || u.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if u.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

func (s SubscriptionServiceConfig) GetPort() int    { return s.Port }
func (s SubscriptionServiceConfig) GetHost() string { return s.Host }
func (s SubscriptionServiceConfig) Validate() error {
	if s.Port < 1 || s.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

func (n NotificationServiceConfig) GetPort() int    { return n.Port }
func (n NotificationServiceConfig) GetHost() string { return n.Host }
func (n NotificationServiceConfig) Validate() error {
	if n.Port < 1 || n.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if n.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	return nil
}

type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST"`
	Port     int    `envconfig:"DB_PORT"`
	User     string `envconfig:"DB_USER"`
	Password string `envconfig:"DB_PASSWORD"`
	Name     string `envconfig:"DB_NAME"`
	SSLMode  string `envconfig:"DB_SSL_MODE"`
}

func (c DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type UserDatabaseConfig struct {
	Host     string `envconfig:"USER_DB_HOST"`
	Port     int    `envconfig:"USER_DB_PORT"`
	User     string `envconfig:"USER_DB_USER"`
	Password string `envconfig:"USER_DB_PASSWORD"`
	Name     string `envconfig:"USER_DB_NAME"`
	SSLMode  string `envconfig:"USER_DB_SSL_MODE"`
}

func (c UserDatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type SubscriptionDatabaseConfig struct {
	Host     string `envconfig:"SUBSCRIPTION_DB_HOST"`
	Port     int    `envconfig:"SUBSCRIPTION_DB_PORT"`
	User     string `envconfig:"SUBSCRIPTION_DB_USER"`
	Password string `envconfig:"SUBSCRIPTION_DB_PASSWORD"`
	Name     string `envconfig:"SUBSCRIPTION_DB_NAME"`
	SSLMode  string `envconfig:"SUBSCRIPTION_DB_SSL_MODE"`
}

func (c SubscriptionDatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type WeatherConfig struct {
	APIKey                string   `envconfig:"WEATHER_API_KEY"`
	BaseURL               string   `envconfig:"WEATHER_API_BASE_URL"`
	OpenWeatherMapKey     string   `envconfig:"OPENWEATHERMAP_API_KEY"`
	OpenWeatherMapBaseURL string   `envconfig:"OPENWEATHERMAP_API_BASE_URL"`
	AccuWeatherKey        string   `envconfig:"ACCUWEATHER_API_KEY"`
	AccuWeatherBaseURL    string   `envconfig:"ACCUWEATHER_API_BASE_URL"`
	ProviderOrder         []string `envconfig:"WEATHER_PROVIDER_ORDER"`
	EnableCache           bool     `envconfig:"WEATHER_ENABLE_CACHE"`
	EnableLogging         bool     `envconfig:"WEATHER_ENABLE_LOGGING"`
	CacheTTLMinutes       int      `envconfig:"WEATHER_CACHE_TTL_MINUTES"`
	LogFilePath           string   `envconfig:"WEATHER_LOG_FILE_PATH"`
}

// CacheType represents the type of cache to use
type CacheType int

const (
	CacheTypeUnknown CacheType = iota
	CacheTypeMemory
	CacheTypeRedis
)

// String returns the string representation of cache type
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

// IsValid checks if the cache type is valid
func (c CacheType) IsValid() bool {
	return c == CacheTypeMemory || c == CacheTypeRedis
}

// CacheTypeFromString converts string to CacheType enum
func CacheTypeFromString(s string) CacheType {
	switch s {
	case "memory":
		return CacheTypeMemory
	case "redis":
		return CacheTypeRedis
	default:
		return CacheTypeUnknown
	}
}

// UnmarshalText implements encoding.TextUnmarshaler for envconfig
func (c *CacheType) UnmarshalText(text []byte) error {
	*c = CacheTypeFromString(string(text))
	return nil
}

// MarshalText implements encoding.TextMarshaler for envconfig
func (c CacheType) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

type CacheConfig struct {
	Type  CacheType   `envconfig:"CACHE_TYPE"`
	Redis RedisConfig `split_words:"true"`
}

type RedisConfig struct {
	Addr         string `envconfig:"REDIS_ADDR"`
	Password     string `envconfig:"REDIS_PASSWORD"`
	DB           int    `envconfig:"REDIS_DB"`
	DialTimeout  int    `envconfig:"REDIS_DIAL_TIMEOUT"`
	ReadTimeout  int    `envconfig:"REDIS_READ_TIMEOUT"`
	WriteTimeout int    `envconfig:"REDIS_WRITE_TIMEOUT"`
}

type EmailConfig struct {
	SMTPHost     string `envconfig:"EMAIL_SMTP_HOST"`
	SMTPPort     int    `envconfig:"EMAIL_SMTP_PORT"`
	SMTPUsername string `envconfig:"EMAIL_SMTP_USERNAME"`
	SMTPPassword string `envconfig:"EMAIL_SMTP_PASSWORD"`
	FromName     string `envconfig:"EMAIL_FROM_NAME"`
	FromAddress  string `envconfig:"EMAIL_FROM_ADDRESS"`
}

type SchedulerConfig struct {
	HourlyInterval int `envconfig:"HOURLY_INTERVAL"`
	DailyInterval  int `envconfig:"DAILY_INTERVAL"`
}

type MessageBrokerConfig struct {
	URL      string `envconfig:"MESSAGE_BROKER_URL"`
	Username string `envconfig:"MESSAGE_BROKER_USERNAME"`
	Password string `envconfig:"MESSAGE_BROKER_PASSWORD"`
}

func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadGatewayConfig loads configuration for API Gateway (no weather API keys required)
func LoadGatewayConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}

	// Debug: print what was actually parsed
	fmt.Printf("DEBUG: Gateway config parsed - Port: %d, Host: %s\n", config.Services.Gateway.Port, config.Services.Gateway.Host)
	fmt.Printf("DEBUG: Weather config - Port: %d, Host: %s\n", config.Services.Weather.Port, config.Services.Weather.Host)
	fmt.Printf("DEBUG: User config - Port: %d, Host: %s\n", config.Services.User.Port, config.Services.User.Host)
	fmt.Printf("DEBUG: Subscription config - Port: %d, Host: %s\n", config.Services.Subscription.Port, config.Services.Subscription.Host)

	// Only validate gateway-specific configuration
	if err := config.Services.Gateway.Validate(); err != nil {
		return nil, fmt.Errorf("gateway service config: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if err := c.Services.Validate(); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	if err := c.UserDB.Validate(); err != nil {
		return err
	}
	if err := c.SubscriptionDB.Validate(); err != nil {
		return err
	}
	if err := c.Weather.Validate(); err != nil {
		return err
	}
	if err := c.Email.Validate(); err != nil {
		return err
	}
	if err := c.Scheduler.Validate(); err != nil {
		return err
	}
	if err := c.Cache.Validate(); err != nil {
		return err
	}
	if err := c.MessageBroker.Validate(); err != nil {
		return err
	}
	if err := c.validateAppBaseURL(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateAppBaseURL() error {
	if c.AppBaseURL == "" {
		return fmt.Errorf("APP_URL cannot be empty")
	}
	if !strings.HasPrefix(c.AppBaseURL, "http://") && !strings.HasPrefix(c.AppBaseURL, "https://") {
		return fmt.Errorf("APP_URL must start with http:// or https://")
	}
	return nil
}

func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > maxPortNumber {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}
	return nil
}

func (s *ServicesConfig) Validate() error {
	if err := s.Gateway.Validate(); err != nil {
		return fmt.Errorf("gateway service config: %w", err)
	}
	if err := s.Weather.Validate(); err != nil {
		return fmt.Errorf("weather service config: %w", err)
	}
	if err := s.User.Validate(); err != nil {
		return fmt.Errorf("user service config: %w", err)
	}
	if err := s.Subscription.Validate(); err != nil {
		return fmt.Errorf("subscription service config: %w", err)
	}
	if err := s.Notification.Validate(); err != nil {
		return fmt.Errorf("notification service config: %w", err)
	}
	return nil
}

func (d *DatabaseConfig) Validate() error {
	if d.Host == "" {
		return fmt.Errorf("DB_HOST cannot be empty")
	}
	if d.Port < 1 || d.Port > maxPortNumber {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	if d.User == "" {
		return fmt.Errorf("DB_USER cannot be empty")
	}
	if d.Name == "" {
		return fmt.Errorf("DB_NAME cannot be empty")
	}
	if err := d.ValidateSSLMode(); err != nil {
		return err
	}
	return nil
}

func (d *DatabaseConfig) ValidateSSLMode() error {
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range validSSLModes {
		if d.SSLMode == mode {
			return nil
		}
	}
	return fmt.Errorf("DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", "))
}

func (d *UserDatabaseConfig) Validate() error {
	if d.Host == "" {
		return fmt.Errorf("USER_DB_HOST cannot be empty")
	}
	if d.Port < 1 || d.Port > maxPortNumber {
		return fmt.Errorf("USER_DB_PORT must be between 1 and 65535")
	}
	if d.User == "" {
		return fmt.Errorf("USER_DB_USER cannot be empty")
	}
	if d.Name == "" {
		return fmt.Errorf("USER_DB_NAME cannot be empty")
	}
	if err := d.ValidateSSLMode(); err != nil {
		return err
	}
	return nil
}

func (d *UserDatabaseConfig) ValidateSSLMode() error {
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range validSSLModes {
		if d.SSLMode == mode {
			return nil
		}
	}
	return fmt.Errorf("USER_DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", "))
}

func (d *SubscriptionDatabaseConfig) Validate() error {
	if d.Host == "" {
		return fmt.Errorf("SUBSCRIPTION_DB_HOST cannot be empty")
	}
	if d.Port < 1 || d.Port > maxPortNumber {
		return fmt.Errorf("SUBSCRIPTION_DB_PORT must be between 1 and 65535")
	}
	if d.User == "" {
		return fmt.Errorf("SUBSCRIPTION_DB_USER cannot be empty")
	}
	if d.Name == "" {
		return fmt.Errorf("SUBSCRIPTION_DB_NAME cannot be empty")
	}
	if err := d.ValidateSSLMode(); err != nil {
		return err
	}
	return nil
}

func (d *SubscriptionDatabaseConfig) ValidateSSLMode() error {
	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	for _, mode := range validSSLModes {
		if d.SSLMode == mode {
			return nil
		}
	}
	return fmt.Errorf("SUBSCRIPTION_DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", "))
}

func (w *WeatherConfig) Validate() error {
	if w.APIKey == "" && w.OpenWeatherMapKey == "" && w.AccuWeatherKey == "" {
		return fmt.Errorf("at least one weather provider API key must be configured")
	}

	if w.APIKey != "" {
		if w.BaseURL == "" {
			return fmt.Errorf("WEATHER_API_BASE_URL cannot be empty when WEATHER_API_KEY is set")
		}
		if !strings.HasPrefix(w.BaseURL, "http://") && !strings.HasPrefix(w.BaseURL, "https://") {
			return fmt.Errorf("WEATHER_API_BASE_URL must start with http:// or https://")
		}
	}

	if w.CacheTTLMinutes < 1 || w.CacheTTLMinutes > maxCacheTTLMinutes {
		return fmt.Errorf("WEATHER_CACHE_TTL_MINUTES must be between 1 and 1440 minutes")
	}

	validProviders := map[string]bool{
		"weatherapi":     true,
		"openweathermap": true,
		"accuweather":    true,
	}

	for _, provider := range w.ProviderOrder {
		if !validProviders[provider] {
			return fmt.Errorf("invalid weather provider in order: %s", provider)
		}
	}

	return nil
}

func (c *CacheConfig) Validate() error {
	if !c.Type.IsValid() {
		return fmt.Errorf("CACHE_TYPE must be one of: memory, redis")
	}

	if c.Type == CacheTypeRedis {
		return c.Redis.Validate()
	}

	return nil
}

func (r *RedisConfig) Validate() error {
	if r.Addr == "" {
		return fmt.Errorf("REDIS_ADDR cannot be empty when using Redis cache")
	}
	if r.DB < 0 || r.DB > maxRedisDB {
		return fmt.Errorf("REDIS_DB must be between 0 and 15")
	}
	if r.DialTimeout < 1 {
		return fmt.Errorf("REDIS_DIAL_TIMEOUT must be at least 1 second")
	}
	if r.ReadTimeout < 1 {
		return fmt.Errorf("REDIS_READ_TIMEOUT must be at least 1 second")
	}
	if r.WriteTimeout < 1 {
		return fmt.Errorf("REDIS_WRITE_TIMEOUT must be at least 1 second")
	}
	return nil
}

func (e *EmailConfig) Validate() error {
	if e.SMTPHost == "" {
		return fmt.Errorf("EMAIL_SMTP_HOST cannot be empty")
	}
	if e.SMTPPort < 1 || e.SMTPPort > maxPortNumber {
		return fmt.Errorf("EMAIL_SMTP_PORT must be between 1 and 65535")
	}
	if (e.SMTPUsername == "") != (e.SMTPPassword == "") {
		return fmt.Errorf("EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty")
	}
	if e.FromName == "" {
		return fmt.Errorf("EMAIL_FROM_NAME cannot be empty")
	}
	if e.FromAddress == "" {
		return fmt.Errorf("EMAIL_FROM_ADDRESS cannot be empty")
	}
	if !strings.Contains(e.FromAddress, "@") {
		return fmt.Errorf("EMAIL_FROM_ADDRESS must be a valid email address")
	}
	return nil
}

func (s *SchedulerConfig) Validate() error {
	if s.HourlyInterval < 1 {
		return fmt.Errorf("HOURLY_INTERVAL must be at least 1 minute")
	}
	if s.DailyInterval < 1 {
		return fmt.Errorf("DAILY_INTERVAL must be at least 1 minute")
	}
	if s.HourlyInterval > maxCacheTTLMinutes {
		return fmt.Errorf("HOURLY_INTERVAL cannot exceed 1440 minutes (24 hours)")
	}
	if s.DailyInterval > maxDailyInterval {
		return fmt.Errorf("DAILY_INTERVAL cannot exceed 10080 minutes (7 days)")
	}
	return nil
}

func (m *MessageBrokerConfig) Validate() error {
	if m.URL == "" {
		return fmt.Errorf("MESSAGE_BROKER_URL cannot be empty")
	}
	if !strings.HasPrefix(m.URL, "nats://") {
		return fmt.Errorf("MESSAGE_BROKER_URL must start with nats:// protocol")
	}
	if (m.Username == "") != (m.Password == "") {
		return fmt.Errorf("MESSAGE_BROKER_USERNAME and MESSAGE_BROKER_PASSWORD must both be provided or both be empty")
	}
	return nil
}
