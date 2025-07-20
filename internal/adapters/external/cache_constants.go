package external

import "time"

// Cache error message constants
const (
	ErrCacheKeyEmpty         = "cache key cannot be empty"
	ErrCacheValueNil         = "cache value cannot be nil"
	ErrCacheTTLNonPositive   = "cache TTL must be positive"
	DefaultConnectionTimeout = 5 * time.Second
)
