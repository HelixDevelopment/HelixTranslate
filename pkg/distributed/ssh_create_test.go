package distributed

import (
	"testing"
)

func TestSSHPool_createConnection(t *testing.T) {
	t.Run("createConnection_NoAuth", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		// Create a worker with no authentication
		worker := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host: "example.com",
				Port: 22,
				User: "test-user",
				// No KeyFile or Password
			},
		}
		
		// Try to create connection - should fail with no auth
		conn, err := sshPool.createConnection(worker)
		
		if err == nil {
			t.Error("Expected error with no authentication methods")
		}
		
		if conn != nil {
			t.Error("Expected nil connection with no authentication")
		}
		
		if !contains(err.Error(), "no authentication method configured") {
			t.Errorf("Expected 'no authentication method configured' error, got: %v", err)
		}
	})
	
	t.Run("createConnection_InvalidKey", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		// Create a worker with invalid key
		worker := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host:    "example.com",
				Port:    22,
				User:    "test-user",
				KeyFile: "invalid-key-content",
			},
		}
		
		// Try to create connection - should fail with invalid key
		conn, err := sshPool.createConnection(worker)
		
		if err == nil {
			t.Error("Expected error with invalid key")
		}
		
		if conn != nil {
			t.Error("Expected nil connection with invalid key")
		}
		
		if !contains(err.Error(), "failed to parse private key") {
			t.Errorf("Expected 'failed to parse private key' error, got: %v", err)
		}
	})
	
	t.Run("createConnection_WithPassword", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		// Create a worker with password auth
		worker := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host:     "non-existent-host-for-testing",
				Port:     22,
				User:     "test-user",
				Password: "test-password",
			},
		}
		
		// Try to create connection - should fail at connection level
		conn, err := sshPool.createConnection(worker)
		
		if err == nil {
			t.Error("Expected error when connecting to non-existent host")
		}
		
		if conn != nil {
			t.Error("Expected nil connection when connection fails")
		}
		
		// Should fail at connection level, not key parsing
		if contains(err.Error(), "failed to parse private key") {
			t.Error("Should not fail at key parsing")
		}
		
		if contains(err.Error(), "no authentication method configured") {
			t.Error("Should not fail due to no auth methods")
		}
	})
	
	t.Run("createConnection_WithKnownHosts", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		// Create a worker with known hosts file
		worker := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host:          "non-existent-host-for-testing",
				Port:          22,
				User:          "test-user",
				Password:      "test-password",
				KnownHostsFile: "/non/existent/known_hosts",
			},
		}
		
		// Try to create connection - should fail at connection level
		conn, err := sshPool.createConnection(worker)
		
		if err == nil {
			t.Error("Expected error when connecting to non-existent host")
		}
		
		if conn != nil {
			t.Error("Expected nil connection when connection fails")
		}
		
		// Should fail at connection level, not known hosts
		if contains(err.Error(), "failed to read known hosts file") {
			// This might happen before connection attempt
		} else if contains(err.Error(), "no authentication method configured") {
			t.Error("Should not fail due to no auth methods")
		}
	})
}