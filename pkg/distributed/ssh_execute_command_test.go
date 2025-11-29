package distributed

import (
	"context"
	"testing"
	"time"
)

func TestSSHConnection_ExecuteCommand(t *testing.T) {
	t.Run("ExecuteCommandWithNilClient", func(t *testing.T) {
		conn := &SSHConnection{
			Client: nil,
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Expect error when trying to execute command with nil client
		_, err := conn.ExecuteCommand(ctx, "test command")
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "SSH client is not initialized" {
			t.Errorf("Expected 'SSH client is not initialized', got: %v", err)
		}
	})
	
	t.Run("ExecuteCommandWithContextCancellationBeforeCall", func(t *testing.T) {
		conn := &SSHConnection{
			Client: nil, // We'll hit the nil client check before context cancellation
		}
		
		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		// Try to execute command - should fail due to nil client before context cancellation
		_, err := conn.ExecuteCommand(ctx, "test command")
		if err == nil {
			t.Error("Expected error due to nil client")
		}
		
		// Should fail with nil client error, not context error
		if err.Error() != "SSH client is not initialized" {
			t.Errorf("Expected 'SSH client is not initialized' error, got: %v", err)
		}
	})
	
	t.Run("ExecuteCommandWithNilClientUpdatesLastUsed", func(t *testing.T) {
		conn := &SSHConnection{
			Client:    nil,
			LastUsed:  time.Time{},
		}
		
		ctx := context.Background()
		
		// Record time before call
		beforeCall := time.Now()
		
		// Try to execute command with nil client
		_, err := conn.ExecuteCommand(ctx, "test command")
		
		// Should fail
		if err == nil {
			t.Error("Expected error for nil client")
		}
		
		// Verify LastUsed was updated
		if !conn.LastUsed.After(beforeCall) {
			t.Error("Expected LastUsed to be updated even with nil client")
		}
	})
}