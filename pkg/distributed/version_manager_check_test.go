package distributed

import (
	"context"
	"testing"
	"time"
	"digital.vasic.translator/pkg/events"
)

func TestVersionManager_CheckWorkerVersion(t *testing.T) {
	t.Run("CheckWorkerVersion_CacheHit", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "https",
		}
		
		// Set up cached version
		cachedVersion := &VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		versionManager.versionCache["test-worker"] = &VersionCacheEntry{
			VersionInfo: *cachedVersion,
			Timestamp:   time.Now(),
			TTL:         time.Minute,
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		
		// Check worker version - should use cache
		upToDate, err := versionManager.CheckWorkerVersion(context.Background(), service)
		
		if err != nil {
			t.Errorf("Unexpected error with cached version: %v", err)
		}
		
		if !upToDate {
			t.Error("Expected up-to-date when versions match")
		}
		
		// Service should have been updated with cached version
		if service.Version.CodebaseVersion != "v1.0.0" {
			t.Errorf("Expected service version v1.0.0, got: %s", service.Version.CodebaseVersion)
		}
	})
	
	t.Run("CheckWorkerVersion_CacheExpired", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "non-existent-host-for-testing",
			Port:     8443,
			Protocol: "https",
		}
		
		// Set up expired cached version
		cachedVersion := &VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		versionManager.versionCache["test-worker"] = &VersionCacheEntry{
			VersionInfo: *cachedVersion,
			Timestamp:   time.Now().Add(-2 * time.Minute), // Expired
			TTL:         time.Minute,
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		
		// Use a timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Check worker version - should try to refresh cache
		_, err := versionManager.CheckWorkerVersion(ctx, service)
		
		// Should fail at network level
		if err == nil {
			t.Error("Expected error when trying to refresh expired cache with unreachable host")
		}
		
		if contains(err.Error(), "failed to query worker version") {
			// This is expected
		} else if contains(err.Error(), "worker version endpoint returned status") {
			// Also acceptable if it gets a response but not 200
		} else {
			t.Errorf("Unexpected error: %v", err)
		}
	})
	
	t.Run("CheckWorkerVersion_InvalidURL", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     -1, // Invalid port
			Protocol: "https",
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		
		// Check worker version - should fail at URL creation
		_, err := versionManager.CheckWorkerVersion(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error with invalid URL")
		}
		
		// Should fail at URL creation, not network
		if contains(err.Error(), "failed to query worker version") {
			t.Error("Should fail at URL creation, not network request")
		}
	})
	
	t.Run("CheckWorkerVersion_NetworkFailure", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "non-existent-host-for-testing",
			Port:     8443,
			Protocol: "https",
		}
		
		// Set local version
		versionManager.localVersion = VersionInfo{
			CodebaseVersion: "v1.0.0",
		}
		
		// Use a timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Check worker version - should fail at network
		_, err := versionManager.CheckWorkerVersion(ctx, service)
		
		if err == nil {
			t.Error("Expected error with network failure")
		}
		
		if contains(err.Error(), "failed to query worker version") {
			// Expected
		} else if contains(err.Error(), "worker version endpoint returned status") {
			// Also acceptable
		} else {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}