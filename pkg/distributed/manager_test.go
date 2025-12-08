package distributed

import (
	"context"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
	"testing"
)

func TestDistributedManager_emitWarning(t *testing.T) {
	t.Run("EmitWarningWithEventBus", func(t *testing.T) {
		// Create an event bus
		eventBus := events.NewEventBus()

		// Create manager with event bus
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Emit warning (should not panic)
		manager.emitWarning("test warning message")

		// We can't easily verify the event was published without more complex setup
		// Just verify no panic occurred
	})

	t.Run("EmitWarningWithoutEventBus", func(t *testing.T) {
		// Create manager without event bus
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Emit warning (should not panic even with nil event bus)
		manager.emitWarning("test warning message")

		// Should not panic
	})
}

func TestDistributedManager_GetVersionMetrics(t *testing.T) {
	t.Run("GetMetricsUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Get metrics from uninitialized manager
		metrics := manager.GetVersionMetrics()

		// Should return empty metrics for uninitialized manager
		if metrics == nil {
			t.Error("Expected non-empty metrics struct")
		}
	})

	t.Run("GetMetricsInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get metrics from initialized manager
		metrics := manager.GetVersionMetrics()

		// Should return metrics struct
		if metrics == nil {
			t.Error("Expected non-empty metrics struct")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetVersionAlerts(t *testing.T) {
	t.Run("GetAlertsUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Get alerts from uninitialized manager
		alerts := manager.GetVersionAlerts()

		// Should return empty slice for uninitialized manager
		if alerts == nil {
			t.Error("Expected non-nil alerts slice")
		}
		if len(alerts) != 0 {
			t.Error("Expected empty alerts slice for uninitialized manager")
		}
	})

	t.Run("GetAlertsInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get alerts from initialized manager
		alerts := manager.GetVersionAlerts()

		// Should return alerts slice
		if alerts == nil {
			t.Error("Expected non-nil alerts slice")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetVersionHealth(t *testing.T) {
	t.Run("GetHealthUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Get health from uninitialized manager
		health := manager.GetVersionHealth()

		// Should return zero health values for uninitialized manager
		if health["status"] != "uninitialized" {
			t.Error("Expected uninitialized status")
		}
	})

	t.Run("GetHealthInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get health from initialized manager
		health := manager.GetVersionHealth()

		// Should return health struct
		if health["status"] == nil {
			t.Error("Expected status in health map")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetAlertHistory(t *testing.T) {
	t.Run("GetAlertHistoryUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Get alert history from uninitialized manager
		history := manager.GetAlertHistory(10)

		// Should return empty slice for uninitialized manager
		if history == nil {
			t.Error("Expected non-nil alert history slice")
		}
		if len(history) != 0 {
			t.Error("Expected empty alert history for uninitialized manager")
		}
	})

	t.Run("GetAlertHistoryInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get alert history from initialized manager
		history := manager.GetAlertHistory(10)

		// Should return history slice
		if history == nil {
			t.Error("Expected non-nil alert history slice")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_AcknowledgeAlert(t *testing.T) {
	t.Run("AcknowledgeAlertUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to acknowledge alert with uninitialized manager
		result := manager.AcknowledgeAlert("non-existent-alert-id", "test-user")

		// Should handle gracefully (no panic)
		if result {
			t.Error("Expected false when acknowledging alert with uninitialized manager")
		}
	})

	t.Run("AcknowledgeAlertInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to acknowledge non-existent alert
		result := manager.AcknowledgeAlert("non-existent-alert-id", "test-user")

		// Should handle gracefully
		if result {
			t.Error("Expected false when acknowledging non-existent alert")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_AddAlertChannel(t *testing.T) {
	t.Run("AddAlertChannelUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Create a mock alert channel
		channel := &MockAlertChannel{}

		// Try to add alert channel with uninitialized manager
		manager.AddAlertChannel(channel)

		// Should handle gracefully (no panic)
		// No assertion needed - just verify it doesn't panic
	})

	t.Run("AddAlertChannelInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Create a mock alert channel
		channel := &MockAlertChannel{}

		// Add alert channel
		manager.AddAlertChannel(channel)
		// Should not panic

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_RemoveWorker(t *testing.T) {
	t.Run("RemoveWorkerUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to remove worker with uninitialized manager
		err := manager.RemoveWorker("non-existent-worker")

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when removing worker with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("RemoveWorkerInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to remove non-existent worker (should not panic)
		err = manager.RemoveWorker("non-existent-worker")
		// Should not error even for non-existent worker
		if err != nil {
			t.Errorf("Unexpected error when removing non-existent worker: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_PairWorker(t *testing.T) {
	t.Run("PairWorkerUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to pair worker with uninitialized manager
		err := manager.PairWorker("non-existent-worker")

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when pairing worker with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("PairWorkerInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to pair non-existent worker
		err = manager.PairWorker("non-existent-worker")
		// May error due to non-existent worker, but should not panic
		if err != nil {
			t.Logf("Expected error for pairing non-existent worker: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_UnpairWorker(t *testing.T) {
	t.Run("UnpairWorkerUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to unpair worker with uninitialized manager
		err := manager.UnpairWorker("non-existent-worker")

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when unpairing worker with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("UnpairWorkerInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to unpair non-existent worker
		err = manager.UnpairWorker("non-existent-worker")
		// May error due to non-existent worker, but should not panic
		if err != nil {
			t.Logf("Expected error for unpairing non-existent worker: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_DiscoverAndPairWorkers(t *testing.T) {
	t.Run("DiscoverAndPairWorkersUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to discover and pair workers with uninitialized manager
		err := manager.DiscoverAndPairWorkers(context.Background())

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when discovering workers with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("DiscoverAndPairWorkersInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		eventBus := events.NewEventBus() // Add event bus to prevent panic
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to discover and pair workers (no workers configured)
		err = manager.DiscoverAndPairWorkers(context.Background())
		// Should not error even with no workers
		if err != nil {
			t.Errorf("Unexpected error when discovering with no workers: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetWorkerByID(t *testing.T) {
	t.Run("GetWorkerByIDUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to get worker with uninitialized manager
		worker := manager.GetWorkerByID("non-existent-worker")

		// Should return nil for uninitialized manager
		if worker != nil {
			t.Error("Expected nil when getting worker with uninitialized manager")
		}
	})

	t.Run("GetWorkerByIDInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to get non-existent worker
		worker := manager.GetWorkerByID("non-existent-worker")

		// Should return nil for non-existent worker
		if worker != nil {
			t.Error("Expected nil for non-existent worker")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_RollbackWorker(t *testing.T) {
	t.Run("RollbackWorkerUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to rollback worker with uninitialized manager
		service := &RemoteService{}
		err := manager.RollbackWorker(context.Background(), service)

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when rolling back worker with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("RollbackWorkerInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to rollback worker
		service := &RemoteService{}
		err = manager.RollbackWorker(context.Background(), service)
		// May error due to no actual worker to rollback, but should not panic
		if err != nil {
			t.Logf("Expected error for rollback without valid worker: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetStatus(t *testing.T) {
	t.Run("GetStatusUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Get status from uninitialized manager
		status := manager.GetStatus()

		// Should return status with initialized=false
		if status["initialized"] != false {
			t.Error("Expected initialized=false")
		}

		// Should have workers, active_connections, remote_instances, and paired_workers keys
		if _, ok := status["workers"]; !ok {
			t.Error("Expected workers key in status")
		}
		if _, ok := status["active_connections"]; !ok {
			t.Error("Expected active_connections key in status")
		}
		if _, ok := status["remote_instances"]; !ok {
			t.Error("Expected remote_instances key in status")
		}
		if _, ok := status["paired_workers"]; !ok {
			t.Error("Expected paired_workers key in status")
		}
	})

	t.Run("GetStatusInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get status from initialized manager
		status := manager.GetStatus()

		// Should return status with initialized=true
		if status["initialized"] != true {
			t.Error("Expected initialized=true")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_GetPairedServices(t *testing.T) {
	t.Run("GetPairedServicesUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to get paired services with uninitialized manager
		services := manager.GetPairedServices()

		// Should return empty map for uninitialized manager
		if services == nil {
			t.Error("Expected non-nil paired services map")
		}
		if len(services) != 0 {
			t.Error("Expected empty paired services map for uninitialized manager")
		}
	})

	t.Run("GetPairedServicesInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Get paired services from initialized manager
		services := manager.GetPairedServices()

		// Should return paired services map
		if services == nil {
			t.Error("Expected non-nil paired services map")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_CheckVersionDrift(t *testing.T) {
	t.Run("CheckVersionDriftUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to check version drift with uninitialized manager
		alerts := manager.CheckVersionDrift(context.Background())

		// Should return empty slice for uninitialized manager
		if alerts == nil {
			t.Error("Expected non-nil alerts slice")
		}
		if len(alerts) != 0 {
			t.Error("Expected empty alerts slice for uninitialized manager")
		}
	})

	t.Run("CheckVersionDriftInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		eventBus := events.NewEventBus() // Add event bus to prevent panic
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Check version drift from initialized manager
		alerts := manager.CheckVersionDrift(context.Background())

		// Should return alerts slice
		if alerts == nil {
			t.Error("Expected non-nil alerts slice")
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_AddWorker(t *testing.T) {
	t.Run("AddWorkerUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to add worker with uninitialized manager
		workerCfg := &WorkerConfig{
			Name: "test-worker",
			SSH: SSHConfig{
				Host: "localhost",
				Port: 22,
				User: "test",
			},
		}

		err := manager.AddWorker("test-worker", workerCfg)

		// Should return error for uninitialized manager
		if err == nil {
			t.Error("Expected error when adding worker with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("AddWorkerInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		eventBus := events.NewEventBus() // Add event bus to prevent panic
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Add worker
		workerCfg := &WorkerConfig{
			Name: "test-worker",
			SSH: SSHConfig{
				Host: "localhost",
				Port: 22,
				User: "test",
			},
		}

		err = manager.AddWorker("test-worker", workerCfg)
		// May error due to SSH configuration, but should not panic
		if err != nil {
			t.Logf("Expected error for adding worker with test config: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_TranslateDistributed(t *testing.T) {
	t.Run("TranslateDistributedUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Try to translate with uninitialized manager
		_, err := manager.TranslateDistributed(context.Background(), "hello", "test")

		// Should handle gracefully
		if err == nil {
			t.Error("Expected error when translating with uninitialized manager")
		}
		if err.Error() != "distributed manager not initialized" {
			t.Errorf("Expected 'distributed manager not initialized' error, got: %v", err)
		}
	})

	t.Run("TranslateDistributedInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		eventBus := events.NewEventBus()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Try to translate (will likely error due to no workers)
		_, err = manager.TranslateDistributed(context.Background(), "hello", "test")

		// Should handle gracefully - may error due to no workers
		if err != nil {
			t.Logf("Expected error for translation without workers: %v", err)
		}

		// Close manager to clean up
		manager.Close()
	})
}

func TestDistributedManager_Close(t *testing.T) {
	t.Run("CloseUninitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, nil, apiLogger)

		// Close uninitialized manager
		err := manager.Close()

		// Should not error even when uninitialized
		if err != nil {
			t.Errorf("Unexpected error when closing uninitialized manager: %v", err)
		}
	})

	t.Run("CloseInitialized", func(t *testing.T) {
		cfg := config.DefaultConfig()
		eventBus := events.NewEventBus()
		apiLogger := &deployment.APICommunicationLogger{}
		manager := NewDistributedManager(cfg, eventBus, apiLogger)

		// Initialize manager
		err := manager.Initialize(nil)
		if err != nil {
			t.Fatalf("Failed to initialize manager: %v", err)
		}

		// Close initialized manager
		err = manager.Close()

		// Should not error
		if err != nil {
			t.Errorf("Unexpected error when closing initialized manager: %v", err)
		}

		// Verify manager is properly closed (cannot initialize again)
		err = manager.Initialize(nil)
		if err == nil {
			t.Error("Expected error when trying to initialize already closed manager")
		}
	})
}
