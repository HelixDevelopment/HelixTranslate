package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	
	t.Run("queryRemoteProviders_WithSuccessfulResponse", func(t *testing.T) {
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
		
		// Create a mock HTTP server that returns provider data
		providersResponse := map[string]interface{}{
			"providers": []interface{}{
				map[string]interface{}{
					"name": "openai",
					"type": "api",
				},
				map[string]interface{}{
					"name": "anthropic",
					"type": "api",
				},
			},
		}
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/providers" {
				t.Errorf("Expected path '/api/v1/providers', got: %s", r.URL.Path)
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(providersResponse)
		}))
		defer server.Close()
		
		// Parse server URL to get host and port
		serverURL := server.URL
		hostPort := serverURL[7:] // Remove "http://"
		parts := strings.Split(hostPort, ":")
		host := parts[0]
		port := 8080 // Default test port
		if len(parts) > 1 {
			if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
				// Successfully parsed port
			}
		}
		
		// Create a service with the mock server's host and port
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
		}
		
		// Try to query providers - should succeed
		providers, err := coordinator.queryRemoteProviders(context.Background(), service)
		
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		
		if providers == nil {
			t.Error("Expected non-nil providers")
		}
		
		// Check that both providers are returned
		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got: %d", len(providers))
		}
		
		// Check for specific providers
		if _, ok := providers["openai"]; !ok {
			t.Error("Expected 'openai' provider in result")
		}
		
		if _, ok := providers["anthropic"]; !ok {
			t.Error("Expected 'anthropic' provider in result")
		}
	})
}