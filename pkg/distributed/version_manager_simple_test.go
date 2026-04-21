package distributed

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

func TestVersionManager_GetLocalVersion(t *testing.T) {
	t.Run("GetLocalVersion_Default", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Get local version
		version := vm.GetLocalVersion()

		// Should return a valid version info
		if version.CodebaseVersion == "" {
			t.Error("Expected non-empty CodebaseVersion")
		}
		if version.GoVersion == "" {
			t.Error("Expected non-empty GoVersion")
		}
		if version.BuildTime == "" {
			t.Error("Expected non-empty BuildTime")
		}
	})
}

func TestVersionManager_SetBaseURL(t *testing.T) {
	t.Run("SetBaseURL", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Set base URL
		baseURL := "http://example.com/api"
		vm.SetBaseURL(baseURL)

		// Check that base URL was set
		// Note: We can't directly access baseURL field since it's private
		// This test mainly ensures the function doesn't panic
		// The field will be used when interacting with workers
	})
}

func TestBatchUpdateResult_GetSuccessRate(t *testing.T) {
	t.Run("GetSuccessRate_ZeroWorkers", func(t *testing.T) {
		result := &BatchUpdateResult{
			TotalWorkers: 0,
		}

		// Should return 100% for zero workers
		rate := result.GetSuccessRate()
		if rate != 100.0 {
			t.Errorf("Expected 100%% success rate for zero workers, got %.1f%%", rate)
		}
	})

	t.Run("GetSuccessRate_MixedResults", func(t *testing.T) {
		result := &BatchUpdateResult{
			TotalWorkers: 10,
			Successful:   make([]string, 0),
		}

		// Add 5 successful workers
		for i := 0; i < 5; i++ {
			result.Successful = append(result.Successful, "worker-id")
		}

		// Should return 50% success rate
		rate := result.GetSuccessRate()
		if rate != 50.0 {
			t.Errorf("Expected 50%% success rate, got %.1f%%", rate)
		}
	})
}

func TestBatchUpdateResult_GetSummary(t *testing.T) {
	t.Run("GetSummary_MixedResults", func(t *testing.T) {
		result := &BatchUpdateResult{
			TotalWorkers: 10,
			Successful:   []string{"worker1", "worker2"},
			Failed: []BatchUpdateError{{
				WorkerID: "worker3",
				Error:    "timeout",
			}, {
				WorkerID: "worker4",
				Error:    "connection refused",
			}, {
				WorkerID: "worker5",
				Error:    "version mismatch",
			}},
			Skipped: []string{"worker6"},
		}

		summary := result.GetSummary()

		// Should contain all the information
		expected := "Batch update completed: 2/10 successful (20.0%), 3 failed, 1 skipped in 0s"
		if summary != expected {
			t.Errorf("Expected summary '%s', got '%s'", expected, summary)
		}
	})

	t.Run("GetSummary_AllSuccessful", func(t *testing.T) {
		result := &BatchUpdateResult{
			TotalWorkers: 5,
			Successful:   []string{"w1", "w2", "w3", "w4", "w5"},
			Failed:       []BatchUpdateError{},
			Skipped:      []string{},
		}

		summary := result.GetSummary()

		// Should show 100% success
		expected := "Batch update completed: 5/5 successful (100.0%), 0 failed, 0 skipped in 0s"
		if summary != expected {
			t.Errorf("Expected summary '%s', got '%s'", expected, summary)
		}
	})
}

func TestVersionManager_ClearCache(t *testing.T) {
	t.Run("ClearCache", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Clear cache - should not panic
		vm.ClearCache()

		// The cache is cleared, but we can't directly inspect it
		// This test ensures the function executes without errors
	})
}

func TestVersionManager_GetCacheStats(t *testing.T) {
	t.Run("GetCacheStats", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Get cache stats
		stats := vm.GetCacheStats()

		// Should return valid stats
		if stats == nil {
			t.Error("Expected non-nil cache stats")
		}
	})
}

func TestVersionManager_SetCacheTTL(t *testing.T) {
	t.Run("SetCacheTTL", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Set cache TTL
		vm.SetCacheTTL(300) // 5 minutes

		// The TTL is set, but we can't directly inspect it
		// This test ensures the function executes without errors
	})
}

func TestVersionManager_InstallWorker(t *testing.T) {
	t.Run("InstallWorker_Success", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Install worker
		err := vm.InstallWorker(context.Background(), "worker1", "localhost", 8080)
		if err != nil {
			t.Errorf("Expected no error for worker installation, got %v", err)
		}
	})

	t.Run("InstallWorker_Cancelled", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create a context that will be cancelled quickly
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Install worker with cancelled context - should fail
		err := vm.InstallWorker(ctx, "worker1", "localhost", 8080)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})
}

func TestVersionManager_BatchUpdateWorkers(t *testing.T) {
	t.Run("BatchUpdateWorkers_EmptyList", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Update with empty service list
		result := vm.BatchUpdateWorkers(context.Background(), []*RemoteService{}, 3)

		// Should have zero workers
		if result.TotalWorkers != 0 {
			t.Errorf("Expected 0 total workers, got %d", result.TotalWorkers)
		}
	})

	t.Run("BatchUpdateWorkers_InvalidConcurrency", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test service
		service := &RemoteService{
			WorkerID: "worker1",
			Host:     "localhost",
			Port:     8080,
		}

		// Update with negative concurrency - should use default
		result := vm.BatchUpdateWorkers(context.Background(), []*RemoteService{service}, -1)

		// Should still execute
		if result.TotalWorkers != 1 {
			t.Errorf("Expected 1 total worker, got %d", result.TotalWorkers)
		}
	})
}

func TestVersionManager_GetMetrics(t *testing.T) {
	t.Run("GetMetrics", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Get metrics
		metrics := vm.GetMetrics()

		// Should return metrics
		if metrics == nil {
			t.Error("Expected non-nil metrics")
		}
	})
}

func TestVersionManager_GetAlerts(t *testing.T) {
	t.Run("GetAlerts", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Get alerts
		alerts := vm.GetAlerts()

		// Should return alerts slice (may be empty)
		if alerts == nil {
			t.Error("Expected non-nil alerts slice")
		}
	})
}

func TestVersionManager_AddAlertChannel(t *testing.T) {
	t.Run("AddAlertChannel", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create mock alert channel
		channel := &MockAlertChannel{}

		// Add alert channel
		vm.AddAlertChannel(channel)

		// This test ensures the function executes without errors
	})
}
