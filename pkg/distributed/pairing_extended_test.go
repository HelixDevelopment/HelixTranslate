package distributed

import (
	"context"
	"testing"
	
	"digital.vasic.translator/pkg/events"
)

func TestPairingManager_DiscoverService(t *testing.T) {
	t.Run("DiscoverNonExistentService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Try to discover non-existent service
		ctx := context.Background()
		service, err := manager.DiscoverService(ctx, "non-existent-worker")
		if err == nil {
			t.Error("Expected error discovering non-existent service")
		}
		
		if service != nil {
			t.Error("Expected nil service when discovery fails")
		}
	})
}

func TestPairingManager_PairWithService(t *testing.T) {
	t.Run("PairWithNonExistentService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Try to pair with non-existent service
		err := manager.PairWithService("non-existent-service-id")
		if err == nil {
			t.Error("Expected error pairing with non-existent service")
		}
	})
}

func TestPairingManager_UnpairService(t *testing.T) {
	t.Run("UnpairNonExistentService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Try to unpair non-existent service (should not panic)
		manager.UnpairService("non-existent-service-id")
		
		// Should not panic, just no-op
	})
}

func TestPairingManager_performHealthChecks(t *testing.T) {
	t.Run("HealthCheckWithNoServices", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Perform health checks with no services (should not panic)
		manager.performHealthChecks()
		
		// Should not panic
	})
}

func TestPairingManager_checkServiceHealth(t *testing.T) {
	t.Run("CheckHealthOfNonExistentService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Check health of non-existent service (should not panic)
		service := &RemoteService{
			WorkerID: "test-worker",
			Name:     "Test Service",
			Host:     "nonexistent.example.com",
			Port:     8080,
			Protocol: "http",
			Status:   "unknown",
		}
		manager.checkServiceHealth("non-existent-service-id", service)
		
		// Should not panic
	})
}

func TestPairingManager_emitEvent(t *testing.T) {
	t.Run("EmitEvent", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Create a mock event
		event := events.Event{
			Type:    "pairing",
			Message: "test event",
			Data: map[string]interface{}{
				"service": "test-service",
			},
		}
		
		// Emit event (should not panic even with nil event bus)
		manager.emitEvent(event)
		
		// Should not panic
	})
}

func TestPairingManager_QueryServiceInfo(t *testing.T) {
	t.Run("QueryNonExistentService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Try to query non-existent service
		service, err := manager.queryServiceInfo("non-existent-worker")
		if err == nil {
			t.Error("Expected error querying non-existent service")
		}
		if service != nil {
			t.Error("Expected nil service for non-existent worker")
		}
	})
	
	t.Run("QueryExistingService", func(t *testing.T) {
		sshPool := NewSSHPool()
		manager := NewPairingManager(sshPool, nil)
		defer manager.Close()
		
		// Add a mock connection to the pool
		config := NewWorkerConfig("test-worker", "Test Worker", "127.0.0.1", "testuser")
		conn := &SSHConnection{Config: config}
		sshPool.connections["test-worker"] = conn
		
		// Try to query existing service (should fail due to no HTTP server)
		service, err := manager.queryServiceInfo("test-worker")
		if err == nil {
			t.Error("Expected error due to no HTTP server running")
		}
		if service != nil {
			t.Error("Expected nil service when HTTP server is not available")
		}
	})
}