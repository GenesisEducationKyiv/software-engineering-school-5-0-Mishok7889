package config

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"weatherapi.app/pkg/errors"
)

const (
	maxRedisDB         = 15
	maxCacheTTLMinutes = 1440
	maxDailyInterval   = 10080
	maxPortNumber      = 65535
)

// Configuration structures matching the original config package

// Config represents the application configuration structure
type Config struct {
	Server     ServerConfig    `split_words:"true"`
	Database   DatabaseConfig  `split_words:"true"`
	Weather    WeatherConfig   `split_words:"true"`
	Email      EmailConfig     `split_words:"true"`
	Scheduler  SchedulerConfig `split_words:"true"`
	Cache      CacheConfig     `split_words:"true"`
	AppBaseURL string          `envconfig:"APP_URL" default:"http://localhost:8080"`
}

type ServerConfig struct {
	Port int `envconfig:"SERVER_PORT" default:"8080"`
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

func LoadConfig() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, errors.NewConfigurationError("error processing config", err)
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
	if err := c.Database.Validate(); err != nil {
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
	if err := c.validateAppBaseURL(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateAppBaseURL() error {
	if c.AppBaseURL == "" {
		return errors.NewConfigurationError("APP_URL cannot be empty", nil)
	}
	if !strings.HasPrefix(c.AppBaseURL, "http://") && !strings.HasPrefix(c.AppBaseURL, "https://") {
		return errors.NewConfigurationError("APP_URL must start with http:// or https://", nil)
	}
	return nil
}

func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > maxPortNumber {
		return errors.NewConfigurationError("SERVER_PORT must be between 1 and 65535", nil)
	}
	return nil
}

func (d *DatabaseConfig) Validate() error {
	if d.Host == "" {
		return errors.NewConfigurationError("DB_HOST cannot be empty", nil)
	}
	if d.Port < 1 || d.Port > maxPortNumber {
		return errors.NewConfigurationError("DB_PORT must be between 1 and 65535", nil)
	}
	if d.User == "" {
		return errors.NewConfigurationError("DB_USER cannot be empty", nil)
	}
	if d.Name == "" {
		return errors.NewConfigurationError("DB_NAME cannot be empty", nil)
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
	return errors.NewConfigurationError(
		fmt.Sprintf("DB_SSL_MODE must be one of: %s", strings.Join(validSSLModes, ", ")), nil)
}

func (w *WeatherConfig) Validate() error {
	if w.APIKey == "" && w.OpenWeatherMapKey == "" && w.AccuWeatherKey == "" {
		return errors.NewConfigurationError("at least one weather provider API key must be configured", nil)
	}

	if w.APIKey != "" {
		if w.BaseURL == "" {
			return errors.NewConfigurationError("WEATHER_API_BASE_URL cannot be empty when WEATHER_API_KEY is set", nil)
		}
		if !strings.HasPrefix(w.BaseURL, "http://") && !strings.HasPrefix(w.BaseURL, "https://") {
			return errors.NewConfigurationError("WEATHER_API_BASE_URL must start with http:// or https://", nil)
		}
	}

	if w.CacheTTLMinutes < 1 || w.CacheTTLMinutes > maxCacheTTLMinutes {
		return errors.NewConfigurationError("WEATHER_CACHE_TTL_MINUTES must be between 1 and 1440 minutes", nil)
	}

	validProviders := map[string]bool{
		"weatherapi":     true,
		"openweathermap": true,
		"accuweather":    true,
	}

	for _, provider := range w.ProviderOrder {
		if !validProviders[provider] {
			return errors.NewConfigurationError(fmt.Sprintf("invalid weather provider in order: %s", provider), nil)
		}
	}

	return nil
}

func (c *CacheConfig) Validate() error {
	if !c.Type.IsValid() {
		return errors.NewConfigurationError("CACHE_TYPE must be one of: memory, redis", nil)
	}

	if c.Type == CacheTypeRedis {
		return c.Redis.Validate()
	}

	return nil
}

func (r *RedisConfig) Validate() error {
	if r.Addr == "" {
		return errors.NewConfigurationError("REDIS_ADDR cannot be empty when using Redis cache", nil)
	}
	if r.DB < 0 || r.DB > maxRedisDB {
		return errors.NewConfigurationError("REDIS_DB must be between 0 and 15", nil)
	}
	if r.DialTimeout < 1 {
		return errors.NewConfigurationError("REDIS_DIAL_TIMEOUT must be at least 1 second", nil)
	}
	if r.ReadTimeout < 1 {
		return errors.NewConfigurationError("REDIS_READ_TIMEOUT must be at least 1 second", nil)
	}
	if r.WriteTimeout < 1 {
		return errors.NewConfigurationError("REDIS_WRITE_TIMEOUT must be at least 1 second", nil)
	}
	return nil
}

func (e *EmailConfig) Validate() error {
	if e.SMTPHost == "" {
		return errors.NewConfigurationError("EMAIL_SMTP_HOST cannot be empty", nil)
	}
	if e.SMTPPort < 1 || e.SMTPPort > maxPortNumber {
		return errors.NewConfigurationError("EMAIL_SMTP_PORT must be between 1 and 65535", nil)
	}
	if (e.SMTPUsername == "") != (e.SMTPPassword == "") {
		return errors.NewConfigurationError("EMAIL_SMTP_USERNAME and EMAIL_SMTP_PASSWORD must both be provided or both be empty", nil)
	}
	if e.FromName == "" {
		return errors.NewConfigurationError("EMAIL_FROM_NAME cannot be empty", nil)
	}
	if e.FromAddress == "" {
		return errors.NewConfigurationError("EMAIL_FROM_ADDRESS cannot be empty", nil)
	}
	if !strings.Contains(e.FromAddress, "@") {
		return errors.NewConfigurationError("EMAIL_FROM_ADDRESS must be a valid email address", nil)
	}
	return nil
}

func (s *SchedulerConfig) Validate() error {
	if s.HourlyInterval < 1 {
		return errors.NewConfigurationError("HOURLY_INTERVAL must be at least 1 minute", nil)
	}
	if s.DailyInterval < 1 {
		return errors.NewConfigurationError("DAILY_INTERVAL must be at least 1 minute", nil)
	}
	if s.HourlyInterval > maxCacheTTLMinutes {
		return errors.NewConfigurationError("HOURLY_INTERVAL cannot exceed 1440 minutes (24 hours)", nil)
	}
	if s.DailyInterval > maxDailyInterval {
		return errors.NewConfigurationError("DAILY_INTERVAL cannot exceed 10080 minutes (7 days)", nil)
	}
	return nil
}
