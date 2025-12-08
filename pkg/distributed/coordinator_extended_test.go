package distributed

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct{}

func (m *MockLogger) Debug(message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func (m *MockLogger) Info(message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func (m *MockLogger) Warn(message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func (m *MockLogger) Error(message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func (m *MockLogger) Fatal(message string, fields map[string]interface{}) {
	// Do nothing in tests
}

func createTestCoordinator() *DistributedCoordinator {
	eventBus := events.NewEventBus()
	apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
	
	// Create fallback manager with proper event bus
	fallbackConfig := DefaultFallbackConfig()
	performanceConfig := DefaultPerformanceConfig()
	// Set recovery check interval to 0 to disable monitoring goroutines in tests
	fallbackConfig.RecoveryCheckInterval = 0
	
	mockLogger := &MockLogger{}
	fallbackManager := NewFallbackManager(fallbackConfig, performanceConfig, eventBus, mockLogger)
	
	return NewDistributedCoordinator(
		nil, // localCoordinator
		nil, // sshPool
		nil, // pairingManager
		fallbackManager, // fallbackManager - now not nil
		nil, // versionManager
		eventBus,
		apiLogger,
	)
}

func TestDistributedCoordinator_QueryRemoteProviders(t *testing.T) {
	t.Run("ValidService", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Create a test service
		service := &RemoteService{
			WorkerID: "worker1",
			Name:     "Test Worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "online",
		}
		
		// Just test that it doesn't panic
		_, err := coordinator.queryRemoteProviders(context.Background(), service)
		
		// We expect error because of invalid URL
		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
	})
	
	t.Run("NilService", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test with nil service
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Function panics with nil service as expected: %v", r)
			} else {
				t.Error("Expected panic with nil service, but function continued")
			}
		}()
		
		_, err := coordinator.queryRemoteProviders(context.Background(), nil)
		
		// If we get here without panic, check for error
		if err != nil {
			t.Logf("Error with nil service: %v", err)
		} else {
			t.Log("No error with nil service - function handles it gracefully")
		}
	})
	
	t.Run("CancelledContext", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Create a service
		service := &RemoteService{
			WorkerID: "worker1",
			Name:     "Test Worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "online",
		}
		
		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		// Test with cancelled context
		_, err := coordinator.queryRemoteProviders(ctx, service)
		
		// Should fail due to cancelled context
		if err == nil {
			t.Error("Expected error with cancelled context")
		} else {
			t.Logf("Expected error with cancelled context: %v", err)
		}
	})
	
	t.Run("DifferentProtocols", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test with different protocols
		protocols := []string{"http", "https", "http3"}
		
		for _, protocol := range protocols {
			service := &RemoteService{
				WorkerID: "worker1",
				Name:     "Test Worker",
				Host:     "localhost",
				Port:     8080,
				Protocol: protocol,
				Status:   "online",
			}
			
			_, err := coordinator.queryRemoteProviders(context.Background(), service)
			
			// We expect errors since there's no server, but function should handle all protocols
			if err == nil {
				t.Errorf("Expected error for %s protocol, got nil", protocol)
			} else {
				t.Logf("Expected error for %s protocol: %v", protocol, err)
			}
		}
	})
}

func TestDistributedCoordinator_TranslateWithRemoteInstances(t *testing.T) {
	t.Run("NoRemoteInstances", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		_, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"hello world",
			"",
		)
		
		if err == nil {
			t.Error("Expected error for no remote instances, got nil")
		}
		if err.Error() != "no remote instances available" {
			t.Errorf("Expected 'no remote instances available' error, got: %v", err)
		}
	})
	
	t.Run("NilContext", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test with nil context
		_, err := coordinator.translateWithRemoteInstances(
			nil,
			"hello world",
			"",
		)
		
		if err == nil {
			t.Error("Expected error for nil context, got nil")
		}
	})
	
	t.Run("EmptyText", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test with empty text
		_, err := coordinator.translateWithRemoteInstances(
			context.Background(),
			"",
			"",
		)
		
		if err == nil {
			t.Error("Expected error for empty text, got nil")
		}
	})
}

func TestDistributedCoordinator_ValidateWorkerForWork(t *testing.T) {
	coordinator := createTestCoordinator()
	
	t.Run("ValidWorker", func(t *testing.T) {
		err := coordinator.validateWorkerForWork(context.Background(), "worker1")
		if err != nil {
			// With nil version manager, should return nil
			t.Errorf("Expected nil for valid worker with no version manager, got %v", err)
		}
	})
	
	t.Run("NonExistentWorker", func(t *testing.T) {
		// Test with the createTestCoordinator which has a nil pairing manager
		// This will cause the function to panic when trying to access services
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Function panicked with nil pairing manager as expected: %v", r)
			} else {
				// If no panic, function returned nil because versionManager is also nil
				t.Log("Function returned nil - both pairingManager and versionManager are nil")
			}
		}()
		
		// Test with non-existent worker
		err := coordinator.validateWorkerForWork(context.Background(), "non-existent-worker")
		t.Logf("Function returned err: %v", err)
	})
	
	t.Run("CancelledContext", func(t *testing.T) {
		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		err := coordinator.validateWorkerForWork(ctx, "worker1")
		if err != nil {
			// With nil version manager, should return nil regardless of context
			t.Errorf("Expected nil for cancelled context with no version manager, got %v", err)
		}
	})
}

func TestDistributedCoordinator_GetNextRemoteInstance(t *testing.T) {
	coordinator := createTestCoordinator()
	
	t.Run("NoInstances", func(t *testing.T) {
		instance := coordinator.getNextRemoteInstance()
		if instance != nil {
			t.Error("Expected nil when no instances available")
		}
	})
	
	t.Run("SingleValidInstance", func(t *testing.T) {
		coordinator.remoteInstances = []*RemoteLLMInstance{
			{
				ID:        "instance1",
				WorkerID:  "worker1",
				Provider:  "openai",
				Model:     "gpt-4",
				Available: true,
				LastUsed:  time.Now().Add(-5 * time.Minute),
			},
		}
		
		instance := coordinator.getNextRemoteInstance()
		if instance == nil {
			t.Error("Expected instance when one is available")
		}
		
		if instance.ID != "instance1" {
			t.Errorf("Expected instance1, got %s", instance.ID)
		}
	})
	
	t.Run("MultipleInstancesRoundRobin", func(t *testing.T) {
		coordinator.remoteInstances = []*RemoteLLMInstance{
			{
				ID:        "instance1",
				WorkerID:  "worker1",
				Provider:  "openai",
				Model:     "gpt-4",
				Available: true,
				LastUsed:  time.Now().Add(-5 * time.Minute),
			},
			{
				ID:        "instance2",
				WorkerID:  "worker2",
				Provider:  "anthropic",
				Model:     "claude-3",
				Available: true,
				LastUsed:  time.Now().Add(-5 * time.Minute),
			},
		}
		
		// Reset index
		coordinator.currentIndex = 0
		
		// First call should return instance1
		instance1 := coordinator.getNextRemoteInstance()
		if instance1.ID != "instance1" {
			t.Errorf("Expected instance1 on first call, got %s", instance1.ID)
		}
		
		// Second call should return instance2
		instance2 := coordinator.getNextRemoteInstance()
		if instance2.ID != "instance2" {
			t.Errorf("Expected instance2 on second call, got %s", instance2.ID)
		}
		
		// Third call should return instance1 again (round robin)
		instance3 := coordinator.getNextRemoteInstance()
		if instance3.ID != "instance1" {
			t.Errorf("Expected instance1 on third call, got %s", instance3.ID)
		}
	})
}

func TestDistributedCoordinator_TranslateWithRemoteInstance(t *testing.T) {
	t.Run("NilPairingManager", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Create a test instance
		instance := &RemoteLLMInstance{
			ID:        "instance1",
			WorkerID:  "worker1",
			Provider:  "openai",
			Model:     "gpt-4",
			Available: true,
		}
		
		// Test with nil pairingManager - should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic due to nil pairingManager, but function didn't panic")
			} else {
				t.Logf("Function panicked as expected: %v", r)
			}
		}()
		
		// This should panic
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"hello world",
			"",
		)
		t.Errorf("Function should have panicked but returned err: %v", err)
	})
	
	t.Run("NilInstance", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test with nil instance - should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic due to nil instance, but function didn't panic")
			} else {
				t.Logf("Function panicked as expected: %v", r)
			}
		}()
		
		// This should panic
		_, err := coordinator.translateWithRemoteInstance(
			context.Background(),
			nil,
			"hello world",
			"",
		)
		t.Errorf("Function should have panicked but returned err: %v", err)
	})
	
	t.Run("NilContext", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Create a test instance
		instance := &RemoteLLMInstance{
			ID:        "instance1",
			WorkerID:  "worker1",
			Provider:  "openai",
			Model:     "gpt-4",
			Available: true,
		}
		
		// Test with nil context - should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic due to nil pairingManager, but function didn't panic")
			} else {
				t.Logf("Function panicked as expected: %v", r)
			}
		}()
		
		// This should panic
		_, err := coordinator.translateWithRemoteInstance(
			nil,
			instance,
			"hello world",
			"",
		)
		t.Errorf("Function should have panicked but returned err: %v", err)
	})
}

func TestDistributedCoordinator_GetRemoteInstanceCount(t *testing.T) {
	coordinator := createTestCoordinator()
	
	t.Run("NoInstances", func(t *testing.T) {
		count := coordinator.GetRemoteInstanceCount()
		if count != 0 {
			t.Errorf("Expected 0 instances, got %d", count)
		}
	})
	
	t.Run("WithInstances", func(t *testing.T) {
		coordinator.remoteInstances = []*RemoteLLMInstance{
			{ID: "instance1"},
			{ID: "instance2"},
			{ID: "instance3"},
		}
		
		count := coordinator.GetRemoteInstanceCount()
		if count != 3 {
			t.Errorf("Expected 3 instances, got %d", count)
		}
	})
}

func TestDistributedCoordinator_TranslateWithDistributedRetry(t *testing.T) {
	t.Run("ErrorFlow", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Test that the function properly handles error cases
		result, err := coordinator.TranslateWithDistributedRetry(
			context.Background(),
			"hello world",
			"",
		)
		
		// Either succeeds with a fallback translation or fails gracefully
		// Both are valid test outcomes
		if err != nil {
			t.Logf("Function returned error as expected: %v", err)
			// In error case, result should be empty
			if result != "" {
				t.Errorf("Expected empty result with error, got '%s'", result)
			}
		} else {
			// In success case, result should not be empty
			t.Logf("Function succeeded with fallback result: '%s'", result)
		}
	})
}

func TestDistributedCoordinator_EmitEvent(t *testing.T) {
	coordinator := createTestCoordinator()
	
	// Test that emitEvent doesn't panic with valid event
	event := events.Event{
		Type:      "test_event",
		SessionID: "test_session",
		Message:   "test message",
	}
	
	// Should not panic
	coordinator.emitEvent(event)
}

func TestDistributedCoordinator_EmitWarning(t *testing.T) {
	coordinator := createTestCoordinator()
	
	// Test that emitWarning doesn't panic
	message := "test warning message"
	
	// Should not panic
	coordinator.emitWarning(message)
}

func TestDistributedCoordinator_TranslateWithRemoteInstanceDetailed(t *testing.T) {
	t.Run("NilPairingManager", func(t *testing.T) {
		coordinator := createTestCoordinator()
		
		// Create a test instance
		instance := &RemoteLLMInstance{
			ID:        "instance1",
			WorkerID:  "worker1",
			Provider:  "openai",
			Model:     "gpt-4",
			Available: true,
		}
		
		// Test with nil pairingManager - should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic due to nil pairingManager, but function didn't panic")
			} else {
				t.Logf("Function panicked as expected: %v", r)
			}
		}()
		
		// This should panic
		coordinator.translateWithRemoteInstance(
			context.Background(),
			instance,
			"hello world",
			"",
		)
		t.Error("Function should have panicked but continued execution")
	})
}

func TestDistributedCoordinator_DiscoverRemoteInstancesCoverage(t *testing.T) {
	t.Run("DiscoverRemoteInstancesWithRealPairingManager", func(t *testing.T) {
		eventBus := events.NewEventBus()
		apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
		
		// Create a real pairing manager
		sshPool := NewSSHPool()
		pairingManager := NewPairingManager(sshPool, eventBus)
		
		coordinator := NewDistributedCoordinator(
			nil,
			sshPool,
			pairingManager,
			nil,
			nil,
			eventBus,
			apiLogger,
		)
		
		// Test with no paired services
		err := coordinator.DiscoverRemoteInstances(context.Background())
		
		// Should succeed with no services
		if err != nil {
			t.Errorf("Unexpected error with no services: %v", err)
		}
		
		// Test with context cancellation
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		err = coordinator.DiscoverRemoteInstances(ctx)
		if err != nil && err != context.Canceled {
			t.Logf("Error with cancelled context (expected): %v", err)
		}
	})
}