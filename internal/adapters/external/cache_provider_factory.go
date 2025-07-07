package external

import (
	"fmt"

	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

type CacheProviderFactory struct{}

func NewCacheProviderFactory() *CacheProviderFactory {
	return &CacheProviderFactory{}
}

func (f *CacheProviderFactory) CreateCacheProvider(cfg *config.CacheConfig) (ports.CacheProvider, error) {
	if cfg == nil {
		return nil, errors.NewConfigurationError("cache config cannot be nil", nil)
	}

	switch cfg.Type {
	case config.CacheTypeMemory:
		return NewMemoryCacheProvider(), nil
	case config.CacheTypeRedis:
		return NewRedisCacheProviderAdapter(&cfg.Redis)
	default:
		return nil, errors.NewConfigurationError(
			fmt.Sprintf("unsupported cache type: %s", cfg.Type.String()), nil)
	}
}
