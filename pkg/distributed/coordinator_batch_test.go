package distributed

import (
	"context"
	"testing"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/deployment"
)

func TestDistributedCoordinator_translateWithRemoteInstances(t *testing.T) {
	t.Run("translateWithRemoteInstances_NoInstances", func(t *testing.T) {
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
		
		// No remote instances added
		result, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error when no remote instances available")
		}
		
		if result != "" {
			t.Error("Expected empty result when no instances available")
		}
		
		if !contains(err.Error(), "no remote instances available") {
			t.Errorf("Expected 'no remote instances available' error, got: %v", err)
		}
	})
	
	t.Run("translateWithRemoteInstances_AllInstancesFail", func(t *testing.T) {
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
		
		// Add remote instances with invalid worker IDs (will fail)
		instance1 := &RemoteLLMInstance{
			ID:       "instance1",
			WorkerID: "invalid-worker-1",
			Provider: "test-provider",
			Model:    "test-model",
		}
		instance2 := &RemoteLLMInstance{
			ID:       "instance2",
			WorkerID: "invalid-worker-2",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		coordinator.remoteInstances = []*RemoteLLMInstance{instance1, instance2}
		
		result, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error when all instances fail")
		}
		
		if result != "" {
			t.Error("Expected empty result when all instances fail")
		}
		
		if !contains(err.Error(), "all distributed translation attempts failed") {
			t.Errorf("Expected 'all distributed translation attempts failed' error, got: %v", err)
		}
	})
	
	t.Run("translateWithRemoteInstances_VersionManagerValidation", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Create a version manager that will return errors for all validations
		versionManager := NewVersionManager(events.NewEventBus())
		
		apiLogger, _ := deployment.NewAPICommunicationLogger("test.log")
		coordinator := NewDistributedCoordinator(
			nil, // localCoordinator
			sshPool,
			pairingManager,
			nil, // fallbackManager
			versionManager,
			events.NewEventBus(),
			apiLogger,
		)
		
		// Add a remote instance with a worker that will fail validation
		instance := &RemoteLLMInstance{
			ID:       "instance1",
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		coordinator.remoteInstances = []*RemoteLLMInstance{instance}
		
		result, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error when version validation fails")
		}
		
		if result != "" {
			t.Error("Expected empty result when version validation fails")
		}
	})
	
	t.Run("translateWithRemoteInstances_WithValidInstance", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Add a service to the pairing manager to make the instance seem valid
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "https",
		}
		pairingManager.services["test-worker"] = service
		
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
		
		// Add a remote instance that will fail at network request (but pass initial validation)
		instance := &RemoteLLMInstance{
			ID:       "instance1",
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		coordinator.remoteInstances = []*RemoteLLMInstance{instance}
		
		result, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"test text",
			"test context",
		)
		
		// Should fail at network level, not at validation level
		if err == nil {
			t.Error("Expected error when trying to connect to unreachable service")
		}
		
		if result != "" {
			t.Error("Expected empty result when network fails")
		}
		
		// The error should be about connection, not about instances
		if contains(err.Error(), "no remote instances available") {
			t.Error("Should not fail due to no instances")
		}
		
		// Check that LastUsed was not updated since the request failed
		if !instance.LastUsed.IsZero() {
			t.Error("LastUsed should not be updated for failed requests")
		}
	})
}