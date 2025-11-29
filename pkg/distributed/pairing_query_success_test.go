package distributed

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

func TestPairingManager_queryServiceInfoSuccess(t *testing.T) {
	// Create mock server that handles both providers and health requests
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if strings.Contains(r.URL.Path, "/api/v1/providers") {
			// Return providers response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"providers": ["openai", "anthropic", "ollama"]
			}`))
		} else if strings.Contains(r.URL.Path, "/health") {
			// Return health response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "healthy",
				"uptime": 12345
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Parse the mock server URL to get host
	parts := strings.Split(mockServer.URL, ":")
	if len(parts) < 3 {
		t.Fatalf("Invalid mock server URL: %s", mockServer.URL)
	}
	host := parts[1][2:] // Remove "//"

	// Create SSH pool and pairing manager
	sshPool := NewSSHPool()
	defer sshPool.Close()
	
	pairingManager := NewPairingManager(sshPool, events.NewEventBus())
	defer pairingManager.Close()

	// Create worker config
	config := &WorkerConfig{
		ID: "test-worker",
		SSH: SSHConfig{
			Host: host,
			Port: 22,
		},
		MaxCapacity: 10,
		Name:        "Test Worker",
		Enabled:     true,
	}

	// Add config and connection to the pool
	sshPool.configs["test-worker"] = config
	conn := &SSHConnection{
		Config: config,
		Client: nil, // Not using actual SSH connection
	}
	sshPool.connections["test-worker"] = conn

	// Replace the HTTP client with one that redirects to our mock server
	originalClient := pairingManager.httpClient
	pairingManager.httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &customTransport{
			baseURL: mockServer.URL,
		},
	}
	defer func() { pairingManager.httpClient = originalClient }()

	// Test successful query
	service, err := pairingManager.queryServiceInfo("test-worker")
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if service == nil {
		t.Fatal("Expected service to be returned, but got nil")
	}
	
	// Verify service details
	if service.WorkerID != "test-worker" {
		t.Errorf("Expected WorkerID to be 'test-worker', got %s", service.WorkerID)
	}
	
	if service.Name != "Test Worker" {
		t.Errorf("Expected Name to be 'Test Worker', got %s", service.Name)
	}
	
	if service.Status != "online" {
		t.Errorf("Expected Status to be 'online', got %s", service.Status)
	}
	
	if service.Capabilities.MaxConcurrent != 10 {
		t.Errorf("Expected MaxConcurrent to be 10, got %d", service.Capabilities.MaxConcurrent)
	}
	
	if !service.Capabilities.SupportsBatch {
		t.Error("Expected SupportsBatch to be true")
	}
	
	// Check that providers were extracted
	expectedProviders := []string{"openai", "anthropic", "ollama"}
	if len(service.Capabilities.Providers) != len(expectedProviders) {
		t.Fatalf("Expected %d providers, got %d", len(expectedProviders), len(service.Capabilities.Providers))
	}
	
	for i, provider := range expectedProviders {
		if service.Capabilities.Providers[i] != provider {
			t.Errorf("Expected provider %s at index %d, got %s", provider, i, service.Capabilities.Providers[i])
		}
	}
}

// customTransport redirects all HTTP requests to our mock server
type customTransport struct {
	baseURL string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Create a new request to our mock server with the same path
	newURL := t.baseURL + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	
	// Copy headers
	for key, values := range req.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}
	
	// Use the default transport to make the request
	return http.DefaultTransport.RoundTrip(newReq)
}