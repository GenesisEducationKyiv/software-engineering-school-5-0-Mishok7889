package ports

import (
	"context"
	"time"
)

// WeatherConfig represents weather service configuration
type WeatherConfig struct {
	EnableCache bool
	CacheTTL    time.Duration
}

// AppConfig represents application configuration
type AppConfig struct {
	BaseURL string
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// EmailConfig represents email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromName     string
	FromAddress  string
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Type  string
	Redis RedisConfig
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  int
	ReadTimeout  int
	WriteTimeout int
}

// SchedulerConfig represents scheduler configuration
type SchedulerConfig struct {
	HourlyInterval int
	DailyInterval  int
}

// ConfigProvider defines the contract for configuration management
type ConfigProvider interface {
	GetWeatherConfig() WeatherConfig
	GetAppConfig() AppConfig
	GetServerConfig() ServerConfig
	GetDatabaseConfig() DatabaseConfig
	GetEmailConfig() EmailConfig
	GetCacheConfig() CacheConfig
	GetSchedulerConfig() SchedulerConfig
}

// Logger defines the contract for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// F creates a log field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// MetricsCollector defines the contract for metrics collection
type MetricsCollector interface {
	RecordCacheHit(ctx context.Context)
	RecordCacheMiss(ctx context.Context)
	RecordWeatherAPICall(ctx context.Context, provider string, success bool)
}
