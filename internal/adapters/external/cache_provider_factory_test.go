package external

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// TestCacheProviderFactory_CreateCacheProvider tests the factory
func TestCacheProviderFactory_CreateCacheProvider(t *testing.T) {
	factory := NewCacheProviderFactory()
	ctx := context.Background()

	tests := []struct {
		name         string
		config       *config.CacheConfig
		expectError  bool
		expectedType string
	}{
		{
			name:        "NilConfig",
			config:      nil,
			expectError: true,
		},
		{
			name: "MemoryCache",
			config: &config.CacheConfig{
				Type: config.CacheTypeMemory,
			},
			expectError:  false,
			expectedType: "*external.CacheMetricsDecorator",
		},
		{
			name: "RedisCache",
			config: &config.CacheConfig{
				Type: config.CacheTypeRedis,
				Redis: config.RedisConfig{
					Addr:         "localhost:6379",
					Password:     "",
					DB:           0,
					DialTimeout:  5,
					ReadTimeout:  3,
					WriteTimeout: 3,
				},
			},
			expectError:  false,
			expectedType: "*external.CacheMetricsDecorator",
		},
		{
			name: "UnknownCacheType",
			config: &config.CacheConfig{
				Type: config.CacheTypeUnknown,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateCacheProvider(ctx, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				if tt.config != nil && tt.config.Type == config.CacheTypeRedis && err != nil {
					t.Skipf("Skipping Redis test due to connection error: %v", err)
				}
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				// Verify the provider is a CacheMetricsDecorator which implements both interfaces
				decorator, ok := provider.(*CacheMetricsDecorator)
				assert.True(t, ok, "Provider should be a CacheMetricsDecorator")

				// Verify it implements both interfaces through the decorator
				var _ ports.CacheProvider = decorator
				var _ ports.CacheMetrics = decorator
			}
		})
	}
}

// TestCacheProviderFactory_MemoryCacheOperations tests memory cache operations through factory
func TestCacheProviderFactory_MemoryCacheOperations(t *testing.T) {
	factory := NewCacheProviderFactory()
	ctx := context.Background()

	config := &config.CacheConfig{
		Type: config.CacheTypeMemory,
	}

	provider, err := factory.CreateCacheProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Clear cache before testing
	require.NoError(t, provider.Clear(ctx))

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := time.Minute

		err := provider.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := provider.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("GetNonExistentKey", func(t *testing.T) {
		key := "non-existent-key"

		retrieved, err := provider.Get(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.True(t, ports.IsNotFoundError(err))
	})

	t.Run("MetricsTracking", func(t *testing.T) {
		// Cast to metrics interface
		metricsProvider, ok := provider.(*CacheMetricsDecorator)
		require.True(t, ok, "Provider should be CacheMetricsDecorator")

		// Clear and get baseline stats
		require.NoError(t, provider.Clear(ctx))
		baselineStats := metricsProvider.GetStats()

		// Set a value and generate hits/misses
		key := "metrics-key"
		value := []byte("metrics-value")
		ttl := time.Minute

		err := provider.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		// Hit
		_, err = provider.Get(ctx, key)
		require.NoError(t, err)

		// Miss
		_, err = provider.Get(ctx, "non-existent")
		assert.Error(t, err)

		// Check stats relative to baseline
		stats := metricsProvider.GetStats()
		expectedHits := baselineStats.Hits + 1
		expectedMisses := baselineStats.Misses + 1
		expectedTotal := baselineStats.TotalOps + 2

		assert.Equal(t, expectedHits, stats.Hits)
		assert.Equal(t, expectedMisses, stats.Misses)
		assert.Equal(t, expectedTotal, stats.TotalOps)

		// Calculate hit ratio based on the new operations only
		if expectedTotal > baselineStats.TotalOps {
			newOps := expectedTotal - baselineStats.TotalOps
			newHits := expectedHits - baselineStats.Hits
			expectedHitRatio := float64(newHits) / float64(newOps)
			actualHitRatio := float64(stats.Hits-baselineStats.Hits) / float64(stats.TotalOps-baselineStats.TotalOps)
			assert.InDelta(t, expectedHitRatio, actualHitRatio, 0.01)
		}
	})
}

// TestCacheProviderFactory_ValidationErrors tests validation error cases
func TestCacheProviderFactory_ValidationErrors(t *testing.T) {
	factory := NewCacheProviderFactory()
	ctx := context.Background()

	config := &config.CacheConfig{
		Type: config.CacheTypeMemory,
	}

	provider, err := factory.CreateCacheProvider(ctx, config)
	require.NoError(t, err)

	tests := []struct {
		name      string
		operation func() error
		errorType string
	}{
		{
			name: "GetEmptyKey",
			operation: func() error {
				_, err := provider.Get(ctx, "")
				return err
			},
			errorType: "VALIDATION_ERROR",
		},
		{
			name: "SetEmptyKey",
			operation: func() error {
				return provider.Set(ctx, "", []byte("value"), time.Minute)
			},
			errorType: "VALIDATION_ERROR",
		},
		{
			name: "SetNilValue",
			operation: func() error {
				return provider.Set(ctx, "key", nil, time.Minute)
			},
			errorType: "VALIDATION_ERROR",
		},
		{
			name: "SetZeroTTL",
			operation: func() error {
				return provider.Set(ctx, "key", []byte("value"), 0)
			},
			errorType: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			assert.Error(t, err)

			var infraErr *infrastructure.InfrastructureError
			if assert.ErrorAs(t, err, &infraErr) {
				assert.Equal(t, tt.errorType, infraErr.Type)
			}
		})
	}
}
