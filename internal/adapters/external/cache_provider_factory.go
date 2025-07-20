package external

import (
	"context"
	"fmt"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

type CacheProviderFactory struct{}

func NewCacheProviderFactory() *CacheProviderFactory {
	return &CacheProviderFactory{}
}

func (f *CacheProviderFactory) CreateCacheProvider(ctx context.Context, cfg *config.CacheConfig) (ports.CacheProvider, error) {
	if cfg == nil {
		return nil, infrastructure.NewConfigurationError("cache config cannot be nil")
	}

	var baseProvider ports.CacheProvider
	var err error

	switch cfg.Type {
	case config.CacheTypeMemory:
		baseProvider = NewMemoryCacheClient()
	case config.CacheTypeRedis:
		baseProvider, err = NewRedisCacheClient(ctx, &cfg.Redis)
		if err != nil {
			return nil, err
		}
	default:
		return nil, infrastructure.NewConfigurationError(
			fmt.Sprintf("unsupported cache type: %s", cfg.Type.String()))
	}

	// Wrap base provider with metrics decorator
	return NewCacheMetricsDecorator(baseProvider), nil
}
