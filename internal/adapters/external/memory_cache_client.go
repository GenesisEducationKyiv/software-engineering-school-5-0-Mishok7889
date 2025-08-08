package external

import (
	"context"
	"sync"
	"time"

	"weatherapi.app/internal/adapters/infrastructure"
	"weatherapi.app/internal/ports"
)

// MemoryCacheClient implements CacheProvider port using in-memory storage without metrics
type MemoryCacheClient struct {
	data  map[string]memoryCacheItem
	mutex sync.RWMutex
}

// NewMemoryCacheClient creates a new memory cache client
func NewMemoryCacheClient() *MemoryCacheClient {
	return &MemoryCacheClient{
		data: make(map[string]memoryCacheItem),
	}
}

func (c *MemoryCacheClient) ValidateKey(key string) error {
	if key == "" {
		return infrastructure.NewValidationError(ErrCacheKeyEmpty)
	}
	return nil
}

func (c *MemoryCacheClient) ValidateValue(value []byte) error {
	if value == nil {
		return infrastructure.NewValidationError(ErrCacheValueNil)
	}
	return nil
}

func (c *MemoryCacheClient) ValidateTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return infrastructure.NewValidationError(ErrCacheTTLNonPositive)
	}
	return nil
}

// Get retrieves a value from memory cache
func (c *MemoryCacheClient) Get(ctx context.Context, key string) ([]byte, error) {
	if err := c.ValidateKey(key); err != nil {
		return nil, err
	}

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists || time.Now().After(item.expiresAt) {
		return nil, ports.NewNotFoundError("cache miss")
	}

	return item.data, nil
}

// Set stores a value in memory cache with TTL
func (c *MemoryCacheClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := c.ValidateKey(key); err != nil {
		return err
	}
	if err := c.ValidateValue(value); err != nil {
		return err
	}
	if err := c.ValidateTTL(ttl); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = memoryCacheItem{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from memory cache
func (c *MemoryCacheClient) Delete(ctx context.Context, key string) error {
	if err := c.ValidateKey(key); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	return nil
}

// Exists checks if a key exists in memory cache
func (c *MemoryCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.ValidateKey(key); err != nil {
		return false, err
	}

	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return false, nil
	}

	return !time.Now().After(item.expiresAt), nil
}

// Clear removes all keys from memory cache
func (c *MemoryCacheClient) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]memoryCacheItem)
	return nil
}
