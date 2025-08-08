package external

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/config"
	"weatherapi.app/internal/ports"
)

// setupMockRedisForClient creates a mock Redis server for testing the client
func setupMockRedisForClient(t *testing.T) (*miniredis.Miniredis, *config.RedisConfig) {
	t.Helper()

	mockRedis := miniredis.RunT(t)

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

// TestRedisCacheClient_NewRedisCacheClient tests the constructor
func TestRedisCacheClient_NewRedisCacheClient(t *testing.T) {
	ctx := context.Background()

	t.Run("ValidConfig", func(t *testing.T) {
		mockRedis, redisConfig := setupMockRedisForClient(t)
		defer mockRedis.Close()

		client, err := NewRedisCacheClient(ctx, redisConfig)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		if client != nil {
			assert.NoError(t, client.Close())
		}
	})

	t.Run("NilConfig", func(t *testing.T) {
		client, err := NewRedisCacheClient(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, client)

		var infraErr *infrastructure.InfrastructureError
		if assert.ErrorAs(t, err, &infraErr) {
			assert.Equal(t, "CONFIGURATION_ERROR", infraErr.Type)
		}
	})

	t.Run("InvalidAddress", func(t *testing.T) {
		invalidConfig := &config.RedisConfig{
			Addr:         "invalid:address:port",
			Password:     "",
			DB:           0,
			DialTimeout:  5,
			ReadTimeout:  3,
			WriteTimeout: 3,
		}

		client, err := NewRedisCacheClient(ctx, invalidConfig)
		assert.Error(t, err)
		assert.Nil(t, client)

		var infraErr *infrastructure.InfrastructureError
		if assert.ErrorAs(t, err, &infraErr) {
			assert.Equal(t, "EXTERNAL_API_ERROR", infraErr.Type)
		}
	})
}

// TestRedisCacheClient_BasicOperations tests successful cache operations
func TestRedisCacheClient_BasicOperations(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	require.NoError(t, client.Clear(ctx))

	t.Run("SetAndGet", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")
		ttl := time.Minute

		err := client.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	t.Run("GetNonExistentKey", func(t *testing.T) {
		key := "non-existent-key"

		retrieved, err := client.Get(ctx, key)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.True(t, ports.IsNotFoundError(err))
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-key"
		value := []byte("delete-value")
		ttl := time.Minute

		err := client.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		err = client.Delete(ctx, key)
		require.NoError(t, err)

		_, err = client.Get(ctx, key)
		assert.Error(t, err)
		assert.True(t, ports.IsNotFoundError(err))
	})

	t.Run("Exists", func(t *testing.T) {
		key := "exists-key"
		value := []byte("exists-value")
		ttl := time.Minute

		exists, err := client.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		err = client.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		exists, err = client.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("TTLExpiration", func(t *testing.T) {
		key := "ttl-key"
		value := []byte("ttl-value")
		ttl := 100 * time.Millisecond

		err := client.Set(ctx, key, value, ttl)
		require.NoError(t, err)

		retrieved, err := client.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		mockRedis.FastForward(150 * time.Millisecond)

		_, err = client.Get(ctx, key)
		assert.Error(t, err)
		assert.True(t, ports.IsNotFoundError(err))
	})
}

// TestRedisCacheClient_ValidationErrors tests validation error cases
func TestRedisCacheClient_ValidationErrors(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	t.Run("EmptyKeyValidation", func(t *testing.T) {
		_, err := client.Get(ctx, "")
		assert.Error(t, err)

		var infraErr *infrastructure.InfrastructureError
		if assert.ErrorAs(t, err, &infraErr) {
			assert.Equal(t, "VALIDATION_ERROR", infraErr.Type)
		}

		err = client.Set(ctx, "", []byte("value"), time.Minute)
		assert.Error(t, err)
		assert.ErrorAs(t, err, &infraErr)

		err = client.Delete(ctx, "")
		assert.Error(t, err)
		assert.ErrorAs(t, err, &infraErr)

		_, err = client.Exists(ctx, "")
		assert.Error(t, err)
		assert.ErrorAs(t, err, &infraErr)
	})

	t.Run("NilValueValidation", func(t *testing.T) {
		err := client.Set(ctx, "key", nil, time.Minute)
		assert.Error(t, err)

		var infraErr *infrastructure.InfrastructureError
		if assert.ErrorAs(t, err, &infraErr) {
			assert.Equal(t, "VALIDATION_ERROR", infraErr.Type)
		}
	})

	t.Run("InvalidTTLValidation", func(t *testing.T) {
		err := client.Set(ctx, "key", []byte("value"), 0)
		assert.Error(t, err)

		var infraErr *infrastructure.InfrastructureError
		if assert.ErrorAs(t, err, &infraErr) {
			assert.Equal(t, "VALIDATION_ERROR", infraErr.Type)
		}

		err = client.Set(ctx, "key", []byte("value"), -time.Minute)
		assert.Error(t, err)
		assert.ErrorAs(t, err, &infraErr)
	})
}

// TestRedisCacheClient_InterfaceCompliance tests interface compliance
func TestRedisCacheClient_InterfaceCompliance(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	// Test CacheProvider interface compliance
	var _ ports.CacheProvider = client
}

// TestRedisCacheClient_ContextCancellation tests context cancellation
func TestRedisCacheClient_ContextCancellation(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.Get(ctx, "key")
	assert.Error(t, err)

	err = client.Set(ctx, "key", []byte("value"), time.Minute)
	assert.Error(t, err)

	err = client.Delete(ctx, "key")
	assert.Error(t, err)

	_, err = client.Exists(ctx, "key")
	assert.Error(t, err)

	err = client.Clear(ctx)
	assert.Error(t, err)
}

// TestRedisCacheClient_Ping tests connection health check
func TestRedisCacheClient_Ping(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	err = client.Ping(ctx)
	assert.NoError(t, err)
}

// TestRedisCacheClient_LargeData tests handling of large data
func TestRedisCacheClient_LargeData(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	key := "large-data-key"
	ttl := time.Minute

	err = client.Set(ctx, key, largeData, ttl)
	require.NoError(t, err)

	retrieved, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, largeData, retrieved)
}

// TestRedisCacheClient_BinaryData tests handling of binary data
func TestRedisCacheClient_BinaryData(t *testing.T) {
	mockRedis, redisConfig := setupMockRedisForClient(t)
	defer mockRedis.Close()

	client, err := NewRedisCacheClient(context.Background(), redisConfig)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x00, 0x00}
	key := "binary-data-key"
	ttl := time.Minute

	err = client.Set(ctx, key, binaryData, ttl)
	require.NoError(t, err)

	retrieved, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, binaryData, retrieved)
}
