package distributed

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
)

// defaultLogger provides a basic logger implementation
type defaultLogger struct{}

func (l *defaultLogger) Log(level, message string, fields map[string]interface{}) {
	log.Printf("[%s] %s %v", level, message, fields)
}

// DistributedManager manages all distributed work functionality
type DistributedManager struct {
	config           *config.Config
	sshPool          *SSHPool
	pairingManager   *PairingManager
	distributedCoord *DistributedCoordinator
	fallbackManager  *FallbackManager
	eventBus         *events.EventBus
	mu               sync.RWMutex
	initialized      bool
}

// NewDistributedManager creates a new distributed manager
func NewDistributedManager(cfg *config.Config, eventBus *events.EventBus, apiLogger *deployment.APICommunicationLogger) *DistributedManager {
	sshPool := NewSSHPool()
	pairingManager := NewPairingManager(sshPool, eventBus)

	// Create fallback manager with default config
	fallbackConfig := DefaultFallbackConfig()
	fallbackManager := NewFallbackManager(fallbackConfig, nil, eventBus, &defaultLogger{})

	// Create distributed coordinator (will be initialized with local coordinator later)
	distributedCoord := NewDistributedCoordinator(nil, sshPool, pairingManager, fallbackManager, eventBus, apiLogger)

	return &DistributedManager{
		config:           cfg,
		sshPool:          sshPool,
		pairingManager:   pairingManager,
		distributedCoord: distributedCoord,
		fallbackManager:  fallbackManager,
		eventBus:         eventBus,
		initialized:      false,
	}
}

// Initialize initializes the distributed manager with worker configurations
func (dm *DistributedManager) Initialize(localCoordinator interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.initialized {
		return fmt.Errorf("distributed manager already initialized")
	}

	// Set local coordinator reference
	dm.distributedCoord.localCoordinator = localCoordinator

	// Load worker configurations
	for workerID, workerCfg := range dm.config.Distributed.Workers {
		sshConfig := SSHConfig{
			Host:       workerCfg.Host,
			Port:       workerCfg.Port,
			User:       workerCfg.User,
			KeyFile:    workerCfg.KeyFile,
			Password:   workerCfg.Password,
			Timeout:    time.Duration(dm.config.Distributed.SSHTimeout) * time.Second,
			MaxRetries: dm.config.Distributed.SSHMaxRetries,
			RetryDelay: 5 * time.Second,
		}

		distWorkerCfg := &WorkerConfig{
			ID:          workerID,
			Name:        workerCfg.Name,
			SSH:         sshConfig,
			Tags:        workerCfg.Tags,
			MaxCapacity: workerCfg.MaxCapacity,
			Enabled:     workerCfg.Enabled,
		}

		dm.sshPool.AddWorker(distWorkerCfg)
	}

	dm.initialized = true

	dm.emitEvent(events.Event{
		Type:      "distributed_manager_initialized",
		SessionID: "system",
		Message:   fmt.Sprintf("Distributed manager initialized with %d workers", len(dm.config.Distributed.Workers)),
		Data: map[string]interface{}{
			"worker_count": len(dm.config.Distributed.Workers),
		},
	})

	return nil
}

// DiscoverAndPairWorkers discovers and pairs with all configured workers
func (dm *DistributedManager) DiscoverAndPairWorkers(ctx context.Context) error {
	dm.mu.RLock()
	if !dm.initialized {
		dm.mu.RUnlock()
		return fmt.Errorf("distributed manager not initialized")
	}
	dm.mu.RUnlock()

	workers := dm.sshPool.GetWorkers()
	successCount := 0

	for workerID := range workers {
		if err := dm.discoverAndPairWorker(ctx, workerID); err != nil {
			dm.emitWarning(fmt.Sprintf("Failed to discover/pair worker %s: %v", workerID, err))
			continue
		}
		successCount++
	}

	// Discover remote instances
	if err := dm.distributedCoord.DiscoverRemoteInstances(ctx); err != nil {
		dm.emitWarning(fmt.Sprintf("Failed to discover remote instances: %v", err))
	}

	dm.emitEvent(events.Event{
		Type:      "distributed_workers_paired",
		SessionID: "system",
		Message:   fmt.Sprintf("Successfully paired with %d/%d workers", successCount, len(workers)),
		Data: map[string]interface{}{
			"paired_count":     successCount,
			"total_count":      len(workers),
			"remote_instances": dm.distributedCoord.GetRemoteInstanceCount(),
		},
	})

	return nil
}

// discoverAndPairWorker discovers and pairs with a single worker
func (dm *DistributedManager) discoverAndPairWorker(ctx context.Context, workerID string) error {
	// Discover service
	service, err := dm.pairingManager.DiscoverService(ctx, workerID)
	if err != nil {
		return fmt.Errorf("failed to discover service: %w", err)
	}

	// Pair with service
	if err := dm.pairingManager.PairWithService(workerID); err != nil {
		return fmt.Errorf("failed to pair with service: %w", err)
	}

	dm.emitEvent(events.Event{
		Type:      "distributed_worker_discovered",
		SessionID: "system",
		Message:   fmt.Sprintf("Discovered and paired with worker %s (%s)", workerID, service.Name),
		Data: map[string]interface{}{
			"worker_id":    workerID,
			"worker_name":  service.Name,
			"host":         service.Host,
			"capabilities": service.Capabilities,
		},
	})

	return nil
}

// TranslateDistributed performs distributed translation
func (dm *DistributedManager) TranslateDistributed(
	ctx context.Context,
	text string,
	contextHint string,
) (string, error) {

	dm.mu.RLock()
	if !dm.initialized {
		dm.mu.RUnlock()
		return "", fmt.Errorf("distributed manager not initialized")
	}
	dm.mu.RUnlock()

	return dm.distributedCoord.TranslateWithDistributedRetry(ctx, text, contextHint)
}

// GetStatus returns the status of all workers and instances
func (dm *DistributedManager) GetStatus() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	workers := dm.sshPool.GetWorkers()
	pairedServices := dm.pairingManager.GetPairedServices()

	workerStatuses := make(map[string]interface{})
	for workerID, worker := range workers {
		status := "configured"
		if service, paired := pairedServices[workerID]; paired {
			status = service.Status
		}

		workerStatuses[workerID] = map[string]interface{}{
			"name":     worker.Name,
			"enabled":  worker.Enabled,
			"status":   status,
			"capacity": worker.MaxCapacity,
		}
	}

	return map[string]interface{}{
		"initialized":        dm.initialized,
		"enabled":            dm.config.Distributed.Enabled,
		"workers":            workerStatuses,
		"active_connections": dm.sshPool.GetActiveConnections(),
		"remote_instances":   dm.distributedCoord.GetRemoteInstanceCount(),
		"paired_workers":     len(pairedServices),
	}
}

// AddWorker adds a new worker dynamically
func (dm *DistributedManager) AddWorker(workerID string, workerCfg *WorkerConfig) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.initialized {
		return fmt.Errorf("distributed manager not initialized")
	}

	dm.sshPool.AddWorker(workerCfg)
	dm.config.Distributed.Workers[workerID] = config.WorkerConfig{
		Name:        workerCfg.Name,
		Host:        workerCfg.SSH.Host,
		Port:        workerCfg.SSH.Port,
		User:        workerCfg.SSH.User,
		KeyFile:     workerCfg.SSH.KeyFile,
		Password:    workerCfg.SSH.Password,
		MaxCapacity: workerCfg.MaxCapacity,
		Tags:        workerCfg.Tags,
		Enabled:     workerCfg.Enabled,
	}

	dm.emitEvent(events.Event{
		Type:      "distributed_worker_added",
		SessionID: "system",
		Message:   fmt.Sprintf("Worker %s added to distributed pool", workerID),
		Data: map[string]interface{}{
			"worker_id":   workerID,
			"worker_name": workerCfg.Name,
		},
	})

	return nil
}

// RemoveWorker removes a worker
func (dm *DistributedManager) RemoveWorker(workerID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.initialized {
		return fmt.Errorf("distributed manager not initialized")
	}

	dm.sshPool.RemoveWorker(workerID)
	delete(dm.config.Distributed.Workers, workerID)

	dm.emitEvent(events.Event{
		Type:      "distributed_worker_removed",
		SessionID: "system",
		Message:   fmt.Sprintf("Worker %s removed from distributed pool", workerID),
		Data: map[string]interface{}{
			"worker_id": workerID,
		},
	})

	return nil
}

// PairWorker pairs with a worker
func (dm *DistributedManager) PairWorker(workerID string) error {
	dm.mu.RLock()
	if !dm.initialized {
		dm.mu.RUnlock()
		return fmt.Errorf("distributed manager not initialized")
	}
	dm.mu.RUnlock()

	return dm.pairingManager.PairWithService(workerID)
}

// UnpairWorker unpairs from a worker
func (dm *DistributedManager) UnpairWorker(workerID string) error {
	dm.mu.RLock()
	if !dm.initialized {
		dm.mu.RUnlock()
		return fmt.Errorf("distributed manager not initialized")
	}
	dm.mu.RUnlock()

	return dm.pairingManager.UnpairService(workerID)
}

// emitEvent emits an event if event bus is available
func (dm *DistributedManager) emitEvent(event events.Event) {
	if dm.eventBus != nil {
		dm.eventBus.Publish(event)
	}
}

// emitWarning emits a warning event
func (dm *DistributedManager) emitWarning(message string) {
	if dm.eventBus != nil {
		dm.eventBus.Publish(events.Event{
			Type:      "distributed_warning",
			SessionID: "system",
			Message:   message,
		})
	}
}

// Close shuts down the distributed manager
func (dm *DistributedManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.pairingManager.Close()
	dm.sshPool.Close()

	dm.emitEvent(events.Event{
		Type:      "distributed_manager_shutdown",
		SessionID: "system",
		Message:   "Distributed manager shut down",
	})

	return nil
}
