//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/distributed"
	"digital.vasic.translator/pkg/events"
)

func TestDistributedManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test configuration
	cfg := &config.Config{}
	cfg.Distributed.Enabled = true
	cfg.Distributed.Workers = map[string]config.WorkerConfig{
		"test-worker": {
			Name:        "Test Worker",
			Host:        "localhost",
			Port:        2222, // Use a port that's likely not in use
			User:        "test",
			MaxCapacity: 5,
			Enabled:     false, // Disable to avoid actual SSH connections
		},
	}

	eventBus := events.NewEventBus()
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	manager := distributed.NewDistributedManager(cfg, eventBus)

	// Test initialization
	err := manager.Initialize(localCoordinator)
	if err != nil {
		t.Fatalf("Failed to initialize distributed manager: %v", err)
	}

	// Test status
	status := manager.GetStatus()
	if !status["initialized"].(bool) {
		t.Error("Manager should be initialized")
	}

	if !status["enabled"].(bool) {
		t.Error("Manager should be enabled")
	}

	workers := status["workers"].(map[string]interface{})
	if len(workers) != 1 {
		t.Errorf("Expected 1 worker, got %d", len(workers))
	}

	// Test that we can add a worker
	newWorker := &distributed.WorkerConfig{
		ID:          "dynamic-worker",
		Name:        "Dynamic Test Worker",
		SSH:         distributed.SSHConfig{Host: "example.com", User: "test"},
		MaxCapacity: 10,
		Enabled:     false,
	}

	err = manager.AddWorker("dynamic-worker", newWorker)
	if err != nil {
		t.Fatalf("Failed to add worker: %v", err)
	}

	// Verify worker was added
	status = manager.GetStatus()
	workers = status["workers"].(map[string]interface{})
	if len(workers) != 2 {
		t.Errorf("Expected 2 workers after adding, got %d", len(workers))
	}

	// Test distributed translation (should fail gracefully since no real workers)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = manager.TranslateDistributed(ctx, "Hello world", "greeting")
	if err == nil {
		t.Error("Expected error for distributed translation with no real workers")
	}

	// Test cleanup
	err = manager.Close()
	if err != nil {
		t.Errorf("Failed to close manager: %v", err)
	}
}

func TestDistributedManager_EventEmission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{}
	cfg.Distributed.Enabled = true

	eventBus := events.NewEventBus()

	// Subscribe to events
	events := make([]events.Event, 0)
	eventBus.SubscribeAll(func(event events.Event) {
		events = append(events, event)
	})

	manager := distributed.NewDistributedManager(cfg, eventBus)

	// Test that initialization emits events
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	err := manager.Initialize(localCoordinator)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Give events time to be processed
	time.Sleep(100 * time.Millisecond)

	// Check for initialization event
	found := false
	for _, event := range events {
		if event.Type == "distributed_manager_initialized" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected distributed_manager_initialized event")
	}
}
