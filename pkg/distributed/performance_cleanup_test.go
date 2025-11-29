package distributed

import (
	"testing"
	"time"
)

func TestConnectionPool_cleanup(t *testing.T) {
	t.Run("cleanup_IdleConnections", func(t *testing.T) {
		// Create a connection pool with very short timeouts for testing
		config := DefaultPerformanceConfig()
		config.ConnectionIdleTimeout = 10 * time.Millisecond
		config.CacheCleanupInterval = 20 * time.Millisecond
		
		securityConfig := &SecurityConfig{}
		auditor := &SecurityAuditor{}
		
		cp := NewConnectionPool(config, securityConfig, auditor)
		
		// Add a mock SSH connection
		mockConn := &SSHConnection{
			Client: nil, // nil client is fine for this test
		}
		
		// Add a connection that's idle
		entry := &ConnectionPoolEntry{
			Connection: mockConn,
			CreatedAt:  time.Now(),
			LastUsed:   time.Now().Add(-time.Hour), // Idle for a long time
			InUse:      false,
		}
		cp.connections["test-key"] = entry
		
		// Wait for cleanup to run (ticker interval)
		time.Sleep(50 * time.Millisecond)
		
		// Check that idle connection was removed
		if _, exists := cp.connections["test-key"]; exists {
			t.Error("Expected idle connection to be removed")
		}
	})
	
	t.Run("cleanup_ActiveConnections", func(t *testing.T) {
		// Create a connection pool with very short timeouts for testing
		config := DefaultPerformanceConfig()
		config.ConnectionIdleTimeout = 10 * time.Millisecond
		config.CacheCleanupInterval = 20 * time.Millisecond
		
		securityConfig := &SecurityConfig{}
		auditor := &SecurityAuditor{}
		
		cp := NewConnectionPool(config, securityConfig, auditor)
		
		// Add a mock SSH connection
		mockConn := &SSHConnection{
			Client: nil,
		}
		
		// Add a connection that's actively in use
		entry := &ConnectionPoolEntry{
			Connection: mockConn,
			CreatedAt:  time.Now(),
			LastUsed:   time.Now().Add(-time.Hour), // Idle for a long time
			InUse:      true, // But marked as in use
		}
		cp.connections["test-key"] = entry
		
		// Wait for cleanup to run (ticker interval)
		time.Sleep(50 * time.Millisecond)
		
		// Check that active connection was NOT removed
		if _, exists := cp.connections["test-key"]; !exists {
			t.Error("Expected active connection to NOT be removed")
		}
	})
	
	t.Run("cleanup_ExpiredConnections", func(t *testing.T) {
		// Create a connection pool with very short timeouts for testing
		config := DefaultPerformanceConfig()
		config.ConnectionMaxLifetime = 10 * time.Millisecond
		config.CacheCleanupInterval = 20 * time.Millisecond
		
		securityConfig := &SecurityConfig{}
		auditor := &SecurityAuditor{}
		
		cp := NewConnectionPool(config, securityConfig, auditor)
		
		// Add a mock SSH connection
		mockConn := &SSHConnection{
			Client: nil,
		}
		
		// Add a connection that's old enough to be expired
		entry := &ConnectionPoolEntry{
			Connection: mockConn,
			CreatedAt:  time.Now().Add(-time.Hour), // Very old
			LastUsed:   time.Now(),                 // Recently used
			InUse:      false,
		}
		cp.connections["test-key"] = entry
		
		// Wait for cleanup to run (ticker interval)
		time.Sleep(50 * time.Millisecond)
		
		// Check that expired connection was removed
		if _, exists := cp.connections["test-key"]; exists {
			t.Error("Expected expired connection to be removed")
		}
	})
}