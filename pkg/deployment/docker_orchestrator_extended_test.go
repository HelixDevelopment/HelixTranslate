package deployment

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"

	"gopkg.in/yaml.v3"
)

// TestDockerOrchestrator_EmitEventExtended tests the emitEvent function
func TestDockerOrchestrator_EmitEventExtended(t *testing.T) {
	t.Run("EmitEvent_ValidEvent", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Create test event
		event := events.Event{
			Type:      "test_event",
			SessionID: "test-session",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"test_key": "test_value",
			},
		}

		// Emit event - should not panic
		orchestrator.emitEvent(event)

		// Verify event was published by checking the event bus directly
		// Since emitEvent just calls eventBus.Publish, we just verify it doesn't panic
	})
}

// TestDockerOrchestrator_ScaleServiceExtended tests the ScaleService method
func TestDockerOrchestrator_ScaleServiceExtended(t *testing.T) {
	t.Run("ScaleService_ValidCommand", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test scaling a service (will fail but we test command generation)
		ctx := context.Background()
		err := orchestrator.ScaleService(ctx, "test-service", 3)

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when scaling non-existent service")
		}
	})
}

// TestDockerOrchestrator_StopDeploymentExtended tests the StopDeployment method
func TestDockerOrchestrator_StopDeploymentExtended(t *testing.T) {
	t.Run("StopDeployment_NoComposeFile", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test stopping deployment without compose file
		ctx := context.Background()
		err := orchestrator.StopDeployment(ctx)

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when stopping deployment without compose file")
		}
	})
}

// TestDockerOrchestrator_UpdateService tests the UpdateService method
func TestDockerOrchestrator_UpdateService(t *testing.T) {
	t.Run("UpdateService_ValidCommand", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test updating a service (will fail but we test command generation)
		ctx := context.Background()
		err := orchestrator.UpdateService(ctx, "test-service", "nginx:latest")

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when updating non-existent service")
		}
	})
}

// TestDockerOrchestrator_RestartService tests the RestartService method
func TestDockerOrchestrator_RestartService(t *testing.T) {
	t.Run("RestartService_ValidCommand", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test restarting a service (will fail but we test command generation)
		ctx := context.Background()
		err := orchestrator.RestartService(ctx, "test-service")

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when restarting non-existent service")
		}
	})
}

// TestDockerOrchestrator_UpdateAllServices tests the UpdateAllServices method
func TestDockerOrchestrator_UpdateAllServices(t *testing.T) {
	t.Run("UpdateAllServices_NoComposeFile", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test updating all services without compose file
		ctx := context.Background()
		err := orchestrator.UpdateAllServices(ctx)

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when updating all services without compose file")
		}
	})
}

// TestDockerOrchestrator_RestartAllServices tests the RestartAllServices method
func TestDockerOrchestrator_RestartAllServices(t *testing.T) {
	t.Run("RestartAllServices_NoComposeFile", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test restarting all services without compose file
		ctx := context.Background()
		err := orchestrator.RestartAllServices(ctx)

		// Expected to fail since no Docker compose file exists
		if err == nil {
			t.Error("Expected error when restarting all services without compose file")
		}
	})
}

// TestDockerOrchestrator_CleanupExtended tests the Cleanup method
func TestDockerOrchestrator_CleanupExtended(t *testing.T) {
	t.Run("Cleanup_Success", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Create a temporary file in compose directory
		tempFile := filepath.Join(orchestrator.composeDir, "test-file.txt")
		err := ioutil.WriteFile(tempFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		// Clean up
		err = orchestrator.Cleanup()
		if err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}

		// Check if directory was cleaned up
		if _, err := os.Stat(tempFile); err == nil {
			t.Error("Temp file still exists after cleanup")
		}
	})
}

// TestDockerOrchestrator_WaitForServiceHealthy tests the waitForServiceHealthy method
func TestDockerOrchestrator_WaitForServiceHealthy(t *testing.T) {
	t.Run("WaitForServiceHealthy_Timeout", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test waiting for a service that doesn't exist (should timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := orchestrator.waitForServiceHealthy(ctx, "non-existent-service")
		if err == nil {
			t.Error("Expected timeout error when waiting for non-existent service")
		}
	})
}

// TestDockerOrchestrator_CheckServiceHealth tests the checkServiceHealth method
func TestDockerOrchestrator_CheckServiceHealth(t *testing.T) {
	t.Run("CheckServiceHealth_NonExistentService", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test checking health of non-existent service
		ctx := context.Background()
		healthy, err := orchestrator.checkServiceHealth(ctx, "non-existent-service")

		// Expected to fail
		if err == nil {
			t.Error("Expected error when checking health of non-existent service")
		}

		// Should be false when error occurs
		if healthy {
			t.Error("Service should not be healthy when error occurs")
		}
	})
}

// TestDockerOrchestrator_GetServiceStatusExtended tests the GetServiceStatus method
func TestDockerOrchestrator_GetServiceStatusExtended(t *testing.T) {
	t.Run("GetServiceStatus_NonExistentService", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Test getting status of non-existent service
		ctx := context.Background()
		status, err := orchestrator.GetServiceStatus(ctx, "non-existent-service")

		// Expected to fail
		if err == nil {
			t.Error("Expected error when getting status of non-existent service")
		}

		// Should be empty when error occurs
		if status != "" {
			t.Errorf("Expected empty status on error, got '%s'", status)
		}
	})
}

// TestDockerOrchestrator_AddSupportingServices tests the addSupportingServices method
func TestDockerOrchestrator_AddSupportingServices(t *testing.T) {
	t.Run("AddSupportingServices_Valid", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Create compose config
		composeConfig := &DockerComposeConfig{
			Version:  "3.8",
			Services: make(map[string]*DockerServiceConfig),
			Networks: make(map[string]*DockerNetworkConfig),
			Volumes:  make(map[string]*DockerVolumeConfig),
		}

		// Add supporting services
		orchestrator.addSupportingServices(composeConfig)

		// Check that database service was added
		if _, exists := composeConfig.Services["postgres"]; !exists {
			t.Error("PostgreSQL service was not added")
		}

		// Check that Redis service was added
		if _, exists := composeConfig.Services["redis"]; !exists {
			t.Error("Redis service was not added")
		}

		// Check that network was NOT added (added in different method)
		if _, exists := composeConfig.Networks["translator-network"]; exists {
			t.Error("Network should not be added by addSupportingServices")
		}

		// Check that database volume was added
		if _, exists := composeConfig.Volumes["postgres-data"]; !exists {
			t.Error("PostgreSQL volume was not added")
		}

		// Check that Redis volume was added
		if _, exists := composeConfig.Volumes["redis-data"]; !exists {
			t.Error("Redis volume was not added")
		}
	})
}

// TestDockerOrchestrator_WriteComposeFile tests the writeComposeFile method
func TestDockerOrchestrator_WriteComposeFile(t *testing.T) {
	t.Run("WriteComposeFile_Success", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Create temporary file path
		tempFile := filepath.Join(os.TempDir(), "test-compose.yml")
		defer os.Remove(tempFile)

		// Create compose config
		composeConfig := &DockerComposeConfig{
			Version: "3.8",
			Services: map[string]*DockerServiceConfig{
				"test-service": {
					Image:         "nginx:alpine",
					ContainerName: "test-container",
					Ports:         []string{"8080:80"},
					Restart:       "unless-stopped",
				},
			},
		}

		// Write compose file
		err := orchestrator.writeComposeFile(composeConfig, tempFile)
		if err != nil {
			t.Fatalf("Failed to write compose file: %v", err)
		}

		// Read and verify file
		data, err := ioutil.ReadFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to read compose file: %v", err)
		}

		// Parse YAML to verify structure
		var parsedConfig DockerComposeConfig
		err = yaml.Unmarshal(data, &parsedConfig)
		if err != nil {
			t.Fatalf("Failed to parse compose file: %v", err)
		}

		// Verify content
		if parsedConfig.Version != "3.8" {
			t.Errorf("Expected version '3.8', got '%s'", parsedConfig.Version)
		}

		service, exists := parsedConfig.Services["test-service"]
		if !exists {
			t.Fatal("Test service not found in compose file")
		}

		if service.Image != "nginx:alpine" {
			t.Errorf("Expected image 'nginx:alpine', got '%s'", service.Image)
		}
	})

	t.Run("WriteComposeFile_InvalidPath", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		// Try to write to invalid path
		invalidPath := "/invalid/path/that/does/not/exist/compose.yml"

		// Create compose config
		composeConfig := &DockerComposeConfig{
			Version: "3.8",
		}

		// Try to write file
		err := orchestrator.writeComposeFile(composeConfig, invalidPath)
		if err == nil {
			t.Error("Expected error when writing to invalid path")
		}
	})
}
