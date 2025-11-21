package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCache tests cache creation
func TestNewCache(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		enabled bool
	}{
		{
			name:    "enabled cache",
			ttl:     time.Hour,
			enabled: true,
		},
		{
			name:    "disabled cache",
			ttl:     time.Hour,
			enabled: false,
		},
		{
			name:    "short TTL",
			ttl:     time.Second,
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(tt.ttl, tt.enabled)
			require.NotNil(t, cache)
			assert.NotNil(t, cache.entries)
			assert.Equal(t, tt.ttl, cache.ttl)
			assert.Equal(t, tt.enabled, cache.enabled)
			assert.Equal(t, 0, cache.Size())
		})
	}
}

// TestCache_SetAndGet tests basic cache operations
func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(time.Hour, true)

	// Set value
	cache.Set("hello", "world")

	// Get value
	value, ok := cache.Get("hello")
	assert.True(t, ok, "Should find cached value")
	assert.Equal(t, "world", value)

	// Get non-existent value
	_, ok = cache.Get("non-existent")
	assert.False(t, ok, "Should not find non-existent value")
}

// TestCache_Disabled tests disabled cache behavior
func TestCache_Disabled(t *testing.T) {
	cache := NewCache(time.Hour, false)

	// Set should be no-op
	cache.Set("key", "value")

	// Get should return false
	value, ok := cache.Get("key")
	assert.False(t, ok, "Disabled cache should not return values")
	assert.Empty(t, value)

	// Delete should be no-op
	cache.Delete("key")

	// Size should be 0
	assert.Equal(t, 0, cache.Size())
}

// TestCache_Delete tests deletion
func TestCache_Delete(t *testing.T) {
	cache := NewCache(time.Hour, true)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	// Delete existing key
	cache.Delete("key1")
	assert.Equal(t, 1, cache.Size())

	_, ok := cache.Get("key1")
	assert.False(t, ok, "Deleted key should not exist")

	value, ok := cache.Get("key2")
	assert.True(t, ok, "Other keys should remain")
	assert.Equal(t, "value2", value)

	// Delete non-existent key (should not panic)
	cache.Delete("non-existent")
	assert.Equal(t, 1, cache.Size())
}

// TestCache_Clear tests clearing all entries
func TestCache_Clear(t *testing.T) {
	cache := NewCache(time.Hour, true)

	// Add multiple entries
	for i := 0; i < 10; i++ {
		cache.Set("key", "value")
	}
	assert.Greater(t, cache.Size(), 0)

	// Clear cache
	cache.Clear()
	assert.Equal(t, 0, cache.Size())

	// Verify all entries are gone
	_, ok := cache.Get("key")
	assert.False(t, ok)
}

// TestCache_Size tests size tracking
func TestCache_Size(t *testing.T) {
	cache := NewCache(time.Hour, true)

	assert.Equal(t, 0, cache.Size())

	cache.Set("key1", "value1")
	assert.Equal(t, 1, cache.Size())

	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	cache.Set("key1", "updated")
	assert.Equal(t, 2, cache.Size(), "Updating existing key should not increase size")

	cache.Delete("key1")
	assert.Equal(t, 1, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

// TestCache_Expiration tests TTL expiration
func TestCache_Expiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	cache := NewCache(200*time.Millisecond, true)

	cache.Set("key", "value")

	// Should be available immediately
	value, ok := cache.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", value)

	// Wait for expiration
	time.Sleep(250 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key")
	assert.False(t, ok, "Entry should be expired")
}

// TestCache_Stats tests statistics
func TestCache_Stats(t *testing.T) {
	cache := NewCache(time.Hour, true)

	// Empty cache
	stats := cache.Stats()
	assert.Equal(t, 0, stats.TotalEntries)
	assert.Equal(t, 0, stats.ValidEntries)
	assert.Equal(t, 0, stats.ExpiredEntries)
	assert.True(t, stats.Enabled)
	assert.Equal(t, time.Hour, stats.TTL)

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	stats = cache.Stats()
	assert.Equal(t, 2, stats.TotalEntries)
	assert.Equal(t, 2, stats.ValidEntries)
	assert.Equal(t, 0, stats.ExpiredEntries)
}

// TestCache_StatsWithExpired tests stats with expired entries
func TestCache_StatsWithExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	cache := NewCache(100*time.Millisecond, true)

	// Add entries
	cache.Set("key1", "value1")
	time.Sleep(150 * time.Millisecond)
	cache.Set("key2", "value2")

	stats := cache.Stats()
	assert.Equal(t, 2, stats.TotalEntries)
	assert.Equal(t, 1, stats.ValidEntries, "One entry should be valid")
	assert.Equal(t, 1, stats.ExpiredEntries, "One entry should be expired")
}

// TestCache_HashKey tests key hashing
func TestCache_HashKey(t *testing.T) {
	cache := NewCache(time.Hour, true)

	hash1 := cache.hashKey("test")
	hash2 := cache.hashKey("test")
	hash3 := cache.hashKey("different")

	// Same input should produce same hash
	assert.Equal(t, hash1, hash2)

	// Different input should produce different hash
	assert.NotEqual(t, hash1, hash3)

	// Hash should be hex encoded
	assert.Len(t, hash1, 64, "SHA256 hex should be 64 characters")
}

// TestCache_RemoveExpired tests manual expired entry removal
func TestCache_RemoveExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}

	cache := NewCache(100*time.Millisecond, true)

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Remove expired
	cache.removeExpired()

	// All entries should be removed
	assert.Equal(t, 0, cache.Size())
}

// TestCache_ThreadSafety tests concurrent access
func TestCache_ThreadSafety(t *testing.T) {
	cache := NewCache(time.Hour, true)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.Set("key", "value")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Get("key")
		}()
	}

	// Concurrent deletes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Delete("key")
		}()
	}

	// Concurrent size checks
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cache.Size()
		}()
	}

	// Concurrent stats
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cache.Stats()
		}()
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// TestCache_ConcurrentReadWrite tests concurrent read/write patterns
func TestCache_ConcurrentReadWrite(t *testing.T) {
	cache := NewCache(time.Hour, true)
	var wg sync.WaitGroup

	// Writer goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Set("key", "value")
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Get("key")
			}
		}()
	}

	wg.Wait()

	// Verify final state
	value, ok := cache.Get("key")
	if ok {
		assert.Equal(t, "value", value)
	}
}

// TestCache_MultipleKeys tests handling many different keys
func TestCache_MultipleKeys(t *testing.T) {
	cache := NewCache(time.Hour, true)

	// Add many keys
	keyCount := 1000
	for i := 0; i < keyCount; i++ {
		cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	assert.Equal(t, keyCount, cache.Size())

	// Verify all keys are retrievable
	for i := 0; i < keyCount; i++ {
		value, ok := cache.Get(fmt.Sprintf("key%d", i))
		assert.True(t, ok, "Key %d should exist", i)
		assert.Equal(t, fmt.Sprintf("value%d", i), value)
	}
}

// TestCache_UpdateValue tests updating existing values
func TestCache_UpdateValue(t *testing.T) {
	cache := NewCache(time.Hour, true)

	cache.Set("key", "value1")
	value, ok := cache.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value1", value)

	// Update value
	cache.Set("key", "value2")
	value, ok = cache.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value2", value)
	assert.Equal(t, 1, cache.Size(), "Size should not change on update")
}

// TestCache_EmptyValues tests caching empty strings
func TestCache_EmptyValues(t *testing.T) {
	cache := NewCache(time.Hour, true)

	cache.Set("empty", "")
	value, ok := cache.Get("empty")
	assert.True(t, ok, "Should be able to cache empty strings")
	assert.Equal(t, "", value)
}

// TestCache_LongValues tests caching long strings
func TestCache_LongValues(t *testing.T) {
	cache := NewCache(time.Hour, true)

	// Create a very long value
	longValue := ""
	for i := 0; i < 10000; i++ {
		longValue += "x"
	}

	cache.Set("long", longValue)
	value, ok := cache.Get("long")
	assert.True(t, ok)
	assert.Equal(t, longValue, value)
	assert.Len(t, value, 10000)
}

// BenchmarkCache_Set benchmarks cache Set operation
func BenchmarkCache_Set(b *testing.B) {
	cache := NewCache(time.Hour, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", "value")
	}
}

// BenchmarkCache_Get benchmarks cache Get operation
func BenchmarkCache_Get(b *testing.B) {
	cache := NewCache(time.Hour, true)
	cache.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

// BenchmarkCache_GetMiss benchmarks cache misses
func BenchmarkCache_GetMiss(b *testing.B) {
	cache := NewCache(time.Hour, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent")
	}
}

// BenchmarkCache_Delete benchmarks cache Delete operation
func BenchmarkCache_Delete(b *testing.B) {
	cache := NewCache(time.Hour, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", "value")
		cache.Delete("key")
	}
}

// BenchmarkCache_Size benchmarks Size operation
func BenchmarkCache_Size(b *testing.B) {
	cache := NewCache(time.Hour, true)
	cache.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Size()
	}
}

// BenchmarkCache_Stats benchmarks Stats operation
func BenchmarkCache_Stats(b *testing.B) {
	cache := NewCache(time.Hour, true)
	for i := 0; i < 100; i++ {
		cache.Set("key", "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Stats()
	}
}

// BenchmarkCache_HashKey benchmarks key hashing
func BenchmarkCache_HashKey(b *testing.B) {
	cache := NewCache(time.Hour, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.hashKey("test key for benchmarking")
	}
}

// BenchmarkCache_ConcurrentReadWrite benchmarks concurrent access
func BenchmarkCache_ConcurrentReadWrite(b *testing.B) {
	cache := NewCache(time.Hour, true)
	cache.Set("key", "value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if b.N%2 == 0 {
				cache.Get("key")
			} else {
				cache.Set("key", "value")
			}
		}
	})
}
