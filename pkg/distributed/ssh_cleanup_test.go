package distributed

import (
	"testing"
	"time"
)

func TestSSHPool_cleanup(t *testing.T) {
	t.Run("cleanup_IdleConnections", func(t *testing.T) {
		// D10: configure short cleanup timing AT CONSTRUCTION (mutating the fields
		// after NewSSHPool races the already-running cleanup goroutine).
		sshPool := NewSSHPool(WithCleanupTiming(10*time.Millisecond, 20*time.Millisecond))

		config := &WorkerConfig{
			ID:  "test-worker",
			SSH: SSHConfig{Host: "example.com", Port: 22},
		}
		conn := &SSHConnection{
			Config:   config,
			Client:   nil,                        // nil client is fine for this test
			LastUsed: time.Now().Add(-time.Hour), // Idle for a long time
		}

		// Add under the pool mutex — cleanup() reads the connections map under the
		// same lock, so an unlocked write here would race it (D10).
		sshPool.mu.Lock()
		sshPool.connections["test-worker"] = conn
		sshPool.mu.Unlock()

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
		// D10: longer idle time, short tick — set at construction (race-free).
		sshPool := NewSSHPool(WithCleanupTiming(50*time.Millisecond, 20*time.Millisecond))

		config := &WorkerConfig{
			ID:  "test-worker",
			SSH: SSHConfig{Host: "example.com", Port: 22},
		}
		conn := &SSHConnection{
			Config:   config,
			Client:   nil,
			LastUsed: time.Now(), // Just used now
		}

		sshPool.mu.Lock()
		sshPool.connections["test-worker"] = conn
		sshPool.mu.Unlock()

		// Wait for cleanup to run (ticker interval) but less than idle timeout
		time.Sleep(25 * time.Millisecond)

		// Check that active connection was NOT removed
		sshPool.mu.Lock()
		_, exists := sshPool.connections["test-worker"]
		elapsed := time.Since(conn.LastUsed)
		maxIdle := sshPool.maxIdleTime
		sshPool.mu.Unlock()
		if !exists {
			t.Errorf("Expected active connection to NOT be removed, but it was removed after %v (idle timeout: %v)", elapsed, maxIdle)
		}

		sshPool.Close()
	})
}
