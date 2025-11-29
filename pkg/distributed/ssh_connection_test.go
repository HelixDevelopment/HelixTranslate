package distributed

import (
	"strings"
	"testing"
	"time"
)

func TestSSHPool_ConnectionMethods(t *testing.T) {
	t.Run("SSHConfig_KeyAndPassword", func(t *testing.T) {
		// Test SSH configuration with both key and password
		sshConfig := NewSSHConfig("test.example.com", "testuser")
		sshConfig.KeyFile = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA1234567890\n-----END RSA PRIVATE KEY-----"
		sshConfig.Password = "testpassword"
		
		// Create worker config
		config := NewWorkerConfig("worker1", "Test Worker", "test.example.com", "testuser")
		config.SSH = *sshConfig
		
		// Create pool to test createConnection
		pool := NewSSHPool()
		defer pool.Close()
		
		// Test createConnection - will fail when trying to connect but tests config parsing
		_, err := pool.createConnection(config)
		if err == nil {
			t.Error("Expected error when trying to connect with invalid key")
		}
		
		// Verify error is about connection, not parsing
		if err.Error() == "no authentication method configured" {
			t.Error("Should not get authentication error when both key and password are provided")
		}
	})
	
	t.Run("SSHConfig_NoAuthentication", func(t *testing.T) {
		// Test SSH configuration with no authentication
		config := NewWorkerConfig("worker1", "Test Worker", "test.example.com", "testuser")
		config.SSH.KeyFile = ""  // No key
		config.SSH.Password = ""  // No password
		
		// Create pool to test createConnection
		pool := NewSSHPool()
		defer pool.Close()
		
		// Test createConnection - should fail early due to no auth method
		_, err := pool.createConnection(config)
		if err == nil {
			t.Error("Expected error when no authentication method is configured")
		}
		
		if err.Error() != "no authentication method configured" {
			t.Errorf("Expected 'no authentication method configured' error, got: %v", err)
		}
	})
	
	t.Run("SSHConfig_InvalidKey", func(t *testing.T) {
		// Test SSH configuration with invalid key format
		config := NewWorkerConfig("worker1", "Test Worker", "test.example.com", "testuser")
		config.SSH.KeyFile = "invalid-key-data"
		config.SSH.Password = ""  // No password
		
		// Create pool to test createConnection
		pool := NewSSHPool()
		defer pool.Close()
		
		// Test createConnection - should fail due to invalid key
		_, err := pool.createConnection(config)
		if err == nil {
			t.Error("Expected error when key format is invalid")
		}
		
		// Should fail due to invalid key parsing, which means no auth methods
		// after parsing fails
		if !strings.Contains(err.Error(), "failed to parse private key") {
			t.Errorf("Expected key parsing error, got: %v", err)
		}
	})
	
	t.Run("SSHConfig_ConnectionFailure", func(t *testing.T) {
		// Test SSH configuration with valid auth but unreachable host
		config := NewWorkerConfig("worker1", "Test Worker", "invalid-host-12345", "testuser")
		config.SSH.Password = "testpassword"  // Use password auth
		config.SSH.Timeout = 10 * time.Millisecond  // Very short timeout
		config.SSH.MaxRetries = 0  // No retries
		
		// Create pool to test createConnection
		pool := NewSSHPool()
		defer pool.Close()
		
		// Test createConnection - should fail due to connection error
		_, err := pool.createConnection(config)
		if err == nil {
			t.Error("Expected error when host is unreachable")
		}
		
		// Should get a connection error, not an auth error
		if err.Error() == "no authentication method configured" {
			t.Error("Should not get authentication error when password is provided")
		}
	})
	
	t.Run("SSHConfig_VerifyConnectionHandling", func(t *testing.T) {
		// Test that createConnection properly handles connection creation
		config := NewWorkerConfig("worker1", "Test Worker", "127.0.0.1", "testuser")
		config.SSH.Password = "testpassword"
		config.SSH.Timeout = 1 * time.Millisecond  // Very short timeout
		config.SSH.MaxRetries = 0  // No retries
		
		// Create pool to test createConnection
		pool := NewSSHPool()
		defer pool.Close()
		
		// Test createConnection - should fail due to timeout
		_, err := pool.createConnection(config)
		if err == nil {
			t.Error("Expected error when connection times out")
		}
		
		// Should fail with connection error
		if err.Error() == "no authentication method configured" {
			t.Error("Should not get authentication error when password is provided")
		}
	})
}