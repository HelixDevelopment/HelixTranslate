package distributed

import (
	"context"
	"testing"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/deployment"
)

func TestDistributedCoordinator_translateWithRemoteInstance(t *testing.T) {
	t.Run("translateWithRemoteInstance_ServiceNotFound", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		apiLogger, _ := deployment.NewAPICommunicationLogger("test.log")
	coordinator := NewDistributedCoordinator(
		nil, // localCoordinator
		sshPool,
		pairingManager,
		nil, // fallbackManager
		nil, // versionManager
		events.NewEventBus(),
		apiLogger,
	)
		
		// Create an instance with a worker that doesn't exist
		instance := &RemoteLLMInstance{
			WorkerID: "non-existent-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// Try to translate with non-existent service
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error for non-existent service")
		}
		
		if !contains(err.Error(), "service not found") {
			t.Errorf("Expected service not found error, got: %v", err)
		}
	})
	
	t.Run("translateWithRemoteInstance_InvalidJSON", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		apiLogger, _ := deployment.NewAPICommunicationLogger("test.log")
	coordinator := NewDistributedCoordinator(
		nil, // localCoordinator
		sshPool,
		pairingManager,
		nil, // fallbackManager
		nil, // versionManager
		events.NewEventBus(),
		apiLogger,
	)
		
		// Add a service to the pairing manager
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "https",
		}
		pairingManager.services["test-worker"] = service
		
		// Create an instance for this worker
		instance := &RemoteLLMInstance{
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// This will fail when trying to make the HTTP request, but will test the JSON marshaling
		// and request creation logic before it fails
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		// Should get an error (probably connection error or timeout)
		if err == nil {
			t.Error("Expected error when trying to connect to unreachable service")
		}
		
		// The error should not be about JSON marshaling or request creation
		if contains(err.Error(), "failed to marshal request") {
			t.Error("Should not fail during JSON marshaling")
		}
		
		if contains(err.Error(), "failed to create request") {
			t.Error("Should not fail during request creation")
		}
	})
	
	t.Run("translateWithRemoteInstance_RequestCreation", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		apiLogger, _ := deployment.NewAPICommunicationLogger("test.log")
	coordinator := NewDistributedCoordinator(
		nil, // localCoordinator
		sshPool,
		pairingManager,
		nil, // fallbackManager
		nil, // versionManager
		events.NewEventBus(),
		apiLogger,
	)
		
		// Add a service with an invalid port that will cause an error during URL creation
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     -1, // Invalid port
			Protocol: "https",
		}
		pairingManager.services["test-worker"] = service
		
		// Create an instance for this worker
		instance := &RemoteLLMInstance{
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// Try to translate - should fail during URL creation
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		// Should get an error
		if err == nil {
			t.Error("Expected error with invalid service configuration")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}