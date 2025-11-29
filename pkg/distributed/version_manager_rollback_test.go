package distributed

import (
	"context"
	"testing"
	"time"
	"digital.vasic.translator/pkg/events"
)

func TestVersionManager_rollbackWorkerUpdate(t *testing.T) {
	t.Run("rollbackWorkerUpdate_NoBackup", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "https",
		}
		
		err := versionManager.rollbackWorkerUpdate(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error when no backup exists")
		}
		
		if !contains(err.Error(), "no backup found") {
			t.Errorf("Expected 'no backup found' error, got: %v", err)
		}
	})
	
	t.Run("rollbackWorkerUpdate_InactiveBackup", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     8443,
			Protocol: "https",
		}
		
		// Add an inactive backup
		backup := &UpdateBackup{
			BackupID: "test-backup",
			Status:   "inactive", // Not active
		}
		versionManager.backups["test-worker"] = backup
		
		err := versionManager.rollbackWorkerUpdate(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error when backup is not active")
		}
		
		if !contains(err.Error(), "is not active") {
			t.Errorf("Expected 'not active' error, got: %v", err)
		}
	})
	
	t.Run("rollbackWorkerUpdate_InvalidURL", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "example.com",
			Port:     -1, // Invalid port
			Protocol: "https",
		}
		
		// Add an active backup
		backup := &UpdateBackup{
			BackupID: "test-backup",
			Status:   "active",
		}
		versionManager.backups["test-worker"] = backup
		
		err := versionManager.rollbackWorkerUpdate(context.Background(), service)
		
		if err == nil {
			t.Error("Expected error with invalid URL")
		}
		
		// Should fail at URL creation or HTTP request
		if contains(err.Error(), "no backup found") {
			t.Error("Should not fail due to missing backup")
		}
		if contains(err.Error(), "not active") {
			t.Error("Should not fail due to inactive backup")
		}
	})
	
	t.Run("rollbackWorkerUpdate_NetworkFailure", func(t *testing.T) {
		eventBus := events.NewEventBus()
		versionManager := NewVersionManager(eventBus)
		
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "non-existent-host-for-testing",
			Port:     8443,
			Protocol: "https",
		}
		
		// Add an active backup
		backup := &UpdateBackup{
			BackupID: "test-backup",
			Status:   "active",
		}
		versionManager.backups["test-worker"] = backup
		
		// Use a timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		err := versionManager.rollbackWorkerUpdate(ctx, service)
		
		if err == nil {
			t.Error("Expected error with network failure")
		}
		
		// Should fail at network level
		if contains(err.Error(), "no backup found") {
			t.Error("Should not fail due to missing backup")
		}
		if contains(err.Error(), "not active") {
			t.Error("Should not fail due to inactive backup")
		}
	})
}