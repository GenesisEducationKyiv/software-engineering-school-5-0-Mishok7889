package external

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
	"weatherapi.app/pkg/errors"
)

// setupMockRedis creates a mock Redis server for testing
func setupMockRedis(t *testing.T) (*miniredis.Miniredis, *config.RedisConfig) {
	t.Helper()

	// Create mock Redis server
	mockRedis := miniredis.RunT(t)

	// Create Redis configuration pointing to mock server
	redisConfig := &config.RedisConfig{
		Addr:         mockRedis.Addr(),
		Password:     "",
		DB:           0,
		DialTimeout:  5,
		ReadTimeout:  3,
		WriteTimeout: 3,
	}

	return mockRedis, redisConfig
}

// TestRedisCacheProviderAdapter_NewRedisCacheProviderAdapter tests the constructor
func TestRedisCacheProviderAdapter_NewRedisCacheProviderAdapter(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.RedisConfig
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name:        "NilConfig",
			config:      nil,
			expectError: true,
			errorType:   errors.ErrorTypeConfiguration,
		},
		{
			name: "ValidConfig",
			config: func() *config.RedisConfig {
				_, cfg := setupMockRedis(t)
				return cfg
			}(),
			expectError: false,
		},
		{
			name: "InvalidAddress",
			config: &config.RedisConfig{
				Addr:         "invalid:address:port",
				Password:     "",
				DB:           0,
				DialTimeout:  5,
				ReadTimeout:  3,
				WriteTimeout: 3,
			},
			expectError: true,
			errorType:   errors.ErrorTypeExternalAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewRedisCacheProviderAdapter(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, adapter)
				if tt.errorType != errors.ErrorTypeUnknown {
					var appErr *errors.AppError
					if assert.ErrorAs(t, err, &appErr) {
						assert.Equal(t, tt.errorType, appErr.Type)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, adapter)
				if adapter != nil {
					assert.NoError(t, adapter.Close())
				}
			}
		})
	}
}

// TestRedisCacheProviderAdapter_Operations tests cache operations
func TestRedisCacheProviderAdapter_Operations(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Clear cache before testing
	require.NoError(t, adapter.Clear(ctx))

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := time.Minute

		err := adapter.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := adapter.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("GetNonExistentKey", func(t *testing.T) {
		key := "non-existent-key"

		retrieved, err := adapter.Get(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, retrieved)

		var appErr *errors.AppError
		if assert.ErrorAs(t, err, &appErr) {
			assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-key"
		value := []byte("delete-value")
		ttl := time.Minute

		err := adapter.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		err = adapter.Delete(ctx, key)
		require.NoError(t, err)

		_, err = adapter.Get(ctx, key)
		assert.Error(t, err)
	})

	t.Run("Exists", func(t *testing.T) {
		key := "exists-key"
		value := []byte("exists-value")
		ttl := time.Minute

		exists, err := adapter.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		err = adapter.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		exists, err = adapter.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("TTLExpiration", func(t *testing.T) {
		key := "ttl-key"
		value := []byte("ttl-value")
		ttl := 100 * time.Millisecond

		err := adapter.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := adapter.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Fast-forward time in mock Redis
		mockRedis.FastForward(150 * time.Millisecond)

		_, err = adapter.Get(ctx, key)
		assert.Error(t, err)
	})
}

// TestRedisCacheProviderAdapter_ValidationErrors tests validation error cases
func TestRedisCacheProviderAdapter_ValidationErrors(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	tests := []struct {
		name      string
		operation func() error
		errorType errors.ErrorType
	}{
		{
			name: "GetEmptyKey",
			operation: func() error {
				_, err := adapter.Get(ctx, "")
				return err
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "SetEmptyKey",
			operation: func() error {
				return adapter.Set(ctx, "", []byte("value"), time.Minute)
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "SetNilValue",
			operation: func() error {
				return adapter.Set(ctx, "key", nil, time.Minute)
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "SetZeroTTL",
			operation: func() error {
				return adapter.Set(ctx, "key", []byte("value"), 0)
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "SetNegativeTTL",
			operation: func() error {
				return adapter.Set(ctx, "key", []byte("value"), -time.Minute)
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "DeleteEmptyKey",
			operation: func() error {
				return adapter.Delete(ctx, "")
			},
			errorType: errors.ErrorTypeValidation,
		},
		{
			name: "ExistsEmptyKey",
			operation: func() error {
				_, err := adapter.Exists(ctx, "")
				return err
			},
			errorType: errors.ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()
			assert.Error(t, err)

			var appErr *errors.AppError
			if assert.ErrorAs(t, err, &appErr) {
				assert.Equal(t, tt.errorType, appErr.Type)
			}
		})
	}
}

// TestRedisCacheProviderAdapter_Metrics tests cache metrics functionality
func TestRedisCacheProviderAdapter_Metrics(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Clear cache and reset stats
	require.NoError(t, adapter.Clear(ctx))

	// Initial stats should be zero
	stats := adapter.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.TotalOps)
	assert.Equal(t, float64(0), stats.HitRatio)

	// Generate some hits and misses
	key := "metrics-key"
	value := []byte("metrics-value")
	ttl := time.Minute

	// Set a value
	err = adapter.Set(ctx, key, value, ttl)
	require.NoError(t, err)

	// Hit - get existing key
	_, err = adapter.Get(ctx, key)
	require.NoError(t, err)

	// Miss - get non-existent key
	_, err = adapter.Get(ctx, "non-existent")
	assert.Error(t, err)

	// Another hit
	_, err = adapter.Get(ctx, key)
	require.NoError(t, err)

	// Check stats: 2 hits, 1 miss
	stats = adapter.GetStats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(3), stats.TotalOps)
	assert.Equal(t, float64(2)/float64(3), stats.HitRatio)
	assert.True(t, stats.LastUpdated.After(time.Now().Add(-time.Second)))
}

// TestRedisCacheProviderAdapter_CacheInterface tests interface compliance
func TestRedisCacheProviderAdapter_CacheInterface(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	// Test that adapter implements CacheProvider interface
	var _ ports.CacheProvider = adapter

	// Test that adapter implements CacheMetrics interface
	var _ ports.CacheMetrics = adapter
}

// TestRedisCacheProviderAdapter_ContextCancellation tests context cancellation
func TestRedisCacheProviderAdapter_ContextCancellation(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// All operations should respect context cancellation
	_, err = adapter.Get(ctx, "key")
	assert.Error(t, err)

	err = adapter.Set(ctx, "key", []byte("value"), time.Minute)
	assert.Error(t, err)

	err = adapter.Delete(ctx, "key")
	assert.Error(t, err)

	_, err = adapter.Exists(ctx, "key")
	assert.Error(t, err)

	err = adapter.Clear(ctx)
	assert.Error(t, err)
}

// TestRedisCacheProviderAdapter_Ping tests connection health check
func TestRedisCacheProviderAdapter_Ping(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	err = adapter.Ping(ctx)
	assert.NoError(t, err)
}

// TestRedisCacheProviderAdapter_LargeData tests handling of large data
func TestRedisCacheProviderAdapter_LargeData(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Test with large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	key := "large-data-key"
	ttl := time.Minute

	err = adapter.Set(ctx, key, largeData, ttl)
	require.NoError(t, err)

	retrieved, err := adapter.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, largeData, retrieved)
}

// TestRedisCacheProviderAdapter_BinaryData tests handling of binary data
func TestRedisCacheProviderAdapter_BinaryData(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	ctx := context.Background()

	// Test with binary data including null bytes
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x00, 0x00}
	key := "binary-data-key"
	ttl := time.Minute

	err = adapter.Set(ctx, key, binaryData, ttl)
	require.NoError(t, err)

	retrieved, err := adapter.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, binaryData, retrieved)
}

// TestRedisCacheProviderAdapter_RecordMethods tests manual record methods
func TestRedisCacheProviderAdapter_RecordMethods(t *testing.T) {
	mockRedis, redisConfig := setupMockRedis(t)
	defer mockRedis.Close()

	adapter, err := NewRedisCacheProviderAdapter(redisConfig)
	require.NoError(t, err)
	defer func() { _ = adapter.Close() }()

	// Initial stats
	stats := adapter.GetStats()
	initialHits := stats.Hits
	initialMisses := stats.Misses

	// Manual recording
	adapter.RecordHit()
	adapter.RecordMiss()
	adapter.RecordOperation("test", time.Millisecond)

	// Check updated stats
	stats = adapter.GetStats()
	assert.Equal(t, initialHits+1, stats.Hits)
	assert.Equal(t, initialMisses+1, stats.Misses)
}
