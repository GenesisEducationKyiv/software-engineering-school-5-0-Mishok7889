package ports

import (
	"context"
	"time"
)

// CacheProvider defines the contract for caching operations
type CacheProvider interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
}

// CacheMetrics defines the contract for cache performance tracking
type CacheMetrics interface {
	GetStats() CacheStats
	RecordHit()
	RecordMiss()
	RecordOperation(operation string, duration time.Duration)
	GetOperationMetrics() map[string]OperationMetrics
}

// OperationMetrics represents metrics for a specific cache operation
type OperationMetrics struct {
	Count        int64         `json:"count"`
	TotalLatency time.Duration `json:"total_latency"`
	AvgLatency   time.Duration `json:"avg_latency"`
	MaxLatency   time.Duration `json:"max_latency"`
	MinLatency   time.Duration `json:"min_latency"`
}

// CacheSerializer defines the contract for data serialization
type CacheSerializer interface {
	Serialize(data interface{}) ([]byte, error)
	Deserialize(data []byte, target interface{}) error
}
