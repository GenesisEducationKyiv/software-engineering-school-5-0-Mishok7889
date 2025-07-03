package providers

import (
	"context"
	"testing"
	"time"

	"weatherapi.app/providers/cache"
)

// This test verifies that the new generic cache architecture works correctly
func TestInstrumentedCacheIntegration(t *testing.T) {
	// Create memory cache
	memCache := cache.NewMemoryCache()

	// Create instrumented cache
	instrumentedCache := NewInstrumentedCache(memCache, "memory")

	// Test basic operations
	key := "test:weather:london"
	testData := []byte(`{"temperature":25.0,"humidity":60.0,"description":"Sunny"}`)

	// Test Set and Get
	instrumentedCache.Set(context.Background(), key, testData, time.Minute)
	result, found := instrumentedCache.Get(context.Background(), key)

	if !found {
		t.Error("Expected to find cached data")
	}

	if string(result) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(result))
	}

	// Verify metrics are collected
	metrics := instrumentedCache.GetMetrics()
	stats := metrics.GetStats()

	if stats["total"].(int64) < 1 {
		t.Error("Expected metrics to be recorded")
	}

	if stats["hits"].(int64) < 1 {
		t.Error("Expected at least one cache hit")
	}
}
