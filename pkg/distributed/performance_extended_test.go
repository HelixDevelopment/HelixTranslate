package distributed

import (
	"testing"
	"time"
)

func TestPerformance_ConnectionPool(t *testing.T) {
	t.Run("NewConnectionPool", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		security := DefaultSecurityConfig()
		auditor := NewSecurityAuditor(false, &MockSecurityLogger{})
		
		pool := NewConnectionPool(config, security, auditor)
		if pool == nil {
			t.Error("Expected non-nil connection pool")
		}
		
		// Test GetPoolStats
		stats := pool.GetPoolStats()
		if stats == nil {
			t.Error("Expected non-nil stats")
		}
		
		// Check initial stats values
		if total, ok := stats["total_connections"].(int); !ok || total != 0 {
			t.Errorf("Expected total_connections to be 0, got %v", stats["total_connections"])
		}
	})
	
	t.Run("GetConnection", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		security := DefaultSecurityConfig()
		auditor := NewSecurityAuditor(false, &MockSecurityLogger{})
		
		pool := NewConnectionPool(config, security, auditor)
		
		workerConfig := NewWorkerConfig("worker1", "Test Worker", "127.0.0.1", "testuser")
		
		// Try to get connection (should fail since we can't actually connect)
		_, err := pool.GetConnection("worker1", workerConfig)
		if err == nil {
			t.Error("Expected error getting connection with test config")
		}
	})
	
	t.Run("ReturnConnection", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		security := DefaultSecurityConfig()
		auditor := NewSecurityAuditor(false, &MockSecurityLogger{})
		
		pool := NewConnectionPool(config, security, auditor)
		
		// Create a mock connection
		// Add a connection to pool
		pool.connections["worker1"] = &ConnectionPoolEntry{
			Connection: &SSHConnection{
				Config: NewWorkerConfig("worker1", "Test Worker", "127.0.0.1", "testuser"),
			},
			LastUsed:   time.Now(),
			CreatedAt:  time.Now(),
			InUse:      false,
		}
		
		// Return connection (should not panic)
		pool.ReturnConnection("worker1")
	})
	
	t.Run("RemoveConnection", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		security := DefaultSecurityConfig()
		auditor := NewSecurityAuditor(false, &MockSecurityLogger{})
		
		pool := NewConnectionPool(config, security, auditor)
		
		// Remove connection (should not panic)
		pool.RemoveConnection("worker1")
	})
}

func TestPerformance_ResultCache(t *testing.T) {
	t.Run("NewResultCache", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		if cache == nil {
			t.Error("Expected non-nil result cache")
		}
	})
	
	t.Run("CacheOperations", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		// Test cache key generation
		key := cache.GenerateCacheKey("hello", "context", "provider", "model")
		if key == "" {
			t.Error("Expected non-empty cache key")
		}
		
		// Test getting non-existent key
		_, found := cache.Get("non-existent")
		if found {
			t.Error("Expected not to find non-existent key")
		}
		
		// Test setting and getting
		cache.Set("test-key", "test-value")
		value, found := cache.Get("test-key")
		if !found {
			t.Error("Expected to find test-key")
		}
		if value != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", value)
		}
	})
}

func TestPerformance_CircuitBreaker(t *testing.T) {
	t.Run("NewCircuitBreaker", func(t *testing.T) {
		cb := NewCircuitBreaker(3, time.Second, 2)
		
		if cb == nil {
			t.Error("Expected non-nil circuit breaker")
		}
		
		// Test initial state
		state := cb.GetState()
		if state != StateClosed {
			t.Errorf("Expected initial state to be StateClosed, got %v", state)
		}
	})
	
	t.Run("CircuitBreakerStates", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 100*time.Millisecond, 1)
		
		// Test successful call
		err := cb.Call(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected no error for successful call, got %v", err)
		}
		
		// Test failed call
		err = cb.Call(func() error {
			return &MockError{message: "test error"}
		})
		if err == nil {
			t.Error("Expected error for failed call")
		}
		
		// State should be OPEN now
		state := cb.GetState()
		if state != StateOpen {
			t.Errorf("Expected state to be StateOpen after failure, got %v", state)
		}
	})
}

func TestPerformance_BatchProcessor(t *testing.T) {
	t.Run("NewBatchProcessor", func(t *testing.T) {
		processFn := func(requests []interface{}) error {
			return nil
		}
		
		processor := NewBatchProcessor(10, time.Second, processFn)
		if processor == nil {
			t.Error("Expected non-nil batch processor")
		}
	})
	
	t.Run("AddRequest", func(t *testing.T) {
		processFn := func(requests []interface{}) error {
			return nil
		}
		
		processor := NewBatchProcessor(10, time.Second, processFn)
		
		// Add request
		err := processor.AddRequest("batch1", "test-request")
		if err != nil {
			t.Errorf("Expected no error adding request, got %v", err)
		}
	})
	
	t.Run("FlushAll", func(t *testing.T) {
		processFn := func(requests []interface{}) error {
			return nil
		}
		
		processor := NewBatchProcessor(10, time.Second, processFn)
		
		// Add a request and flush
		processor.AddRequest("batch1", "test-request")
		processor.FlushAll()
		
		// Should not panic
	})
}

// Mock error for testing
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

func TestResultCache_RemoveExpired(t *testing.T) {
	t.Run("RemoveExpiredEntries", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		// Add some entries with different expiration times
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")
		
		// Manually expire an entry by setting its expiration time in the past
		if entry, exists := cache.cache["key1"]; exists {
			entry.ExpiresAt = time.Now().Add(-time.Hour)
		}
		
		// Remove expired entries
		cache.removeExpired()
		
		// Check that expired entry was removed
		if _, exists := cache.cache["key1"]; exists {
			t.Error("Expected expired entry to be removed")
		}
		
		// Check that non-expired entry still exists
		if _, exists := cache.cache["key2"]; !exists {
			t.Error("Expected non-expired entry to still exist")
		}
	})
	
	t.Run("RemoveFromEmptyCache", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		// Remove expired from empty cache (should not panic)
		cache.removeExpired()
		
		// Should not panic
	})
}

func TestResultCache_RemoveOldest(t *testing.T) {
	t.Run("RemoveOldestEntry", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		// Add some entries
		cache.Set("key1", "value1")
		time.Sleep(time.Millisecond * 10) // Ensure different timestamps
		cache.Set("key2", "value2")
		time.Sleep(time.Millisecond * 10) // Ensure different timestamps
		cache.Set("key3", "value3")
		
		// Remove oldest entry
		cache.removeOldest()
		
		// Check that oldest entry was removed
		if _, exists := cache.cache["key1"]; exists {
			t.Error("Expected oldest entry (key1) to be removed")
		}
		
		// Check that other entries still exist
		if _, exists := cache.cache["key2"]; !exists {
			t.Error("Expected key2 to still exist")
		}
		if _, exists := cache.cache["key3"]; !exists {
			t.Error("Expected key3 to still exist")
		}
	})
	
	t.Run("RemoveFromEmptyCache", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		// Remove oldest from empty cache (should not panic)
		cache.removeOldest()
		
		// Should not panic
	})
}