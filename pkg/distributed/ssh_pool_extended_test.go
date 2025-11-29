package distributed

import (
	"context"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestSSHConnection_Close(t *testing.T) {
	t.Run("CloseNilClient", func(t *testing.T) {
		conn := &SSHConnection{
			Client: nil,
		}
		
		err := conn.Close()
		if err != nil {
			t.Errorf("Expected no error closing nil client, got %v", err)
		}
	})
}

func TestSSHConnection_ExecuteCommand(t *testing.T) {
	t.Run("ExecuteCommandWithNilClient", func(t *testing.T) {
		conn := &SSHConnection{
			Client: nil,
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Expect error when trying to execute command with nil client
		_, err := conn.ExecuteCommand(ctx, "test command")
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "SSH client is not initialized" {
			t.Errorf("Expected 'SSH client is not initialized', got: %v", err)
		}
	})
}

func TestAppendToKnownHosts(t *testing.T) {
	t.Run("AppendToNewFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := tmpDir + "/known_hosts"
		hostname := "example.com"
		
		// Create a simple test key
		testKey := &MockPublicKey{}
		
		err := appendToKnownHosts(filename, hostname, testKey)
		if err != nil {
			t.Errorf("Expected no error appending to known hosts, got %v", err)
		}
		
		// Verify file was created and has content
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("Expected to read known hosts file, got %v", err)
		}
		
		expectedContent := hostname + " test-key-data\n"
		if string(content) != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
		}
	})
	
	t.Run("AppendToExistingFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := tmpDir + "/known_hosts"
		
		// Create initial file
		err := os.WriteFile(filename, []byte("existing-content\n"), 0600)
		if err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}
		
		hostname := "example.com"
		testKey := &MockPublicKey{}
		
		err = appendToKnownHosts(filename, hostname, testKey)
		if err != nil {
			t.Errorf("Expected no error appending to existing file, got %v", err)
		}
		
		// Verify file has original + new content
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("Expected to read known hosts file, got %v", err)
		}
		
		expectedContent := "existing-content\n" + hostname + " test-key-data\n"
		if string(content) != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
		}
	})
}

func TestSSHPool_GetWorkers(t *testing.T) {
	t.Run("EmptyPool", func(t *testing.T) {
		pool := NewSSHPool()
		defer pool.Close()
		
		workers := pool.GetWorkers()
		if len(workers) != 0 {
			t.Errorf("Expected no workers in empty pool, got %d", len(workers))
		}
	})
	
	t.Run("PoolWithWorkers", func(t *testing.T) {
		pool := NewSSHPool()
		defer pool.Close()
		
		// Add some workers
		config1 := NewWorkerConfig("worker1", "Worker 1", "127.0.0.1", "user1")
		config2 := NewWorkerConfig("worker2", "Worker 2", "127.0.0.2", "user2")
		
		pool.AddWorker(config1)
		pool.AddWorker(config2)
		
		workers := pool.GetWorkers()
		if len(workers) != 2 {
			t.Errorf("Expected 2 workers, got %d", len(workers))
		}
		
		if workers["worker1"] == nil {
			t.Error("Expected worker1 to be present")
		}
		
		if workers["worker2"] == nil {
			t.Error("Expected worker2 to be present")
		}
		
		// Verify returned configs are copies (can't test easily but ensure no panics)
		workers["worker3"] = NewWorkerConfig("worker3", "Worker 3", "127.0.0.3", "user3")
		
		// Verify original pool wasn't modified
		originalWorkers := pool.GetWorkers()
		if len(originalWorkers) != 2 {
			t.Error("Original pool should not be modified by modifying returned map")
		}
	})
}

// Mock implementations for testing

type MockPublicKey struct{}

func (m *MockPublicKey) Type() string {
	return "mock-key"
}

func (m *MockPublicKey) Marshal() []byte {
	return []byte("test-key-data")
}

func (m *MockPublicKey) Verify(data []byte, sig *ssh.Signature) error {
	return &MockSSHError{message: "mock verify not implemented"}
}

type MockSSHError struct {
	message string
}

func (e *MockSSHError) Error() string {
	return e.message
}