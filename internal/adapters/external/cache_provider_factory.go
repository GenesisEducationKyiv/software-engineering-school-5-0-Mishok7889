package external

import (
	"fmt"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

type CacheProviderFactory struct{}

func NewCacheProviderFactory() *CacheProviderFactory {
	return &CacheProviderFactory{}
}

func (f *CacheProviderFactory) CreateCacheProvider(cfg *config.CacheConfig) (ports.CacheProvider, error) {
	if cfg == nil {
		return nil, infrastructure.NewConfigurationError("cache config cannot be nil")
	}

	switch cfg.Type {
	case config.CacheTypeMemory:
		return NewMemoryCacheProvider(), nil
	case config.CacheTypeRedis:
		return NewRedisCacheProviderAdapter(&cfg.Redis)
	default:
		return nil, infrastructure.NewConfigurationError(
			fmt.Sprintf("unsupported cache type: %s", cfg.Type.String()))
	}
}
