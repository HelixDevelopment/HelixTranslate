package distributed

import (
	"context"
	"testing"
	"time"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/deployment"
)

func TestDistributedCoordinator_queryRemoteProviders(t *testing.T) {
	t.Run("queryRemoteProviders_InvalidURL", func(t *testing.T) {
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
		
		// Create a service with invalid port
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     -1, // Invalid port
			Protocol: "https",
		}
		
		// Try to query providers - should fail at URL creation
		providers, err := coordinator.queryRemoteProviders(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error with invalid URL")
		}
		
		if providers != nil {
			t.Error("Expected nil providers with invalid URL")
		}
	})
	
	t.Run("queryRemoteProviders_NetworkFailure", func(t *testing.T) {
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
		
		// Create a service with non-existent host
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "non-existent-host-for-testing",
			Port:     8443,
			Protocol: "https",
		}
		
		// Use a timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Try to query providers - should fail at network level
		providers, err := coordinator.queryRemoteProviders(ctx, service)
		
		if err == nil {
			t.Error("Expected error with network failure")
		}
		
		if providers != nil {
			t.Error("Expected nil providers with network failure")
		}
	})
	
	t.Run("queryRemoteProviders_InvalidProtocol", func(t *testing.T) {
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
		
		// Create a service with invalid protocol
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "invalid-protocol",
		}
		
		// Try to query providers - should fail at HTTP request
		providers, err := coordinator.queryRemoteProviders(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error with invalid protocol")
		}
		
		if providers != nil {
			t.Error("Expected nil providers with invalid protocol")
		}
	})
	
	t.Run("queryRemoteProviders_WithValidConfig", func(t *testing.T) {
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
		
		// Create a service with valid config but unreachable host
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "127.0.0.1", // localhost but non-existent service
			Port:     12345,        // Non-existent service port
			Protocol: "https",
		}
		
		// Use a short timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Try to query providers - should fail at connection level
		providers, err := coordinator.queryRemoteProviders(ctx, service)
		
		if err == nil {
			t.Error("Expected error when connecting to unreachable service")
		}
		
		if providers != nil {
			t.Error("Expected nil providers when connection fails")
		}
	})
}