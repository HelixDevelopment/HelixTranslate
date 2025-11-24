package distributed_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data structures
type WorkerNode struct {
	ID         string                 `json:"id"`
	Host       string                 `json:"host"`
	Port       int                    `json:"port"`
	Status     string                 `json:"status"`
	LastSeen   time.Time             `json:"last_seen"`
	Load       float64               `json:"load"`
	ActiveJobs int                   `json:"active_jobs"`
	MaxJobs    int                   `json:"max_jobs"`
	Tags       map[string]string      `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type ManagerConfig struct {
	DiscoveryInterval    time.Duration `json:"discovery_interval"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	LoadBalanceStrategy string         `json:"load_balance_strategy"`
	MaxRetries          int           `json:"max_retries"`
	Timeout             time.Duration `json:"timeout"`
}

type DistributedManager struct {
	config          ManagerConfig
	workers         map[string]*WorkerNode
	running         bool
	startTime       time.Time
	index           int // For round-robin strategy
	timeoutOverride time.Duration // For testing health check timeouts
}

type JobAssignment struct {
	JobID      string    `json:"job_id"`
	WorkerID   string    `json:"worker_id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

func TestDistributedManager_WorkerDiscovery(t *testing.T) {
	config := ManagerConfig{
		DiscoveryInterval:   30 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		LoadBalanceStrategy: "round_robin",
		MaxRetries:          3,
		Timeout:             5 * time.Second,
	}
	
	manager := NewDistributedManager(config)
	
	// Test 1: Discover new workers
	t.Run("DiscoverNewWorkers", func(t *testing.T) {
		// Mock worker discovery
		workers := []*WorkerNode{
			{
				ID:         "worker-1",
				Host:       "worker1.local",
				Port:       8080,
				Status:     "active",
				Load:       0.2,
				ActiveJobs: 2,
				MaxJobs:    10,
				Tags:       map[string]string{"region": "us-west", "capacity": "high"},
			},
			{
				ID:         "worker-2",
				Host:       "worker2.local",
				Port:       8080,
				Status:     "active",
				Load:       0.7,
				ActiveJobs: 7,
				MaxJobs:    10,
				Tags:       map[string]string{"region": "us-east", "capacity": "medium"},
			},
		}
		
		for _, worker := range workers {
			err := manager.RegisterWorker(worker)
			require.NoError(t, err)
		}
		
		// Verify workers are registered
		registeredWorkers := manager.GetWorkers()
		assert.Len(t, registeredWorkers, 2)
		
		for _, worker := range workers {
			registered, exists := registeredWorkers[worker.ID]
			assert.True(t, exists)
			assert.Equal(t, worker.Host, registered.Host)
			assert.Equal(t, worker.Status, registered.Status)
		}
	})
	
	// Test 2: Duplicate worker registration
	t.Run("DuplicateWorkerRegistration", func(t *testing.T) {
		duplicateWorker := &WorkerNode{
			ID:     "worker-1", // Same ID as above
			Host:   "worker1-updated.local",
			Port:   8080,
			Status: "active",
		}
		
		err := manager.RegisterWorker(duplicateWorker)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "worker already exists")
	})
	
	// Test 3: Remove workers
	t.Run("RemoveWorkers", func(t *testing.T) {
		err := manager.UnregisterWorker("worker-1")
		require.NoError(t, err)
		
		workers := manager.GetWorkers()
		assert.Len(t, workers, 1)
		
		_, exists := workers["worker-1"]
		assert.False(t, exists)
		
		_, exists = workers["worker-2"]
		assert.True(t, exists)
	})
	
	// Test 4: Remove non-existent worker
	t.Run("RemoveNonExistentWorker", func(t *testing.T) {
		err := manager.UnregisterWorker("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "worker not found")
	})
}

func TestDistributedManager_JobAssignment(t *testing.T) {
	config := ManagerConfig{
		DiscoveryInterval:   30 * time.Second,
		HealthCheckInterval: 10 * time.Second,
		LoadBalanceStrategy: "least_loaded",
		MaxRetries:          3,
		Timeout:             5 * time.Second,
	}
	
	manager := NewDistributedManager(config)
	
	// Register workers
	workers := []*WorkerNode{
		{
			ID:         "worker-light",
			Host:       "light.local",
			Status:     "active",
			Load:       0.1,
			ActiveJobs: 1,
			MaxJobs:    10,
		},
		{
			ID:         "worker-heavy",
			Host:       "heavy.local",
			Status:     "active",
			Load:       0.8,
			ActiveJobs: 8,
			MaxJobs:    10,
		},
	}
	
	for _, worker := range workers {
		err := manager.RegisterWorker(worker)
		require.NoError(t, err)
	}
	
	// Test 1: Assign job to least loaded worker
	t.Run("AssignJobToLeastLoaded", func(t *testing.T) {
		assignment, err := manager.AssignJob("test-job-1")
		require.NoError(t, err)
		assert.Equal(t, "test-job-1", assignment.JobID)
		assert.Equal(t, "worker-light", assignment.WorkerID) // Should assign to lighter worker
		assert.Equal(t, "assigned", assignment.Status)
		assert.NotZero(t, assignment.StartTime)
	})
	
	// Test 2: Assign job with no workers available
	t.Run("AssignJobNoWorkers", func(t *testing.T) {
		// Remove all workers
		manager.UnregisterWorker("worker-light")
		manager.UnregisterWorker("worker-heavy")
		
		_, err := manager.AssignJob("test-job-2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no available workers")
	})
	
	// Test 3: Assign job with specific worker preference
	t.Run("AssignJobToSpecificWorker", func(t *testing.T) {
		// Re-register workers
		for _, worker := range workers {
			manager.RegisterWorker(worker)
		}
		
		assignment, err := manager.AssignJobToWorker("test-job-3", "worker-heavy")
		require.NoError(t, err)
		assert.Equal(t, "worker-heavy", assignment.WorkerID)
	})
	
	// Test 4: Assign job to non-existent worker
	t.Run("AssignJobToNonExistentWorker", func(t *testing.T) {
		_, err := manager.AssignJobToWorker("test-job-4", "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "worker not found")
	})
}

func TestDistributedManager_LoadBalancing(t *testing.T) {
	// Test 1: Round robin strategy
	t.Run("RoundRobinStrategy", func(t *testing.T) {
		config := ManagerConfig{
			LoadBalanceStrategy: "round_robin",
		}
		
		manager := NewDistributedManager(config)
		
		// Register workers
		workers := []*WorkerNode{
			{ID: "worker-1", Status: "active", MaxJobs: 10},
			{ID: "worker-2", Status: "active", MaxJobs: 10},
			{ID: "worker-3", Status: "active", MaxJobs: 10},
		}
		
		for _, worker := range workers {
			manager.RegisterWorker(worker)
		}
		
		// Assign jobs and check round-robin distribution
		assignments := make(map[string]int)
		
		for i := 0; i < 9; i++ {
			assignment, err := manager.AssignJob(fmt.Sprintf("job-%d", i))
			require.NoError(t, err)
			assignments[assignment.WorkerID]++
		}
		
		// Should be distributed among all workers (not perfectly even due to MaxJobs=10)
		assert.Equal(t, 3, len(assignments)) // Should have assignments for all 3 workers
		for workerID, count := range assignments {
			assert.Greater(t, count, 0, "Worker %s should have received jobs", workerID)
			assert.LessOrEqual(t, count, 10, "Worker %s should not exceed max jobs", workerID)
		}
	})
	
	// Test 2: Least loaded strategy
	t.Run("LeastLoadedStrategy", func(t *testing.T) {
		config := ManagerConfig{
			LoadBalanceStrategy: "least_loaded",
		}
		
		manager := NewDistributedManager(config)
		
		// Register workers with different loads
		workers := []*WorkerNode{
			{ID: "worker-busy", Status: "active", Load: 0.8, ActiveJobs: 8, MaxJobs: 10},
			{ID: "worker-free", Status: "active", Load: 0.1, ActiveJobs: 1, MaxJobs: 10},
		}
		
		for _, worker := range workers {
			manager.RegisterWorker(worker)
		}
		
		// Assign multiple jobs
		for i := 0; i < 5; i++ {
			assignment, err := manager.AssignJob(fmt.Sprintf("job-%d", i))
			require.NoError(t, err)
			
			// All should go to the free worker initially
			if i < 4 { // Free worker has capacity for 4 more jobs (max 10 - current 1 = 9)
				assert.Equal(t, "worker-free", assignment.WorkerID)
			}
		}
	})
	
	// Test 3: Random strategy
	t.Run("RandomStrategy", func(t *testing.T) {
		config := ManagerConfig{
			LoadBalanceStrategy: "random",
		}
		
		manager := NewDistributedManager(config)
		
		// Register workers
		workers := []*WorkerNode{
			{ID: "worker-1", Status: "active", MaxJobs: 10},
			{ID: "worker-2", Status: "active", MaxJobs: 10},
		}
		
		for _, worker := range workers {
			manager.RegisterWorker(worker)
		}
		
		// Assign jobs and check distribution
		assignments := make(map[string]int)
		
		for i := 0; i < 20; i++ {
			assignment, err := manager.AssignJob(fmt.Sprintf("job-%d", i))
			require.NoError(t, err)
			assignments[assignment.WorkerID]++
		}
		
		// Should be somewhat distributed (not all on one worker)
		assert.Greater(t, assignments["worker-1"], 0)
		assert.Greater(t, assignments["worker-2"], 0)
		assert.Less(t, assignments["worker-1"], 20)
		assert.Less(t, assignments["worker-2"], 20)
	})
}

func TestDistributedManager_HealthChecks(t *testing.T) {
	config := ManagerConfig{
		HealthCheckInterval: 1 * time.Second, // Short for testing
		MaxRetries:         3,
		Timeout:            2 * time.Second,
	}
	
	manager := NewDistributedManager(config)
	
	// Register workers
	workers := []*WorkerNode{
		{
			ID:         "worker-healthy",
			Host:       "healthy.local",
			Port:       8080,
			Status:     "active",
			LastSeen:   time.Now(),
		},
		{
			ID:         "worker-unhealthy",
			Host:       "unhealthy.local",
			Port:       8080,
			Status:     "active",
			LastSeen:   time.Now().Add(-5 * time.Minute), // Old timestamp
		},
	}
	
	for _, worker := range workers {
		manager.RegisterWorker(worker)
	}
	
	// Test 1: Health check marks unhealthy workers
	t.Run("HealthCheckMarksUnhealthy", func(t *testing.T) {
		// Set very short timeout for testing
		timeout := 1 * time.Second
		
		// Update the health check method to use this timeout
		manager.timeoutOverride = timeout
		
		// Force update the unhealthy worker to be even older
		unhealthyWorker := manager.workers["worker-unhealthy"]
		unhealthyWorker.LastSeen = time.Now().Add(-2 * time.Second) // Older than timeout
		
		err := manager.PerformHealthCheck()
		require.NoError(t, err)
		
		workers := manager.GetWorkers()
		
		// Healthy worker should still be active
		healthyWorker := workers["worker-healthy"]
		assert.Equal(t, "active", healthyWorker.Status)
		
		// Unhealthy worker should be marked as inactive
		unhealthyWorker = workers["worker-unhealthy"]
		assert.Equal(t, "inactive", unhealthyWorker.Status)
	})
	
	// Test 2: Health check recovery
	t.Run("HealthCheckRecovery", func(t *testing.T) {
		// Update last seen for unhealthy worker
		workers := manager.GetWorkers()
		unhealthyWorker := workers["worker-unhealthy"]
		unhealthyWorker.LastSeen = time.Now()
		unhealthyWorker.Status = "active"
		
		// Perform health check again
		err := manager.PerformHealthCheck()
		require.NoError(t, err)
		
		workers = manager.GetWorkers()
		updatedWorker := workers["worker-unhealthy"]
		assert.Equal(t, "active", updatedWorker.Status)
	})
}

func TestDistributedManager_ConcurrentOperations(t *testing.T) {
	config := ManagerConfig{
		LoadBalanceStrategy: "least_loaded",
		MaxRetries:         3,
	}
	
	manager := NewDistributedManager(config)
	
	// Register workers
	workers := []*WorkerNode{
		{ID: "worker-1", Status: "active", Load: 0.2, MaxJobs: 10},
		{ID: "worker-2", Status: "active", Load: 0.3, MaxJobs: 10},
	}
	
	for _, worker := range workers {
		manager.RegisterWorker(worker)
	}
	
	// Test: Concurrent job assignments
	t.Run("ConcurrentJobAssignments", func(t *testing.T) {
		const numJobs = 50
		results := make(chan *JobAssignment, numJobs)
		errors := make(chan error, numJobs)
		
		// Launch concurrent job assignments
		for i := 0; i < numJobs; i++ {
			go func(id int) {
				assignment, err := manager.AssignJob(fmt.Sprintf("job-%d", id))
				if err != nil {
					errors <- err
				} else {
					results <- assignment
				}
			}(i)
		}
		
		// Collect results
		assignments := make([]*JobAssignment, 0)
		errorCount := 0
		
		for i := 0; i < numJobs; i++ {
			select {
			case assignment := <-results:
				assignments = append(assignments, assignment)
			case err := <-errors:
				assert.Error(t, err)
				errorCount++
			}
		}
		
		// Should have some successful assignments
		assert.Greater(t, len(assignments), 0)
		assert.GreaterOrEqual(t, len(assignments)+errorCount, numJobs)
		
		// Verify assignments are distributed
		workerAssignments := make(map[string]int)
		for _, assignment := range assignments {
			workerAssignments[assignment.WorkerID]++
		}
		
		assert.Greater(t, len(workerAssignments), 0)
	})
}

func TestDistributedManager_WorkerTags(t *testing.T) {
	config := ManagerConfig{
		LoadBalanceStrategy: "tag_based",
	}
	
	manager := NewDistributedManager(config)
	
	// Register workers with different tags
	workers := []*WorkerNode{
		{ID: "worker-gpu", Status: "active", Tags: map[string]string{"type": "gpu", "region": "us-west"}, MaxJobs: 10},
		{
			ID:   "worker-cpu", Status: "active", Tags: map[string]string{"type": "cpu", "region": "us-east"}, MaxJobs: 10,
		},
		{
			ID: "worker-gpu-east", Status: "active", Tags: map[string]string{"type": "gpu", "region": "us-east"}, MaxJobs: 10,
		},
	}
	
	for _, worker := range workers {
		manager.RegisterWorker(worker)
	}
	
	// Test 1: Find workers by tag
	t.Run("FindWorkersByTag", func(t *testing.T) {
		gpuWorkers := manager.FindWorkersByTag("type", "gpu")
		assert.Len(t, gpuWorkers, 2)
		
		for _, worker := range gpuWorkers {
			assert.Equal(t, "gpu", worker.Tags["type"])
		}
		
		cpuWorkers := manager.FindWorkersByTag("type", "cpu")
		assert.Len(t, cpuWorkers, 1)
		assert.Equal(t, "worker-cpu", cpuWorkers[0].ID)
		
		eastWorkers := manager.FindWorkersByTag("region", "us-east")
		assert.Len(t, eastWorkers, 2)
	})
	
	// Test 2: Assign job with tag requirements
	t.Run("AssignJobWithTagRequirements", func(t *testing.T) {
		assignment, err := manager.AssignJobWithTags("gpu-job", map[string]string{"type": "gpu"})
		require.NoError(t, err)
		
		// Should assign to one of the GPU workers
		assert.Contains(t, []string{"worker-gpu", "worker-gpu-east"}, assignment.WorkerID)
		
		// Assign job with multiple tag requirements
		assignment, err = manager.AssignJobWithTags("gpu-east-job", map[string]string{
			"type":   "gpu",
			"region": "us-east",
		})
		require.NoError(t, err)
		assert.Equal(t, "worker-gpu-east", assignment.WorkerID)
	})
	
	// Test 3: No matching workers for tag requirements
	t.Run("NoMatchingWorkersForTags", func(t *testing.T) {
		_, err := manager.AssignJobWithTags("quantum-job", map[string]string{"type": "quantum"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no workers match tag requirements")
	})
}

// Mock implementations
func NewDistributedManager(config ManagerConfig) *DistributedManager {
	return &DistributedManager{
		config:    config,
		workers:   make(map[string]*WorkerNode),
		running:   false,
		startTime: time.Now(),
		index:     0,
	}
}

func (dm *DistributedManager) RegisterWorker(worker *WorkerNode) error {
	if _, exists := dm.workers[worker.ID]; exists {
		return fmt.Errorf("worker already exists")
	}
	
	worker.LastSeen = time.Now()
	dm.workers[worker.ID] = worker
	return nil
}

func (dm *DistributedManager) UnregisterWorker(workerID string) error {
	if _, exists := dm.workers[workerID]; !exists {
		return fmt.Errorf("worker not found")
	}
	
	delete(dm.workers, workerID)
	return nil
}

func (dm *DistributedManager) GetWorkers() map[string]*WorkerNode {
	// Return direct pointers to allow modification in tests
	return dm.workers
}

func (dm *DistributedManager) AssignJob(jobID string) (*JobAssignment, error) {
	return dm.assignJobWithStrategy(jobID, dm.config.LoadBalanceStrategy)
}

func (dm *DistributedManager) AssignJobToWorker(jobID, workerID string) (*JobAssignment, error) {
	worker, exists := dm.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker not found")
	}
	
	if worker.Status != "active" {
		return nil, fmt.Errorf("worker not active")
	}
	
	if worker.ActiveJobs >= worker.MaxJobs {
		return nil, fmt.Errorf("worker at capacity")
	}
	
	// Update worker job count
	worker.ActiveJobs++
	worker.Load = float64(worker.ActiveJobs) / float64(worker.MaxJobs)
	
	return &JobAssignment{
		JobID:     jobID,
		WorkerID:  workerID,
		Status:    "assigned",
		StartTime: time.Now(),
	}, nil
}

func (dm *DistributedManager) AssignJobWithTags(jobID string, tags map[string]string) (*JobAssignment, error) {
	// Find workers matching tags
	candidates := dm.findWorkersMatchingTags(tags)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no workers match tag requirements")
	}
	
	// Use least loaded strategy among candidates
	var bestWorker *WorkerNode
	for _, worker := range candidates {
		if bestWorker == nil || worker.Load < bestWorker.Load {
			bestWorker = worker
		}
	}
	
	return dm.AssignJobToWorker(jobID, bestWorker.ID)
}

func (dm *DistributedManager) FindWorkersByTag(key, value string) []*WorkerNode {
	var matchingWorkers []*WorkerNode
	
	for _, worker := range dm.workers {
		if worker.Tags[key] == value {
			matchingWorkers = append(matchingWorkers, worker)
		}
	}
	
	return matchingWorkers
}

func (dm *DistributedManager) PerformHealthCheck() error {
	now := time.Now()
	timeout := 5 * time.Minute
	
	// Use override if set (for testing)
	if dm.timeoutOverride > 0 {
		timeout = dm.timeoutOverride
	}
	
	for _, worker := range dm.workers {
		if now.Sub(worker.LastSeen) > timeout {
			worker.Status = "inactive"
		} else if worker.Status == "inactive" {
			worker.Status = "active"
		}
	}
	
	return nil
}

func (dm *DistributedManager) assignJobWithStrategy(jobID, strategy string) (*JobAssignment, error) {
	activeWorkers := make([]*WorkerNode, 0)
	for _, worker := range dm.workers {
		if worker.Status == "active" && worker.ActiveJobs < worker.MaxJobs {
			activeWorkers = append(activeWorkers, worker)
		}
	}
	
	if len(activeWorkers) == 0 {
		return nil, fmt.Errorf("no available workers")
	}
	
	var selectedWorker *WorkerNode
	
	switch strategy {
	case "round_robin":
		// Simple round-robin implementation
		selectedWorker = activeWorkers[dm.index%len(activeWorkers)]
		dm.index++
	case "least_loaded":
		selectedWorker = activeWorkers[0]
		for _, worker := range activeWorkers[1:] {
			if worker.Load < selectedWorker.Load {
				selectedWorker = worker
			}
		}
	case "random":
		index := time.Now().Nanosecond() % len(activeWorkers)
		selectedWorker = activeWorkers[index]
	default:
		// Default to least loaded
		selectedWorker = activeWorkers[0]
		for _, worker := range activeWorkers[1:] {
			if worker.Load < selectedWorker.Load {
				selectedWorker = worker
			}
		}
	}
	
	return dm.AssignJobToWorker(jobID, selectedWorker.ID)
}

func (dm *DistributedManager) findWorkersMatchingTags(requiredTags map[string]string) []*WorkerNode {
	var matchingWorkers []*WorkerNode
	
	for _, worker := range dm.workers {
		if worker.Status != "active" {
			continue
		}
		
		matches := true
		for key, value := range requiredTags {
			if worker.Tags[key] != value {
				matches = false
				break
			}
		}
		
		if matches {
			matchingWorkers = append(matchingWorkers, worker)
		}
	}
	
	return matchingWorkers
}