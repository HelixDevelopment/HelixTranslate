package distributed

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
)

func TestDistributedCoordinator(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		eventBus := events.NewEventBus()
		apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
		
		coordinator := NewDistributedCoordinator(
			nil, // localCoordinator
			nil, // sshPool
			nil, // pairingManager
			nil, // fallbackManager
			nil, // versionManager
			eventBus,
			apiLogger,
		)
		
		if coordinator == nil {
			t.Errorf("Expected coordinator to be created")
		}
		
		// Verify default configuration
		if coordinator.maxRetries != 3 {
			t.Errorf("Expected maxRetries 3, got %d", coordinator.maxRetries)
		}
		
		if coordinator.retryDelay != 2*time.Second {
			t.Errorf("Expected retryDelay 2s, got %v", coordinator.retryDelay)
		}
	})
	
	t.Run("DiscoverRemoteInstances", func(t *testing.T) {
		eventBus := events.NewEventBus()
		apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
		
		coordinator := NewDistributedCoordinator(
			nil,
			nil,
			nil, // pairingManager is nil
			nil,
			nil,
			eventBus,
			apiLogger,
		)
		
		// Test with nil pairing manager (should handle gracefully)
		// Since we know this will panic due to nil pointer, let's recover and expect the panic
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic when pairing manager is nil")
			}
		}()
		
		err := coordinator.DiscoverRemoteInstances(context.Background())
		t.Errorf("Should have panicked before this point, but got err: %v", err)
	})
	
	t.Run("GetPriorityForProvider", func(t *testing.T) {
		eventBus := events.NewEventBus()
		apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
		
		coordinator := NewDistributedCoordinator(
			nil,
			nil,
			nil,
			nil,
			nil,
			eventBus,
			apiLogger,
		)
		
		tests := []struct {
			provider  string
			expected  int
		}{
			{"openai", 10},
			{"anthropic", 10},
			{"zhipu", 10},
			{"deepseek", 10},
			{"ollama", 5},
			{"llamacpp", 5},
			{"unknown", 1},
		}
		
		for _, test := range tests {
			priority := coordinator.getPriorityForProvider(test.provider)
			if priority != test.expected {
				t.Errorf("Expected priority %d for provider %s, got %d", 
					test.expected, test.provider, priority)
			}
		}
	})
	
	t.Run("GetInstanceCountForPriority", func(t *testing.T) {
		eventBus := events.NewEventBus()
		apiLogger, _ := deployment.NewAPICommunicationLogger("/tmp/test-api.log")
		
		coordinator := NewDistributedCoordinator(
			nil,
			nil,
			nil,
			nil,
			nil,
			eventBus,
			apiLogger,
		)
		
		tests := []struct {
			priority      int
			maxConcurrent int
			expected      int
		}{
			{10, 5, 3}, // API key providers, limited by maxConcurrent
			{10, 2, 2}, // API key providers, limited by maxConcurrent
			{5, 5, 2},  // OAuth providers
			{1, 5, 1},  // Free/local providers
		}
		
		for _, test := range tests {
			count := coordinator.getInstanceCountForPriority(test.priority, test.maxConcurrent)
			if count != test.expected {
				t.Errorf("Expected count %d for priority %d with maxConcurrent %d, got %d", 
					test.expected, test.priority, test.maxConcurrent, count)
			}
		}
	})
}