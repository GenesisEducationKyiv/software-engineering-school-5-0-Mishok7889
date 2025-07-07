package infrastructure

import (
	"time"

	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// ConfigProviderAdapter implements the ConfigProvider port
type ConfigProviderAdapter struct {
	config *config.Config
}

// NewConfigProviderAdapter creates a new config provider adapter
func NewConfigProviderAdapter(cfg *config.Config) *ConfigProviderAdapter {
	return &ConfigProviderAdapter{
		config: cfg,
	}
}

// GetAppConfig returns application configuration
func (c *ConfigProviderAdapter) GetAppConfig() ports.AppConfig {
	return ports.AppConfig{
		BaseURL: c.config.AppBaseURL,
	}
}

// GetDatabaseConfig returns database configuration
func (c *ConfigProviderAdapter) GetDatabaseConfig() ports.DatabaseConfig {
	return ports.DatabaseConfig{
		Host:     c.config.Database.Host,
		Port:     c.config.Database.Port,
		User:     c.config.Database.User,
		Password: c.config.Database.Password,
		Name:     c.config.Database.Name,
		SSLMode:  c.config.Database.SSLMode,
	}
}

// GetServerConfig returns server configuration
func (c *ConfigProviderAdapter) GetServerConfig() ports.ServerConfig {
	return ports.ServerConfig{
		Port: c.config.Server.Port,
	}
}

// GetWeatherConfig returns weather configuration
func (c *ConfigProviderAdapter) GetWeatherConfig() ports.WeatherConfig {
	return ports.WeatherConfig{
		EnableCache: c.config.Weather.EnableCache,
		CacheTTL:    time.Duration(c.config.Weather.CacheTTLMinutes) * time.Minute,
	}
}

// GetEmailConfig returns email configuration
func (c *ConfigProviderAdapter) GetEmailConfig() ports.EmailConfig {
	return ports.EmailConfig{
		SMTPHost:     c.config.Email.SMTPHost,
		SMTPPort:     c.config.Email.SMTPPort,
		SMTPUsername: c.config.Email.SMTPUsername,
		SMTPPassword: c.config.Email.SMTPPassword,
		FromName:     c.config.Email.FromName,
		FromAddress:  c.config.Email.FromAddress,
	}
}

// GetCacheConfig returns cache configuration
func (c *ConfigProviderAdapter) GetCacheConfig() ports.CacheConfig {
	return ports.CacheConfig{
		Type: c.config.Cache.Type.String(),
		Redis: ports.RedisConfig{
			Addr:         c.config.Cache.Redis.Addr,
			Password:     c.config.Cache.Redis.Password,
			DB:           c.config.Cache.Redis.DB,
			DialTimeout:  c.config.Cache.Redis.DialTimeout,
			ReadTimeout:  c.config.Cache.Redis.ReadTimeout,
			WriteTimeout: c.config.Cache.Redis.WriteTimeout,
		},
	}
}

// GetSchedulerConfig returns scheduler configuration
func (c *ConfigProviderAdapter) GetSchedulerConfig() ports.SchedulerConfig {
	return ports.SchedulerConfig{
		HourlyInterval: c.config.Scheduler.HourlyInterval,
		DailyInterval:  c.config.Scheduler.DailyInterval,
	}
}
