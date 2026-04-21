package distributed

import (
	"testing"
	"time"
)

func TestNewSSHConfig(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := NewSSHConfig("example.com", "testuser")

		if config.Host != "example.com" {
			t.Errorf("Expected host 'example.com', got '%s'", config.Host)
		}

		if config.Port != 22 {
			t.Errorf("Expected port 22, got %d", config.Port)
		}

		if config.User != "testuser" {
			t.Errorf("Expected user 'testuser', got '%s'", config.User)
		}

		if config.Timeout != 30*time.Second {
			t.Errorf("Expected timeout 30s, got %v", config.Timeout)
		}

		if config.MaxRetries != 3 {
			t.Errorf("Expected max retries 3, got %d", config.MaxRetries)
		}

		if config.RetryDelay != 5*time.Second {
			t.Errorf("Expected retry delay 5s, got %v", config.RetryDelay)
		}
	})
}

func TestNewWorkerConfig(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := NewWorkerConfig("worker1", "Test Worker", "example.com", "testuser")

		if config.ID != "worker1" {
			t.Errorf("Expected ID 'worker1', got '%s'", config.ID)
		}

		if config.Name != "Test Worker" {
			t.Errorf("Expected name 'Test Worker', got '%s'", config.Name)
		}

		if config.SSH.Host != "example.com" {
			t.Errorf("Expected SSH host 'example.com', got '%s'", config.SSH.Host)
		}

		if config.SSH.User != "testuser" {
			t.Errorf("Expected SSH user 'testuser', got '%s'", config.SSH.User)
		}

		if config.MaxCapacity != 5 {
			t.Errorf("Expected max capacity 5, got %d", config.MaxCapacity)
		}

		if !config.Enabled {
			t.Error("Expected worker to be enabled")
		}
	})
}

func TestSSHPool_BasicOperations(t *testing.T) {
	t.Run("AddAndRemoveWorker", func(t *testing.T) {
		pool := NewSSHPool()
		defer pool.Close()

		// Create worker config
		config := NewWorkerConfig("worker1", "Test Worker", "127.0.0.1", "testuser")

		// Add worker
		pool.AddWorker(config)

		// Try to get connection (should fail since we can't actually connect)
		_, err := pool.GetConnection("worker1")
		if err == nil {
			t.Error("Expected connection to fail with test config")
		}

		// Remove worker
		pool.RemoveWorker("worker1")

		// Try to get connection after removal
		_, err = pool.GetConnection("worker1")
		if err == nil {
			t.Error("Expected connection to fail after worker removal")
		}
	})

	t.Run("GetConnectionNonExistent", func(t *testing.T) {
		pool := NewSSHPool()
		defer pool.Close()

		// Try to get connection for non-existent worker
		_, err := pool.GetConnection("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent worker")
		}

		expectedError := "worker nonexistent not configured"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("DisabledWorker", func(t *testing.T) {
		pool := NewSSHPool()
		defer pool.Close()

		// Create disabled worker config
		config := NewWorkerConfig("worker1", "Test Worker", "127.0.0.1", "testuser")
		config.Enabled = false

		// Add disabled worker
		pool.AddWorker(config)

		// Try to get connection
		_, err := pool.GetConnection("worker1")
		if err == nil {
			t.Error("Expected connection to fail for disabled worker")
		}

		expectedError := "worker worker1 is disabled"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}
