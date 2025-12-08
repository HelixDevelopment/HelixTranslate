package distributed

import (
	"testing"
	"time"
)

// MockSecurityLoggerTest for testing - implements distributed.Logger interface
type MockSecurityLoggerTest struct{}

func (m *MockSecurityLoggerTest) Log(level, message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func TestDefaultPerformanceConfig(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		
		// Test connection pooling defaults
		if config.MaxConnectionsPerWorker != 10 {
			t.Errorf("Expected max connections per worker to be 10, got %d", config.MaxConnectionsPerWorker)
		}
		
		if config.ConnectionIdleTimeout != 5*time.Minute {
			t.Errorf("Expected connection idle timeout to be 5m, got %v", config.ConnectionIdleTimeout)
		}
		
		if config.ConnectionMaxLifetime != 30*time.Minute {
			t.Errorf("Expected connection max lifetime to be 30m, got %v", config.ConnectionMaxLifetime)
		}
		
		// Test batching defaults
		if !config.EnableBatching {
			t.Error("Expected batching to be enabled")
		}
		
		if config.BatchSize != 10 {
			t.Errorf("Expected batch size to be 10, got %d", config.BatchSize)
		}
		
		if config.BatchTimeout != 100*time.Millisecond {
			t.Errorf("Expected batch timeout to be 100ms, got %v", config.BatchTimeout)
		}
		
		// Test caching defaults
		if !config.EnableResultCaching {
			t.Error("Expected result caching to be enabled")
		}
		
		if config.CacheTTL != 10*time.Minute {
			t.Errorf("Expected cache TTL to be 10m, got %v", config.CacheTTL)
		}
		
		if config.MaxCacheSize != 10000 {
			t.Errorf("Expected max cache size to be 10000, got %d", config.MaxCacheSize)
		}
		
		// Test load balancing defaults
		if config.LoadBalancingStrategy != "least_loaded" {
			t.Errorf("Expected load balancing strategy to be 'least_loaded', got '%s'", config.LoadBalancingStrategy)
		}
		
		if config.HealthCheckInterval != 30*time.Second {
			t.Errorf("Expected health check interval to be 30s, got %v", config.HealthCheckInterval)
		}
		
		// Test circuit breaker defaults
		if !config.EnableCircuitBreaker {
			t.Error("Expected circuit breaker to be enabled")
		}
		
		if config.FailureThreshold != 5 {
			t.Errorf("Expected failure threshold to be 5, got %d", config.FailureThreshold)
		}
		
		if config.RecoveryTimeout != 60*time.Second {
			t.Errorf("Expected recovery timeout to be 60s, got %v", config.RecoveryTimeout)
		}
		
		if config.SuccessThreshold != 3 {
			t.Errorf("Expected success threshold to be 3, got %d", config.SuccessThreshold)
		}
		
		// Test metrics defaults
		if !config.EnableMetrics {
			t.Error("Expected metrics to be enabled")
		}
		
		if config.MetricsInterval != 10*time.Second {
			t.Errorf("Expected metrics interval to be 10s, got %v", config.MetricsInterval)
		}
	})
}

func TestNewConnectionPool(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		security := DefaultSecurityConfig()
		auditor := NewSecurityAuditor(false, &MockSecurityLoggerTest{})
		
		pool := NewConnectionPool(config, security, auditor)
		
		if pool == nil {
			t.Error("Expected non-nil connection pool")
		}
		
		if pool.config != config {
			t.Error("Expected config to be set correctly")
		}
		
		if pool.security != security {
			t.Error("Expected security to be set correctly")
		}
		
		if pool.auditor != auditor {
			t.Error("Expected auditor to be set correctly")
		}
	})
}

func TestNewResultCache(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		if cache == nil {
			t.Error("Expected non-nil result cache")
		}
		
		// Test that it's empty initially
		// No easy way to check cache size without adding size tracking
		// This will be tested through Get/Set operations
	})
}

func TestResultCache_BasicOperations(t *testing.T) {
	t.Run("GetAndSet", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		key := "test-key"
		value := "test-value"
		
		// Test getting non-existent value
		result, found := cache.Get(key)
		if found {
			t.Error("Expected not to find value")
		}
		if result != "" {
			t.Errorf("Expected empty result for non-existent key, got '%s'", result)
		}
		
		// Test setting a value
		cache.Set(key, value)
		
		// Test getting existing value
		result, found = cache.Get(key)
		if !found {
			t.Error("Expected to find value")
		}
		
		if result != value {
			t.Errorf("Expected value '%s', got '%s'", value, result)
		}
	})
}

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cb := NewCircuitBreaker(config.FailureThreshold, config.RecoveryTimeout, config.SuccessThreshold)
		
		if cb == nil {
			t.Error("Expected non-nil circuit breaker")
		}
		
		// Test initial state
		state := cb.GetState()
		if state != StateClosed {
			t.Errorf("Expected initial state to be StateClosed, got %v", state)
		}
	})
}

func TestNewBatchProcessor(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		processor := NewBatchProcessor(config.BatchSize, config.BatchTimeout, func(requests []interface{}) error {
			// Mock processing function
			return nil
		})
		
		if processor == nil {
			t.Error("Expected non-nil batch processor")
		}
	})
}

func TestGenerateCacheKey(t *testing.T) {
	t.Run("CacheKeyGeneration", func(t *testing.T) {
		config := DefaultPerformanceConfig()
		cache := NewResultCache(config)
		
		key := cache.GenerateCacheKey("test-text", "", "test-provider", "test-model")
		
		if key == "" {
			t.Error("Expected non-empty cache key")
		}
		
		// Test that the same inputs generate the same key
		key2 := cache.GenerateCacheKey("test-text", "", "test-provider", "test-model")
		if key != key2 {
			t.Error("Expected same inputs to generate same cache key")
		}
		
		// Test that different inputs generate different keys
		key3 := cache.GenerateCacheKey("different-text", "", "test-provider", "test-model")
		if key == key3 {
			t.Error("Expected different inputs to generate different cache keys")
		}
	})
}