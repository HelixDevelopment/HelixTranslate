package distributed

import (
	"context"
	"testing"
)

func TestPairingManager_DiscoverService(t *testing.T) {
	t.Run("DiscoverService_NoConnection", func(t *testing.T) {
		sshPool := NewSSHPool()
		defer sshPool.Close()
		
		pairingManager := NewPairingManager(sshPool, nil)
		defer pairingManager.Close()
		
		// Try to discover a service for a worker that doesn't exist
		service, err := pairingManager.DiscoverService(context.Background(), "non-existent-worker")
		
		if err == nil {
			t.Error("Expected error when discovering non-existent worker")
		}
		
		if service != nil {
			t.Error("Expected nil service when discovering non-existent worker")
		}
		
		if !contains(err.Error(), "failed to get SSH connection") {
			t.Errorf("Expected SSH connection error, got: %v", err)
		}
	})
}