package distributed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// parseTestServerURL parses httptest server URL into host and port
func parseTestServerURL(serverURL string) (host string, port int) {
	parts := strings.Split(strings.TrimPrefix(serverURL, "https://"), ":")
	host = parts[0]
	port = 443 // httptest uses 443 for HTTPS
	return
}

func TestVersionManager_CheckWorkerVersion(t *testing.T) {
	// Create a mock server that returns version info
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-b7e4c27", // Match the current local version
				BuildTime:       "2024-01-01T00:00:00Z",
				GitCommit:       "abc123",
				GoVersion:       "go1.21.0",
				Components: map[string]string{
					"translator":  "dev-b7e4c27",
					"api":         "1.0.0",
					"distributed": "1.0.0",
				},
				LastUpdated: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Parse server URL
	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	// Override the HTTP client and base URL for testing
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	// Create a mock service
	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	// Test version checking
	upToDate, err := vm.CheckWorkerVersion(context.Background(), service)
	if err != nil {
		t.Fatalf("CheckWorkerVersion failed: %v", err)
	}

	if !upToDate {
		t.Errorf("Expected worker to be up to date, but it was not")
	}

	// Verify version was updated
	if service.Version.CodebaseVersion != "dev-b7e4c27" {
		t.Errorf("Expected version dev-b7e4c27, got %s", service.Version.CodebaseVersion)
	}
}

func TestVersionManager_CheckWorkerVersion_Outdated(t *testing.T) {
	// Create a mock server that returns outdated version
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-old123", // Different from current local version
				BuildTime:       "2024-01-01T00:00:00Z",
				GitCommit:       "abc123",
				GoVersion:       "go1.21.0",
				Components: map[string]string{
					"translator": "dev-old123",
					"api":        "1.0.0",
				},
				LastUpdated: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	upToDate, err := vm.CheckWorkerVersion(context.Background(), service)
	if err != nil {
		t.Fatalf("CheckWorkerVersion failed: %v", err)
	}

	if upToDate {
		t.Errorf("Expected worker to be outdated, but it was up to date")
	}
}

func TestVersionManager_ValidateWorkerForWork(t *testing.T) {
	// Create a mock server for health checks
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-b7e4c27",
				BuildTime:       "2024-01-01T00:00:00Z",
				GitCommit:       "abc123",
				GoVersion:       "go1.21.0",
				Components: map[string]string{
					"translator":  "dev-b7e4c27",
					"api":         "1.0.0",
					"distributed": "1.0.0",
				},
				LastUpdated: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "dev-b7e4c27",
			Components: map[string]string{
				"translator":  "dev-b7e4c27",
				"api":         "1.0.0",
				"distributed": "1.0.0",
			},
		},
	}

	err := vm.ValidateWorkerForWork(context.Background(), service)
	if err != nil {
		t.Fatalf("ValidateWorkerForWork failed: %v", err)
	}
}

func TestVersionManager_ValidateWorkerForWork_Outdated(t *testing.T) {
	// Create a mock server that returns outdated version
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "0.9.0", // Outdated version
				BuildTime:       "2024-01-01T00:00:00Z",
				GitCommit:       "abc123",
				GoVersion:       "go1.21.0",
				Components: map[string]string{
					"translator":  "0.9.0",
					"api":         "0.9.0",
					"distributed": "0.9.0",
				},
				LastUpdated: time.Now(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "0.9.0", // Initially outdated
			Components: map[string]string{
				"translator": "0.9.0",
			},
		},
	}

	err := vm.ValidateWorkerForWork(context.Background(), service)
	if err == nil {
		t.Errorf("Expected validation to fail for outdated worker")
	}

	if !strings.Contains(err.Error(), "outdated") {
		t.Errorf("Expected error message to contain 'outdated', got: %s", err.Error())
	}
}

func TestVersionManager_ValidateWorkerForWork_Unhealthy(t *testing.T) {
	// Create a mock server that returns unhealthy status
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "1.0.0",
				Components: map[string]string{
					"translator": "1.0.0",
					"api":        "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "1.0.0",
			Components: map[string]string{
				"translator": "1.0.0",
				"api":        "1.0.0",
			},
		},
	}

	err := vm.ValidateWorkerForWork(context.Background(), service)
	if err == nil {
		t.Errorf("Expected validation to fail for unhealthy worker")
	}
}

func TestVersionManager_UpdateWorker(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "version-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock update package
	updateContent := "mock update content"
	updatePath := filepath.Join(tempDir, "update-1.1.0.tar.gz")
	if err := os.WriteFile(updatePath, []byte(updateContent), 0644); err != nil {
		t.Fatalf("Failed to create mock update file: %v", err)
	}

	// Create a mock server for update operations
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/update/upload") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"uploaded"}`))
		} else if r.Method == "POST" && strings.Contains(r.URL.Path, "/update/apply") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"applied"}`))
		} else if r.URL.Path == "/api/v1/version" {
			// Return updated version after "update" (matches local version)
			version := VersionInfo{
				CodebaseVersion: "dev-b7e4c27",
				Components: map[string]string{
					"translator":  "dev-b7e4c27",
					"api":         "1.0.0",
					"distributed": "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)
	vm.updateDir = tempDir

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "1.0.0",
		},
	}

	err = vm.UpdateWorker(context.Background(), service)
	if err != nil {
		t.Fatalf("UpdateWorker failed: %v", err)
	}

	// Verify version was updated
	if service.Version.CodebaseVersion != "dev-b7e4c27" {
		t.Errorf("Expected version dev-b7e4c27 after update, got %s", service.Version.CodebaseVersion)
	}
}

func TestVersionManager_compareVersions(t *testing.T) {
	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)

	tests := []struct {
		name     string
		local    VersionInfo
		remote   VersionInfo
		expected bool
	}{
		{
			name: "identical versions",
			local: VersionInfo{
				CodebaseVersion: "1.0.0",
				Components:      map[string]string{"translator": "1.0.0", "api": "1.0.0"},
			},
			remote: VersionInfo{
				CodebaseVersion: "1.0.0",
				Components:      map[string]string{"translator": "1.0.0", "api": "1.0.0"},
			},
			expected: true,
		},
		{
			name: "different codebase versions",
			local: VersionInfo{
				CodebaseVersion: "1.1.0",
				Components:      map[string]string{"translator": "1.0.0", "api": "1.0.0"},
			},
			remote: VersionInfo{
				CodebaseVersion: "1.0.0",
				Components:      map[string]string{"translator": "1.0.0", "api": "1.0.0"},
			},
			expected: false,
		},
		{
			name: "different component versions",
			local: VersionInfo{
				CodebaseVersion: "1.0.0",
				Components:      map[string]string{"translator": "1.1.0", "api": "1.0.0"},
			},
			remote: VersionInfo{
				CodebaseVersion: "1.0.0",
				Components:      map[string]string{"translator": "1.0.0", "api": "1.0.0"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vm.compareVersions(tt.local, tt.remote)
			if result != tt.expected {
				t.Errorf("compareVersions() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestVersionManager_CheckWorkerVersion_NetworkTimeout(t *testing.T) {
	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)

	// Create a service with invalid host to simulate network timeout
	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     "192.0.2.1", // TEST-NET-1 (RFC 5737) - should timeout
		Port:     12345,       // Likely closed port
		Protocol: "https",
	}

	// Set a very short timeout for testing
	vm.httpClient.Timeout = 1 * time.Millisecond

	_, err := vm.CheckWorkerVersion(context.Background(), service)
	if err == nil {
		t.Errorf("Expected network timeout error, but got none")
	}

	if !strings.Contains(err.Error(), "failed to query worker version") {
		t.Errorf("Expected timeout-related error, got: %s", err.Error())
	}
}

func TestVersionManager_CheckWorkerVersion_HTTPError(t *testing.T) {
	// Create a mock server that returns HTTP error
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	_, err := vm.CheckWorkerVersion(context.Background(), service)
	if err == nil {
		t.Errorf("Expected HTTP error, but got none")
	}

	expectedError := "worker version endpoint returned status 500"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %s", expectedError, err.Error())
	}
}

func TestVersionManager_CheckWorkerVersion_MalformedJSON(t *testing.T) {
	// Create a mock server that returns malformed JSON
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"codebase_version": "1.0.0", "invalid_json": `)) // Malformed JSON
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	_, err := vm.CheckWorkerVersion(context.Background(), service)
	if err == nil {
		t.Errorf("Expected JSON parsing error, but got none")
	}

	if !strings.Contains(err.Error(), "failed to decode worker version") {
		t.Errorf("Expected JSON decode error, got: %s", err.Error())
	}
}

func TestVersionManager_UpdateWorker_UploadFailure(t *testing.T) {
	// Create a mock server that fails upload
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/update/upload") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"upload failed"}`))
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-old123",
				Components: map[string]string{
					"translator": "dev-old123",
					"api":        "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)
	vm.updateDir = "/tmp/test-updates" // Use a test directory

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "dev-old123",
		},
	}

	err := vm.UpdateWorker(context.Background(), service)
	if err == nil {
		t.Errorf("Expected update to fail due to upload error")
	}

	if !strings.Contains(err.Error(), "failed to upload update package") {
		t.Errorf("Expected upload failure error, got: %s", err.Error())
	}

	// Verify service status was reset to "outdated"
	if service.Status != "outdated" {
		t.Errorf("Expected service status to be 'outdated', got '%s'", service.Status)
	}
}

func TestVersionManager_UpdateWorker_ApplyFailure(t *testing.T) {
	// Create a mock server that fails apply
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/update/upload") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"uploaded"}`))
		} else if strings.Contains(r.URL.Path, "/update/apply") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"apply failed"}`))
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-old123",
				Components: map[string]string{
					"translator": "dev-old123",
					"api":        "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)
	vm.updateDir = "/tmp/test-updates"

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "dev-old123",
		},
	}

	err := vm.UpdateWorker(context.Background(), service)
	if err == nil {
		t.Errorf("Expected update to fail due to apply error")
	}

	if !strings.Contains(err.Error(), "failed to trigger worker update") {
		t.Errorf("Expected apply failure error, got: %s", err.Error())
	}

	// Verify service status was reset to "outdated"
	if service.Status != "outdated" {
		t.Errorf("Expected service status to be 'outdated', got '%s'", service.Status)
	}
}

func TestVersionManager_UpdateWorker_VerificationFailure(t *testing.T) {
	// Create a mock server that succeeds but returns wrong version after "update"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/update/upload") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"uploaded"}`))
		} else if strings.Contains(r.URL.Path, "/update/apply") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"applied"}`))
		} else if r.URL.Path == "/api/v1/version" {
			// Always return the old version, simulating failed update
			version := VersionInfo{
				CodebaseVersion: "dev-old123",
				Components: map[string]string{
					"translator": "dev-old123",
					"api":        "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)
	vm.updateDir = "/tmp/test-updates"

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "dev-old123",
		},
	}

	err := vm.UpdateWorker(context.Background(), service)
	if err == nil {
		t.Errorf("Expected update to fail due to verification failure")
	}

	if !strings.Contains(err.Error(), "update verification failed") {
		t.Errorf("Expected verification failure error, got: %s", err.Error())
	}

	// Verify service status was reset to "outdated"
	if service.Status != "outdated" {
		t.Errorf("Expected service status to be 'outdated', got '%s'", service.Status)
	}
}

func TestVersionManager_CheckWorkerVersion_EmptyVersion(t *testing.T) {
	// Create a mock server that returns empty version
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "",
				Components:      map[string]string{},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	upToDate, err := vm.CheckWorkerVersion(context.Background(), service)
	if err != nil {
		t.Fatalf("CheckWorkerVersion failed: %v", err)
	}

	// Should be not up to date due to empty version
	if upToDate {
		t.Errorf("Expected worker to be not up to date due to empty version")
	}
}

func TestVersionManager_CheckWorkerVersion_MissingComponents(t *testing.T) {
	// Create a mock server that returns version missing critical components
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-b7e4c27",
				Components: map[string]string{
					"translator": "dev-b7e4c27",
					// Missing "api" and "distributed" components
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	upToDate, err := vm.CheckWorkerVersion(context.Background(), service)
	if err != nil {
		t.Fatalf("CheckWorkerVersion failed: %v", err)
	}

	// Should be not up to date due to missing components
	if upToDate {
		t.Errorf("Expected worker to be not up to date due to missing components")
	}
}

func TestVersionManager_GetLocalVersion(t *testing.T) {
	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)

	localVersion := vm.GetLocalVersion()

	// Verify basic structure
	if localVersion.CodebaseVersion == "" {
		t.Errorf("Expected non-empty codebase version")
	}

	if localVersion.Components == nil {
		t.Errorf("Expected non-nil components map")
	}

	// Check critical components exist
	criticalComponents := []string{"translator", "api", "distributed"}
	for _, component := range criticalComponents {
		if version, exists := localVersion.Components[component]; !exists || version == "" {
			t.Errorf("Expected component %s to exist and have non-empty version", component)
		}
	}
}

func TestVersionManager_ValidateWorkerForWork_HealthCheckFailure(t *testing.T) {
	// Create a mock server that fails health check
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-b7e4c27",
				Components: map[string]string{
					"translator":  "dev-b7e4c27",
					"api":         "1.0.0",
					"distributed": "1.0.0",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
	}

	err := vm.ValidateWorkerForWork(context.Background(), service)
	if err == nil {
		t.Errorf("Expected validation to fail due to health check failure")
	}

	if !strings.Contains(err.Error(), "worker health check failed") {
		t.Errorf("Expected health check failure error, got: %s", err.Error())
	}
}

func TestVersionManager_UpdateWorker_WithRollback(t *testing.T) {
	// Create a mock server that fails during update but allows rollback
	callCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			version := VersionInfo{
				CodebaseVersion: "dev-old123", // Initially outdated
				Components: map[string]string{
					"translator":  "dev-old123",
					"api":         "1.0.0",
					"distributed": "1.0.0",
				},
			}
			// After first call (during rollback check), return updated version
			if callCount > 0 {
				version.CodebaseVersion = "dev-b7e4c27"
				version.Components["translator"] = "dev-b7e4c27"
			}
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(version)
		} else if strings.Contains(r.URL.Path, "/update/upload") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"uploaded"}`))
		} else if strings.Contains(r.URL.Path, "/update/apply") {
			// Simulate failure during apply
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"apply failed"}`))
		} else if strings.Contains(r.URL.Path, "/update/rollback") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"rolled back"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host, port := parseTestServerURL(server.URL)

	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)
	vm.httpClient = server.Client()
	vm.SetBaseURL(server.URL)
	vm.updateDir = "/tmp/test-updates"

	service := &RemoteService{
		WorkerID: "test-worker",
		Host:     host,
		Port:     port,
		Protocol: "https",
		Version: VersionInfo{
			CodebaseVersion: "dev-old123",
		},
	}

	// Update should fail and trigger rollback
	err := vm.UpdateWorker(context.Background(), service)
	if err == nil {
		t.Errorf("Expected update to fail and rollback")
	}

	if !strings.Contains(err.Error(), "failed to trigger worker update") {
		t.Errorf("Expected apply failure error, got: %s", err.Error())
	}

	// Verify service was rolled back to original version
	if service.Version.CodebaseVersion != "dev-old123" {
		t.Errorf("Expected service to be rolled back to dev-old123, got %s", service.Version.CodebaseVersion)
	}

	// Verify service status is paired (after successful rollback)
	if service.Status != "paired" {
		t.Errorf("Expected service status to be 'paired' after rollback, got '%s'", service.Status)
	}
}

func TestVersionManager_RollbackWorkerUpdate_NoBackup(t *testing.T) {
	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)

	service := &RemoteService{
		WorkerID: "test-worker",
		Version: VersionInfo{
			CodebaseVersion: "dev-b7e4c27",
		},
	}

	err := vm.rollbackWorkerUpdate(context.Background(), service)
	if err == nil {
		t.Errorf("Expected rollback to fail with no backup")
	}

	expectedError := "no backup found for worker test-worker"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %s", expectedError, err.Error())
	}
}
