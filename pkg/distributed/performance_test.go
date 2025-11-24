package distributed

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/sshworker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Task represents a translation task for testing
type Task struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	WorkerID  string                 `json:"worker_id"`
	Success   bool                   `json:"success"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WorkerInfo represents worker information
type WorkerInfo struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Username string `json:"username"`
}

// PerformanceMetrics collects performance metrics for testing
type PerformanceMetrics struct {
	StartTime         time.Time
	EndTime           time.Time
	OperationCount    int64
	SuccessCount      int64
	ErrorCount        int64
	TotalDuration     time.Duration
	AverageLatency    time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	P50Latency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	MemoryUsageMB     float64
	CPUUsagePercent   float64
	ThroughputPerSec  float64
	ErrorRatePercent  float64
	mu                sync.RWMutex
	latencies         []time.Duration
}

// NewPerformanceMetrics creates a new performance metrics collector
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		MinLatency: time.Hour, // Initialize to very high value
		latencies:  make([]time.Duration, 0),
	}
}

// StartMeasurement begins performance measurement
func (pm *PerformanceMetrics) StartMeasurement() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.StartTime = time.Now()
	runtime.GC() // Clear GC before measurement
}

// RecordOperation records a single operation's latency and success status
func (pm *PerformanceMetrics) RecordOperation(latency time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.OperationCount++
	pm.latencies = append(pm.latencies, latency)
	
	if success {
		pm.SuccessCount++
	} else {
		pm.ErrorCount++
	}
	
	if latency < pm.MinLatency {
		pm.MinLatency = latency
	}
	if latency > pm.MaxLatency {
		pm.MaxLatency = latency
	}
}

// StopMeasurement ends performance measurement and calculates final metrics
func (pm *PerformanceMetrics) StopMeasurement() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.EndTime = time.Now()
	pm.TotalDuration = pm.EndTime.Sub(pm.StartTime)
	
	if pm.OperationCount > 0 {
		pm.AverageLatency = pm.TotalDuration / time.Duration(pm.OperationCount)
		pm.ErrorRatePercent = float64(pm.ErrorCount) / float64(pm.OperationCount) * 100
		pm.ThroughputPerSec = float64(pm.OperationCount) / pm.TotalDuration.Seconds()
		
		// Calculate percentiles
		if len(pm.latencies) > 0 {
			// Simple percentile calculation
			sorted := make([]time.Duration, len(pm.latencies))
			copy(sorted, pm.latencies)
			
			// Sort latencies (in-place, would use proper sort in production)
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i] > sorted[j] {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
			
			pm.P50Latency = sorted[len(sorted)*50/100]
			pm.P95Latency = sorted[len(sorted)*95/100]
			pm.P99Latency = sorted[len(sorted)*99/100]
		}
	}
	
	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	pm.MemoryUsageMB = float64(m.Alloc) / 1024 / 1024
}

// GetSnapshot returns a snapshot of current metrics
func (pm *PerformanceMetrics) GetSnapshot() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return map[string]interface{}{
		"start_time":          pm.StartTime,
		"end_time":            pm.EndTime,
		"operation_count":     pm.OperationCount,
		"success_count":       pm.SuccessCount,
		"error_count":         pm.ErrorCount,
		"total_duration_ms":   pm.TotalDuration.Milliseconds(),
		"average_latency_ms":  pm.AverageLatency.Milliseconds(),
		"min_latency_ms":      pm.MinLatency.Milliseconds(),
		"max_latency_ms":      pm.MaxLatency.Milliseconds(),
		"p50_latency_ms":      pm.P50Latency.Milliseconds(),
		"p95_latency_ms":      pm.P95Latency.Milliseconds(),
		"p99_latency_ms":      pm.P99Latency.Milliseconds(),
		"memory_usage_mb":     pm.MemoryUsageMB,
		"cpu_usage_percent":   pm.CPUUsagePercent,
		"throughput_per_sec":  pm.ThroughputPerSec,
		"error_rate_percent":  pm.ErrorRatePercent,
	}
}

// PerformanceTestConfig defines configuration for performance tests
type PerformanceTestConfig struct {
	ConcurrentWorkers int
	Duration          time.Duration
	TasksPerWorker    int
	TaskComplexity    string // "simple", "medium", "complex"
	ExpectedLatency   time.Duration
	ExpectedThroughput float64
	MaxErrorRate      float64
	MaxMemoryUsageMB  float64
}

// DefaultPerformanceTestConfig returns a default configuration
func DefaultPerformanceTestConfig() PerformanceTestConfig {
	return PerformanceTestConfig{
		ConcurrentWorkers:    10,
		Duration:             30 * time.Second,
		TasksPerWorker:       100,
		TaskComplexity:        "medium",
		ExpectedLatency:       100 * time.Millisecond,
		ExpectedThroughput:    100.0,
		MaxErrorRate:          1.0, // 1%
		MaxMemoryUsageMB:      100.0,
	}
}

// MockDistributedSystem simulates a distributed system for performance testing
type MockDistributedSystem struct {
	workers      []string
	taskDelay    time.Duration
	errorRate    float64
	mu           sync.RWMutex
	requestCount int64
	errorCount   int64
}

func NewMockDistributedSystem(workerCount int, taskDelay time.Duration, errorRate float64) *MockDistributedSystem {
	workers := make([]string, workerCount)
	for i := 0; i < workerCount; i++ {
		workers[i] = fmt.Sprintf("worker-%d", i)
	}
	
	return &MockDistributedSystem{
		workers:   workers,
		taskDelay: taskDelay,
		errorRate: errorRate,
	}
}

func (m *MockDistributedSystem) ExecuteTask(ctx context.Context, task Task) (*TaskResult, error) {
	m.mu.Lock()
	m.requestCount++
	m.mu.Unlock()
	
	// Simulate task processing time
	select {
	case <-time.After(m.taskDelay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// Simulate errors based on error rate
	if m.errorRate > 0 {
		// Simple random error simulation
		if m.requestCount%int(1.0/m.errorRate*100) == 0 {
			m.mu.Lock()
			m.errorCount++
			m.mu.Unlock()
			return nil, fmt.Errorf("simulated error for task %s", task.ID)
		}
	}
	
	result := &worker.TaskResult{
		TaskID:    task.ID,
		WorkerID:  m.workers[m.requestCount%len(m.workers)],
		Success:   true,
		Duration:  m.taskDelay,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"processed_text": fmt.Sprintf("Processed: %v", task.Data),
		},
	}
	
	return result, nil
}

func (m *MockDistributedSystem) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"request_count": m.requestCount,
		"error_count":   m.errorCount,
		"worker_count":  len(m.workers),
		"error_rate":    float64(m.errorCount) / float64(m.requestCount) * 100,
	}
}

// TestPerformanceBasicLoad tests basic load performance
func TestPerformanceBasicLoad(t *testing.T) {
	config := DefaultPerformanceTestConfig()
	system := NewMockDistributedSystem(config.ConcurrentWorkers, 10*time.Millisecond, 0.01)
	
	metrics := NewPerformanceMetrics()
	metrics.StartMeasurement()
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	for i := 0; i < config.ConcurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < config.TasksPerWorker; j++ {
				start := time.Now()
				
				task := Task{
					ID:   fmt.Sprintf("task-%d-%d", workerID, j),
					Type: "translation",
					Data: map[string]interface{}{
						"text": fmt.Sprintf("Test text %d", j),
					},
				}
				
				_, err := system.ExecuteTask(ctx, task)
				latency := time.Since(start)
				
				metrics.RecordOperation(latency, err == nil)
				
				if err != nil {
					t.Logf("Task failed: %v", err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	metrics.StopMeasurement()
	
	snapshot := metrics.GetSnapshot()
	
	// Performance assertions
	assert.Less(t, snapshot["average_latency_ms"], config.ExpectedLatency.Milliseconds())
	assert.GreaterOrEqual(t, snapshot["throughput_per_sec"], config.ExpectedThroughput)
	assert.LessOrEqual(t, snapshot["error_rate_percent"], config.MaxErrorRate)
	assert.LessOrEqual(t, snapshot["memory_usage_mb"], config.MaxMemoryUsageMB)
	
	t.Logf("Performance Results:")
	t.Logf("  Total Operations: %d", snapshot["operation_count"])
	t.Logf("  Average Latency: %.2f ms", snapshot["average_latency_ms"])
	t.Logf("  P95 Latency: %.2f ms", snapshot["p95_latency_ms"])
	t.Logf("  Throughput: %.2f ops/sec", snapshot["throughput_per_sec"])
	t.Logf("  Error Rate: %.2f%%", snapshot["error_rate_percent"])
	t.Logf("  Memory Usage: %.2f MB", snapshot["memory_usage_mb"])
}

// TestPerformanceUnderLoad tests performance under high load
func TestPerformanceUnderLoad(t *testing.T) {
	config := PerformanceTestConfig{
		ConcurrentWorkers:    50,
		Duration:             60 * time.Second,
		TasksPerWorker:       200,
		TaskComplexity:        "complex",
		ExpectedLatency:       500 * time.Millisecond,
		ExpectedThroughput:    50.0,
		MaxErrorRate:          2.0,
		MaxMemoryUsageMB:      500.0,
	}
	
	system := NewMockDistributedSystem(config.ConcurrentWorkers, 50*time.Millisecond, 0.02)
	
	metrics := NewPerformanceMetrics()
	metrics.StartMeasurement()
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	for i := 0; i < config.ConcurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < config.TasksPerWorker; j++ {
				start := time.Now()
				
				task := Task{
					ID:   fmt.Sprintf("load-task-%d-%d", workerID, j),
					Type: "complex_translation",
					Data: map[string]interface{}{
						"text": fmt.Sprintf("Complex test text %d with additional data", j),
						"metadata": map[string]interface{}{
							"priority": "high",
							"timeout":  30,
						},
					},
				}
				
				_, err := system.ExecuteTask(ctx, task)
				latency := time.Since(start)
				
				metrics.RecordOperation(latency, err == nil)
			}
		}(i)
	}
	
	wg.Wait()
	metrics.StopMeasurement()
	
	snapshot := metrics.GetSnapshot()
	
	// Performance assertions under load
	assert.Less(t, snapshot["average_latency_ms"], config.ExpectedLatency.Milliseconds())
	assert.GreaterOrEqual(t, snapshot["throughput_per_sec"], config.ExpectedThroughput)
	assert.LessOrEqual(t, snapshot["error_rate_percent"], config.MaxErrorRate)
	assert.LessOrEqual(t, snapshot["memory_usage_mb"], config.MaxMemoryUsageMB)
	
	t.Logf("High Load Performance Results:")
	t.Logf("  Total Operations: %d", snapshot["operation_count"])
	t.Logf("  Average Latency: %.2f ms", snapshot["average_latency_ms"])
	t.Logf("  P95 Latency: %.2f ms", snapshot["p95_latency_ms"])
	t.Logf("  P99 Latency: %.2f ms", snapshot["p99_latency_ms"])
	t.Logf("  Throughput: %.2f ops/sec", snapshot["throughput_per_sec"])
	t.Logf("  Error Rate: %.2f%%", snapshot["error_rate_percent"])
	t.Logf("  Memory Usage: %.2f MB", snapshot["memory_usage_mb"])
}

// TestPerformanceScalability tests how performance scales with worker count
func TestPerformanceScalability(t *testing.T) {
	workerCounts := []int{1, 5, 10, 20, 50}
	scalabilityResults := make(map[int]map[string]interface{})
	
	for _, workerCount := range workerCounts {
		config := PerformanceTestConfig{
			ConcurrentWorkers:    workerCount,
			Duration:             10 * time.Second,
			TasksPerWorker:       100,
			TaskComplexity:        "simple",
			ExpectedLatency:       100 * time.Millisecond,
			ExpectedThroughput:    float64(workerCount) * 10.0,
			MaxErrorRate:          0.5,
			MaxMemoryUsageMB:      float64(workerCount) * 10.0,
		}
		
		system := NewMockDistributedSystem(workerCount, 5*time.Millisecond, 0.005)
		
		metrics := NewPerformanceMetrics()
		metrics.StartMeasurement()
		
		ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
		defer cancel()
		
		var wg sync.WaitGroup
		
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				
				for j := 0; j < config.TasksPerWorker; j++ {
					start := time.Now()
					
					task := worker.Task{
						ID:   fmt.Sprintf("scale-task-%d-%d", workerID, j),
						Type: "translation",
						Data: map[string]interface{}{
							"text": fmt.Sprintf("Scale test text %d", j),
						},
					}
					
					_, err := system.ExecuteTask(ctx, task)
					latency := time.Since(start)
					
					metrics.RecordOperation(latency, err == nil)
				}
			}(i)
		}
		
		wg.Wait()
		metrics.StopMeasurement()
		
		snapshot := metrics.GetSnapshot()
		scalabilityResults[workerCount] = snapshot
		
		t.Logf("Scalability Results for %d workers:", workerCount)
		t.Logf("  Throughput: %.2f ops/sec", snapshot["throughput_per_sec"])
		t.Logf("  Average Latency: %.2f ms", snapshot["average_latency_ms"])
		t.Logf("  Memory Usage: %.2f MB", snapshot["memory_usage_mb"])
		
		// Basic performance assertions
		assert.Greater(t, snapshot["throughput_per_sec"], 0.0)
		assert.Less(t, snapshot["error_rate_percent"], config.MaxErrorRate)
	}
	
	// Test scalability: throughput should increase with worker count
	// (within reason - allow for some overhead)
	for i := 1; i < len(workerCounts); i++ {
		prevWorkers := workerCounts[i-1]
		currWorkers := workerCounts[i]
		
		prevThroughput := scalabilityResults[prevWorkers]["throughput_per_sec"].(float64)
		currThroughput := scalabilityResults[currWorkers]["throughput_per_sec"].(float64)
		
		// Throughput should at least increase somewhat
		scalabilityRatio := currThroughput / prevThroughput
		expectedMinRatio := float64(currWorkers) / float64(prevWorkers) * 0.5 // Allow 50% efficiency
		
		assert.GreaterOrEqual(t, scalabilityRatio, expectedMinRatio,
			"Scalability test failed: %d workers (%.2f ops/sec) vs %d workers (%.2f ops/sec)",
			prevWorkers, prevThroughput, currWorkers, currThroughput)
	}
}

// TestPerformanceUnderStress tests system behavior under extreme stress
func TestPerformanceUnderStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	config := PerformanceTestConfig{
		ConcurrentWorkers:    100,
		Duration:             120 * time.Second,
		TasksPerWorker:       500,
		TaskComplexity:        "complex",
		ExpectedLatency:       1000 * time.Millisecond,
		ExpectedThroughput:    25.0,
		MaxErrorRate:          5.0, // Allow higher error rate under stress
		MaxMemoryUsageMB:      1000.0,
	}
	
	system := NewMockDistributedSystem(config.ConcurrentWorkers, 100*time.Millisecond, 0.05)
	
	metrics := NewPerformanceMetrics()
	metrics.StartMeasurement()
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()
	
	var wg sync.WaitGroup
	
	for i := 0; i < config.ConcurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < config.TasksPerWorker; j++ {
				start := time.Now()
				
				task := Task{
					ID:   fmt.Sprintf("stress-task-%d-%d", workerID, j),
					Type: "stress_test",
					Data: map[string]interface{}{
						"text": fmt.Sprintf("Stress test text %d with lots of additional data to increase complexity", j),
						"metadata": map[string]interface{}{
							"priority":     "urgent",
							"timeout":      60,
							"retry_count":  3,
							"complex_data": generateComplexData(100),
						},
					},
				}
				
				_, err := system.ExecuteTask(ctx, task)
				latency := time.Since(start)
				
				metrics.RecordOperation(latency, err == nil)
				
				// Add some random delay to simulate real-world conditions
				time.Sleep(time.Millisecond * time.Duration(j%10))
			}
		}(i)
	}
	
	wg.Wait()
	metrics.StopMeasurement()
	
	snapshot := metrics.GetSnapshot()
	systemStats := system.GetStats()
	
	// Stress test assertions - more lenient thresholds
	assert.Less(t, snapshot["average_latency_ms"], config.ExpectedLatency.Milliseconds())
	assert.GreaterOrEqual(t, snapshot["throughput_per_sec"], config.ExpectedThroughput)
	assert.LessOrEqual(t, snapshot["error_rate_percent"], config.MaxErrorRate)
	assert.LessOrEqual(t, snapshot["memory_usage_mb"], config.MaxMemoryUsageMB)
	
	t.Logf("Stress Test Results:")
	t.Logf("  Total Operations: %d", snapshot["operation_count"])
	t.Logf("  System Requests: %d", systemStats["request_count"])
	t.Logf("  System Errors: %d", systemStats["error_count"])
	t.Logf("  Average Latency: %.2f ms", snapshot["average_latency_ms"])
	t.Logf("  P95 Latency: %.2f ms", snapshot["p95_latency_ms"])
	t.Logf("  P99 Latency: %.2f ms", snapshot["p99_latency_ms"])
	t.Logf("  Throughput: %.2f ops/sec", snapshot["throughput_per_sec"])
	t.Logf("  Error Rate: %.2f%%", snapshot["error_rate_percent"])
	t.Logf("  Memory Usage: %.2f MB", snapshot["memory_usage_mb"])
}

// Helper function to generate complex data for stress testing
func generateComplexData(size int) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d_with_additional_content_to_increase_size", i)
		data[key] = value
	}
	return data
}

// BenchmarkPerformanceSingleOperation benchmarks single operation performance
func BenchmarkPerformanceSingleOperation(b *testing.B) {
	system := NewMockDistributedSystem(10, 1*time.Millisecond, 0.001)
	ctx := context.Background()
	
	task := Task{
		ID:   "benchmark-task",
		Type: "translation",
		Data: map[string]interface{}{
			"text": "Benchmark test text",
		},
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := system.ExecuteTask(ctx, task)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPerformanceConcurrentOperations benchmarks concurrent operation performance
func BenchmarkPerformanceConcurrentOperations(b *testing.B) {
	system := NewMockDistributedSystem(50, 5*time.Millisecond, 0.01)
	ctx := context.Background()
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			task := Task{
				ID:   fmt.Sprintf("concurrent-task-%d", i),
				Type: "translation",
				Data: map[string]interface{}{
					"text": fmt.Sprintf("Concurrent benchmark test text %d", i),
				},
			}
			
			_, err := system.ExecuteTask(ctx, task)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}