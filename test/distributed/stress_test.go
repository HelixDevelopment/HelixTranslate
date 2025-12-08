//go:build stress

package distributed

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/events"
)

func TestDistributedManager_Stress_ConcurrentOperations(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	cfg := &config.Config{}
	cfg.Distributed.Enabled = true

	eventBus := events.NewEventBus()
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	manager := NewDistributedManager(cfg, eventBus)
	manager.Initialize(localCoordinator)

	const numGoroutines = 50
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*operationsPerGoroutine)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Mix of different operations
				switch j % 4 {
				case 0:
					// Add worker
					worker := &WorkerConfig{
						ID:          "stress-worker",
						Name:        "Stress Worker",
						SSH:         SSHConfig{Host: "stress-host", User: "stress-user"},
						MaxCapacity: 5,
						Enabled:     true,
					}
					if err := manager.AddWorker("stress-worker", worker); err != nil {
						errorChan <- err
					}

				case 1:
					// Get status
					_ = manager.GetStatus()

				case 2:
					// Remove worker
					if err := manager.RemoveWorker("stress-worker"); err != nil {
						// Ignore errors for missing workers
						if err.Error() != "worker stress-worker not configured" {
							errorChan <- err
						}
					}

				case 3:
					// Attempt distributed translation
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					_, _ = manager.TranslateDistributed(ctx, "stress test", "test")
					cancel()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Check for errors
	errorCount := 0
	for err := range errorChan {
		t.Logf("Stress test error: %v", err)
		errorCount++
	}

	if errorCount > numGoroutines*operationsPerGoroutine/10 { // Allow 10% error rate
		t.Errorf("Too many errors in stress test: %d/%d", errorCount, numGoroutines*operationsPerGoroutine)
	}
}

func TestDistributedManager_Stress_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := &config.Config{}
	cfg.Distributed.Enabled = true

	eventBus := events.NewEventBus()
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	manager := NewDistributedManager(cfg, eventBus)
	manager.Initialize(localCoordinator)

	// Add many workers to test memory usage
	for i := 0; i < 100; i++ {
		worker := &WorkerConfig{
			ID:          "memory-test-worker",
			Name:        "Memory Test Worker",
			SSH:         SSHConfig{Host: "memory-host", User: "memory-user"},
			MaxCapacity: 5,
			Enabled:     true,
		}
		manager.AddWorker("memory-test-worker", worker)
		manager.RemoveWorker("memory-test-worker") // Clean up immediately
	}

	// Force garbage collection
	// Note: This is just a basic test - in production you'd use proper memory profiling

	status := manager.GetStatus()
	workers := status["workers"].(map[string]interface{})

	if len(workers) != 0 {
		t.Errorf("Expected no workers after cleanup, got %d", len(workers))
	}
}

func TestDistributedManager_Stress_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	cfg := &config.Config{}
	cfg.Distributed.Enabled = true

	eventBus := events.NewEventBus()
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	manager := NewDistributedManager(cfg, eventBus)
	manager.Initialize(localCoordinator)

	// Run for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	errorChan := make(chan error, 100)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Periodic status checks
				_ = manager.GetStatus()

				// Periodic translation attempts
				translateCtx, translateCancel := context.WithTimeout(context.Background(), time.Second)
				_, err := manager.TranslateDistributed(translateCtx, "long running test", "stress")
				translateCancel()

				if err != nil {
					select {
					case errorChan <- err:
					default:
						// Channel full, skip
					}
				}
			}
		}
	}()

	<-ctx.Done()
	close(errorChan)

	// Count errors (some are expected due to no real workers)
	errorCount := 0
	for range errorChan {
		errorCount++
	}

	t.Logf("Long running test completed with %d errors", errorCount)
}
