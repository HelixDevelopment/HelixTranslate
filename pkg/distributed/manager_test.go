package distributed

import (
	"testing"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"
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