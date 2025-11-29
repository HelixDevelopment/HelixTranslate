package distributed

import (
	"testing"
)

func TestPairingManager_queryServiceInfo(t *testing.T) {
	t.Run("queryServiceInfo_NoConnection", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Try to query service info for a worker that doesn't exist
		service, err := pairingManager.queryServiceInfo("non-existent-worker")
		
		if err == nil {
			t.Error("Expected error when querying non-existent worker")
		}
		
		if service != nil {
			t.Error("Expected nil service when querying non-existent worker")
		}
		
		if !contains(err.Error(), "worker non-existent-worker not configured") {
			t.Errorf("Expected 'worker not configured' error, got: %v", err)
		}
	})
	
	t.Run("queryServiceInfo_InvalidEndpoints", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Create a mock worker config
		config := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host: "invalid-host-that-does-not-exist",
				Port: 22,
			},
			Enabled: true,
		}
		
		// Add config to the pool (not just connection)
		sshPool.configs["test-worker"] = config
		
		// Add connection directly to the pool
		conn := &SSHConnection{
			Config: config,
			Client: nil, // No actual SSH client
		}
		sshPool.connections["test-worker"] = conn
		
		// Try to query service info - will fail at all endpoints
		service, err := pairingManager.queryServiceInfo("test-worker")
		
		// Should return an error after trying all endpoints
		if err == nil {
			t.Error("Expected error when all endpoints are unreachable")
		}
		
		if service != nil {
			t.Error("Expected nil service when all endpoints are unreachable")
		}
		
		// Set the client to nil manually to avoid panic in Close
		conn.Client = nil
	})
	
	t.Run("queryServiceInfo_ConnectionError", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Create a connection to a non-existent host
		config := &WorkerConfig{
			ID: "test-worker",
			SSH: SSHConfig{
				Host: "127.0.0.1", // Localhost but wrong port
				Port: 12345,       // Non-existent service port
			},
			Enabled: true,
		}
		
		// Add config to the pool
		sshPool.configs["test-worker"] = config
		
		conn := &SSHConnection{
			Config: config,
			Client: nil,
		}
		sshPool.connections["test-worker"] = conn
		
		// Try to query service info - will fail to connect
		service, err := pairingManager.queryServiceInfo("test-worker")
		
		if err == nil {
			t.Error("Expected error when connection fails")
		}
		
		if service != nil {
			t.Error("Expected nil service when connection fails")
		}
	})
}