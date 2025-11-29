package distributed

import (
	"testing"
	"time"
)

func TestSSHPool_cleanup(t *testing.T) {
	t.Run("cleanup_IdleConnections", func(t *testing.T) {
		// Create SSH pool normally with proper initialization
		sshPool := NewSSHPool()
		
		// Override settings for testing
		sshPool.maxIdleTime = 10 * time.Millisecond
		sshPool.cleanupTick = 20 * time.Millisecond
		
		// Add a mock connection
		config := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host: "example.com",
				Port: 22,
			},
		}
		
		conn := &SSHConnection{
			Config:   config,
			Client:   nil, // nil client is fine for this test
			LastUsed: time.Now().Add(-time.Hour), // Idle for a long time
		}
		
		sshPool.connections["test-worker"] = conn
		
		// Wait for cleanup to run (ticker interval)
		time.Sleep(25 * time.Millisecond)
		
		// Check that idle connection was removed
		sshPool.mu.Lock()
		_, exists := sshPool.connections["test-worker"]
		sshPool.mu.Unlock()
		
		if exists {
			t.Error("Expected idle connection to be removed")
		}
		
		sshPool.Close()
	})
	
	t.Run("cleanup_ActiveConnections", func(t *testing.T) {
		// Create SSH pool normally
		sshPool := NewSSHPool()
		
		// Override settings for testing - use longer idle time
		sshPool.maxIdleTime = 50 * time.Millisecond
		sshPool.cleanupTick = 20 * time.Millisecond
		
		// Add a mock connection
		config := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host: "example.com",
				Port: 22,
			},
		}
		
		conn := &SSHConnection{
			Config:   config,
			Client:   nil,
			LastUsed: time.Now(), // Just used now
		}
		
		sshPool.connections["test-worker"] = conn
		
		// Wait for cleanup to run (ticker interval) but less than idle timeout
		time.Sleep(25 * time.Millisecond)
		
		// Check that active connection was NOT removed
		sshPool.mu.Lock()
		_, exists := sshPool.connections["test-worker"]
		if !exists {
			// Let's add some debug info
			elapsed := time.Since(conn.LastUsed)
			t.Errorf("Expected active connection to NOT be removed, but it was removed after %v (idle timeout: %v)", elapsed, sshPool.maxIdleTime)
		}
		sshPool.mu.Unlock()
		
		sshPool.Close()
	})
}