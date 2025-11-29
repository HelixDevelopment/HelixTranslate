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
)

func TestVersionManager_CheckWorkerVersion_Comprehensive(t *testing.T) {
	t.Run("CheckWorkerVersion_WithValidResponse", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		// Create a test HTTP server
		expectedVersion := VersionInfo{
			CodebaseVersion: "v1.2.3",
			BuildTime:      "2023-01-01T00:00:00Z",
			GitCommit:      "abc123",
		}
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/version" {
				t.Errorf("Expected path '/api/v1/version', got: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedVersion)
		}))
		defer server.Close()
		
		// Parse server URL to get host and port
		serverURL := server.URL
		hostPort := strings.TrimPrefix(serverURL, "http://")
		parts := strings.Split(hostPort, ":")
		host := parts[0]
		port := 8080 // Default test port
		if len(parts) > 1 {
			// Extract the actual port from the test server
			if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
				// Successfully parsed port
			}
		}
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.2.3",
		}
		
		// Use a short timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Check worker version
		upToDate, err := versionManager.CheckWorkerVersion(ctx, service)
		
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		
		if !upToDate {
			t.Error("Expected up-to-date when versions match")
		}
		
		// Service should have been updated with version info
		if service.Version.CodebaseVersion != "v1.2.3" {
			t.Errorf("Expected service version v1.2.3, got: %s", service.Version.CodebaseVersion)
		}
	})
	
	t.Run("CheckWorkerVersion_WithMismatchedVersion", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		// Create a test HTTP server with different version
		expectedVersion := VersionInfo{
			CodebaseVersion: "v1.2.3",
			BuildTime:      "2023-01-01T00:00:00Z",
			GitCommit:      "abc123",
		}
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedVersion)
		}))
		defer server.Close()
		
		// Parse server URL to get host and port
		serverURL := server.URL
		hostPort := strings.TrimPrefix(serverURL, "http://")
		parts := strings.Split(hostPort, ":")
		host := parts[0]
		port := 8080 // Default test port
		if len(parts) > 1 {
			if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
				// Successfully parsed port
			}
		}
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
		}
		
		// Set local version to different value
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.1.0",
		}
		
		// Use a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Check worker version
		upToDate, err := versionManager.CheckWorkerVersion(ctx, service)
		
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		
		if upToDate {
			t.Error("Expected not up-to-date when versions don't match")
		}
	})
	
	t.Run("CheckWorkerVersion_WithInvalidJSON", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		// Create a test HTTP server with invalid JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{ invalid json }"))
		}))
		defer server.Close()
		
		// Parse server URL to get host and port
		serverURL := server.URL
		hostPort := strings.TrimPrefix(serverURL, "http://")
		parts := strings.Split(hostPort, ":")
		host := parts[0]
		port := 8080 // Default test port
		if len(parts) > 1 {
			if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
				// Successfully parsed port
			}
		}
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     host,
			Port:     port,
			Protocol: "http",
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.2.3",
		}
		
		// Use a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Check worker version - should fail to decode JSON
		upToDate, err := versionManager.CheckWorkerVersion(ctx, service)
		
		if err == nil {
			t.Error("Expected error with invalid JSON response")
		}
		
		if upToDate {
			t.Error("Expected false up-to-date with error")
		}
		
		if !contains(err.Error(), "failed to decode worker version") {
			t.Errorf("Expected 'failed to decode worker version' error, got: %v", err)
		}
	})
}