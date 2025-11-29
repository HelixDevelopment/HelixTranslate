package distributed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/deployment"
)

func TestDistributedCoordinator_DiscoverRemoteInstancesSuccess(t *testing.T) {
	// Create mock HTTP server for provider API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate successful provider response with "providers" key
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"providers": {
				"gpt-4": {
					"models": ["gpt-4-turbo", "gpt-4-vision"],
					"type": "chat"
				},
				"claude-3": {
					"models": ["claude-3-opus", "claude-3-sonnet"],
					"type": "chat"
				}
			}
		}`))
	}))
	defer mockServer.Close()

	// Parse the mock server URL to get host and port
	var host string
	var port int
	
	parts := strings.Split(mockServer.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("Invalid mock server URL: %s", mockServer.URL)
	}
	
	// Extract host and port from URL like "http://127.0.0.1:12345"
	host = strings.Join(parts[1:len(parts)-1], ":")
	if strings.HasPrefix(host, "//") {
		host = host[2:] // Remove "//"
	}
	
	portNum, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		t.Fatalf("Failed to parse port from URL: %v", err)
	}
	port = portNum

	// Create event bus and logger
	eventBus := events.NewEventBus()
	apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
	
	// Create pairing manager with a paired service
	sshPool := NewSSHPool()
	pairingManager := NewPairingManager(sshPool, eventBus)
	
	// Add a paired service manually
	service := &RemoteService{
		WorkerID:     "test-worker-1",
		Name:         "Test Worker",
		Host:         host,
		Port:         port,
		Protocol:     "http",
		Status:       "paired",
		Capabilities: ServiceCapabilities{
			MaxConcurrent: 5,
		},
		Version: VersionInfo{
			CodebaseVersion: "1.0.0",
			BuildTime:       time.Now().Format(time.RFC3339),
			GitCommit:       "abc123",
			GoVersion:       "1.19",
			Components:      make(map[string]string),
			LastUpdated:     time.Now(),
		},
		LastSeen:     time.Now(),
	}
	
	// Use reflection or expose a method to add the service
	pairingManager.services = make(map[string]*RemoteService)
	pairingManager.services[service.WorkerID] = service
	
	// Create coordinator
	coordinator := NewDistributedCoordinator(
		nil,
		sshPool,
		pairingManager,
		nil,
		nil,
		eventBus,
		apiLogger,
	)
	
	// Test successful discovery
	ctx := context.Background()
	err = coordinator.DiscoverRemoteInstances(ctx)
	
	if err != nil {
		t.Fatalf("Unexpected error during discovery: %v", err)
	}
	
	// Verify remote instances were created
	coordinator.mu.Lock()
	instances := coordinator.remoteInstances
	coordinator.mu.Unlock()
	
	// Should have instances created for each provider/model combination
	if len(instances) == 0 {
		t.Error("Expected remote instances to be created, but got none")
	}
	
	// Check first instance
	if len(instances) > 0 {
		instance := instances[0]
		if instance.WorkerID != "test-worker-1" {
			t.Errorf("Expected WorkerID to be 'test-worker-1', got %s", instance.WorkerID)
		}
		if instance.Provider == "" {
			t.Error("Expected Provider to be set, but got empty string")
		}
		if instance.Model == "" {
			t.Error("Expected Model to be set, but got empty string")
		}
		if !instance.Available {
			t.Error("Expected Available to be true, but got false")
		}
	}
}