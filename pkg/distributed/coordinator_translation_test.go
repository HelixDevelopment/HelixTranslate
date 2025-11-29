package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	
	t.Run("translateWithRemoteInstance_Success", func(t *testing.T) {
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
		
		// Create a mock HTTP server to handle the translation request
		expectedResponse := map[string]interface{}{
			"translated_text": "Translated text",
			"provider":        "test-provider",
			"model":           "test-model",
		}
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/translate" {
				t.Errorf("Expected path '/api/v1/translate', got: %s", r.URL.Path)
			}
			
			if r.Method != "POST" {
				t.Errorf("Expected POST method, got: %s", r.Method)
			}
			
			// Verify request body
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
			}
			
			if request["text"] != "test text" {
				t.Errorf("Expected text 'test text', got: %v", request["text"])
			}
			
			if request["context_hint"] != "test context" {
				t.Errorf("Expected context_hint 'test context', got: %v", request["context_hint"])
			}
			
			// Send successful response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedResponse)
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
		
		// Add a service with the mock server's host and port
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
			Status:   "paired", // Set status to paired
		}
		pairingManager.services["test-worker"] = service
		
		// Create an instance for this worker
		instance := &RemoteLLMInstance{
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// Try to translate - should succeed
		translated, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		
		if translated != "Translated text" {
			t.Errorf("Expected 'Translated text', got: %s", translated)
		}
	})
	
	t.Run("translateWithRemoteInstance_InvalidResponse", func(t *testing.T) {
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
		
		// Create a mock HTTP server that returns invalid response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Response missing translated_text field
			w.Write([]byte(`{"status": "success"}`))
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
		
		// Add a service with the mock server's host and port
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
			Status:   "paired", // Set status to paired
		}
		pairingManager.services["test-worker"] = service
		
		// Create an instance for this worker
		instance := &RemoteLLMInstance{
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// Try to translate - should fail due to invalid response
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error with invalid response format")
		}
		
		if !contains(err.Error(), "invalid response format") {
			t.Errorf("Expected 'invalid response format' error, got: %v", err)
		}
	})
	
	t.Run("translateWithRemoteInstance_HTTPError", func(t *testing.T) {
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
		
		// Create a mock HTTP server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
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
		
		// Add a service with the mock server's host and port
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
			Status:   "paired", // Set status to paired
		}
		pairingManager.services["test-worker"] = service
		
		// Create an instance for this worker
		instance := &RemoteLLMInstance{
			WorkerID: "test-worker",
			Provider: "test-provider",
			Model:    "test-model",
		}
		
		// Try to translate - should fail due to HTTP error
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"test text",
			"test context",
		)
		
		if err == nil {
			t.Error("Expected error with HTTP error status")
		}
		
		if !contains(err.Error(), "translation failed with status 500") {
			t.Errorf("Expected 'translation failed with status 500' error, got: %v", err)
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