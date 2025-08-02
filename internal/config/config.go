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

	// Default service ports
	defaultWeatherServicePort      = 8081
	defaultUserServicePort         = 8082
	defaultSubscriptionServicePort = 8083
	defaultNotificationServicePort = 8084
)

// Configuration structures matching the original config package

// Config represents the application configuration structure
type Config struct {
	Server         ServerConfig               `split_words:"true"`
	Services       ServicesConfig             `split_words:"true"`
	Database       DatabaseConfig             `split_words:"true"`
	UserDB         UserDatabaseConfig         `split_words:"true"`
	SubscriptionDB SubscriptionDatabaseConfig `split_words:"true"`
	Weather        WeatherConfig              `split_words:"true"`
	Email          EmailConfig                `split_words:"true"`
	Scheduler      SchedulerConfig            `split_words:"true"`
	Cache          CacheConfig                `split_words:"true"`
	MessageBroker  MessageBrokerConfig        `split_words:"true"`
	AppBaseURL     string                     `envconfig:"APP_URL" default:"http://localhost:8080"`
}

type ServerConfig struct {
	Port int `envconfig:"SERVER_PORT" default:"8080"`
}

type ServicesConfig struct {
	Weather      ServiceConfig `split_words:"true"`
	User         ServiceConfig `split_words:"true"`
	Subscription ServiceConfig `split_words:"true"`
	Notification ServiceConfig `split_words:"true"`
}

type ServiceConfig struct {
	Port int    `envconfig:"SERVICE_PORT"`
	Host string `envconfig:"SERVICE_HOST" default:"localhost"`
}

type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST" default:"localhost"`
	Port     int    `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" default:"postgres"`
	Password string `envconfig:"DB_PASSWORD" default:"postgres"`
	Name     string `envconfig:"DB_NAME" default:"weatherapi"`
	SSLMode  string `envconfig:"DB_SSL_MODE" default:"disable"`
}

func (c DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type UserDatabaseConfig struct {
	Host     string `envconfig:"USER_DB_HOST" default:"localhost"`
	Port     int    `envconfig:"USER_DB_PORT" default:"5433"`
	User     string `envconfig:"USER_DB_USER" default:"postgres"`
	Password string `envconfig:"USER_DB_PASSWORD" default:"postgres"`
	Name     string `envconfig:"USER_DB_NAME" default:"userapi"`
	SSLMode  string `envconfig:"USER_DB_SSL_MODE" default:"disable"`
}

func (c UserDatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type SubscriptionDatabaseConfig struct {
	Host     string `envconfig:"SUBSCRIPTION_DB_HOST" default:"localhost"`
	Port     int    `envconfig:"SUBSCRIPTION_DB_PORT" default:"5434"`
	User     string `envconfig:"SUBSCRIPTION_DB_USER" default:"postgres"`
	Password string `envconfig:"SUBSCRIPTION_DB_PASSWORD" default:"postgres"`
	Name     string `envconfig:"SUBSCRIPTION_DB_NAME" default:"subscriptionapi"`
	SSLMode  string `envconfig:"SUBSCRIPTION_DB_SSL_MODE" default:"disable"`
}

func (c SubscriptionDatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

type WeatherConfig struct {
	APIKey                string   `envconfig:"WEATHER_API_KEY"`
	BaseURL               string   `envconfig:"WEATHER_API_BASE_URL" default:"https://api.weatherapi.com/v1"`
	OpenWeatherMapKey     string   `envconfig:"OPENWEATHERMAP_API_KEY"`
	OpenWeatherMapBaseURL string   `envconfig:"OPENWEATHERMAP_API_BASE_URL" default:"https://api.openweathermap.org/data/2.5"`
	AccuWeatherKey        string   `envconfig:"ACCUWEATHER_API_KEY"`
	AccuWeatherBaseURL    string   `envconfig:"ACCUWEATHER_API_BASE_URL" default:"http://dataservice.accuweather.com/currentconditions/v1"`
	ProviderOrder         []string `envconfig:"WEATHER_PROVIDER_ORDER" default:"weatherapi,openweathermap,accuweather"`
	EnableCache           bool     `envconfig:"WEATHER_ENABLE_CACHE" default:"true"`
	EnableLogging         bool     `envconfig:"WEATHER_ENABLE_LOGGING" default:"true"`
	CacheTTLMinutes       int      `envconfig:"WEATHER_CACHE_TTL_MINUTES" default:"10"`
	LogFilePath           string   `envconfig:"WEATHER_LOG_FILE_PATH" default:"logs/weather_providers.log"`
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
	Type  CacheType   `envconfig:"CACHE_TYPE" default:"memory"`
	Redis RedisConfig `split_words:"true"`
}

type RedisConfig struct {
	Addr         string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	Password     string `envconfig:"REDIS_PASSWORD" default:""`
	DB           int    `envconfig:"REDIS_DB" default:"0"`
	DialTimeout  int    `envconfig:"REDIS_DIAL_TIMEOUT" default:"5"`
	ReadTimeout  int    `envconfig:"REDIS_READ_TIMEOUT" default:"3"`
	WriteTimeout int    `envconfig:"REDIS_WRITE_TIMEOUT" default:"3"`
}

type EmailConfig struct {
	SMTPHost     string `envconfig:"EMAIL_SMTP_HOST" default:"smtp.gmail.com"`
	SMTPPort     int    `envconfig:"EMAIL_SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"EMAIL_SMTP_USERNAME"`
	SMTPPassword string `envconfig:"EMAIL_SMTP_PASSWORD"`
	FromName     string `envconfig:"EMAIL_FROM_NAME" default:"Weather API"`
	FromAddress  string `envconfig:"EMAIL_FROM_ADDRESS" default:"no-reply@weatherapi.app"`
}

type SchedulerConfig struct {
	HourlyInterval int `envconfig:"HOURLY_INTERVAL" default:"60"`
	DailyInterval  int `envconfig:"DAILY_INTERVAL" default:"1440"`
}

type MessageBrokerConfig struct {
	URL      string `envconfig:"MESSAGE_BROKER_URL" default:"nats://localhost:4222"`
	Username string `envconfig:"MESSAGE_BROKER_USERNAME"`
	Password string `envconfig:"MESSAGE_BROKER_PASSWORD"`
}

func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing config: %w", err)
	}

	// Set default service ports if not provided
	if config.Services.Weather.Port == 0 {
		config.Services.Weather.Port = defaultWeatherServicePort
	}
	if config.Services.User.Port == 0 {
		config.Services.User.Port = defaultUserServicePort
	}
	if config.Services.Subscription.Port == 0 {
		config.Services.Subscription.Port = defaultSubscriptionServicePort
	}
	if config.Services.Notification.Port == 0 {
		config.Services.Notification.Port = defaultNotificationServicePort
	}

	if err := config.Validate(); err != nil {
		return nil, err
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

func (s *ServiceConfig) Validate() error {
	if s.Port < 1 || s.Port > maxPortNumber {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if s.Host == "" {
		return fmt.Errorf("host cannot be empty")
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
