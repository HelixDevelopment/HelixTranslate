package deployment

import (
	"fmt"
	"testing"
	"time"
)

func TestDeploymentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *DeploymentRequest
		wantErr bool
		errType string
	}{
		{
			name: "valid request",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
				Timeout:     30 * time.Minute,
				Retries:     3,
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing id",
			req: &DeploymentRequest{
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
				CreatedAt:   time.Now(),
			},
			wantErr: true,
			errType: "id",
		},
		{
			name: "missing target host",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
				CreatedAt:   time.Now(),
			},
			wantErr: true,
			errType: "target_host",
		},
		{
			name: "missing package path",
			req: &DeploymentRequest{
				ID:         "test-deploy-001",
				TargetHost: "worker1.example.com",
				Command:    "./worker start",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errType: "package_path",
		},
		{
			name: "missing command",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				CreatedAt:   time.Now(),
			},
			wantErr: true,
			errType: "command",
		},
		{
			name: "zero timeout uses default",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
				Timeout:     0,
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "zero retries uses default",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
				Retries:     0,
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "zero created_at uses now",
			req: &DeploymentRequest{
				ID:          "test-deploy-001",
				TargetHost:  "worker1.example.com",
				PackagePath: "/tmp/translator-worker.tar.gz",
				Command:     "./worker start",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.errType, ve.Field)
			} else {
				require.NoError(t, err)
				// Check defaults are set
				if tt.req.Timeout == 0 {
					assert.Equal(t, 30*time.Minute, tt.req.Timeout)
				}
				if tt.req.Retries == 0 {
					assert.Equal(t, 3, tt.req.Retries)
				}
				if tt.req.CreatedAt.IsZero() {
					assert.False(t, tt.req.CreatedAt.IsZero())
				}
			}
		})
	}
}

func TestBatchDeploymentRequest_Validate(t *testing.T) {
	validSubReq := &DeploymentRequest{
		ID:          "sub-deploy-001",
		TargetHost:  "worker1.example.com",
		PackagePath: "/tmp/worker.tar.gz",
		Command:     "./worker start",
		CreatedAt:   time.Now(),
	}

	tests := []struct {
		name    string
		req     *BatchDeploymentRequest
		wantErr bool
		errType string
	}{
		{
			name: "valid batch request",
			req: &BatchDeploymentRequest{
				ID:          "batch-001",
				Name:        "Deploy workers",
				Requests:    []DeploymentRequest{*validSubReq},
				Parallelism: 1,
				Timeout:     2 * time.Hour,
				RollbackMode: "any",
				CreatedAt:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing id",
			req: &BatchDeploymentRequest{
				Name:     "Deploy workers",
				Requests: []DeploymentRequest{*validSubReq},
			},
			wantErr: true,
			errType: "id",
		},
		{
			name: "missing name",
			req: &BatchDeploymentRequest{
				ID:       "batch-001",
				Requests: []DeploymentRequest{*validSubReq},
			},
			wantErr: true,
			errType: "name",
		},
		{
			name: "empty requests",
			req: &BatchDeploymentRequest{
				ID:   "batch-001",
				Name: "Deploy workers",
			},
			wantErr: true,
			errType: "requests",
		},
		{
			name: "zero parallelism uses default",
			req: &BatchDeploymentRequest{
				ID:       "batch-001",
				Name:     "Deploy workers",
				Requests: []DeploymentRequest{*validSubReq},
				Parallelism: 0,
			},
			wantErr: false,
		},
		{
			name: "zero timeout uses default",
			req: &BatchDeploymentRequest{
				ID:       "batch-001",
				Name:     "Deploy workers",
				Requests: []DeploymentRequest{*validSubReq},
				Timeout:  0,
			},
			wantErr: false,
		},
		{
			name: "zero created_at uses now",
			req: &BatchDeploymentRequest{
				ID:       "batch-001",
				Name:     "Deploy workers",
				Requests: []DeploymentRequest{*validSubReq},
			},
			wantErr: false,
		},
		{
			name: "invalid sub-request",
			req: &BatchDeploymentRequest{
				ID:       "batch-001",
				Name:     "Deploy workers",
				Requests: []DeploymentRequest{
					{
						ID: "", // Invalid sub-request
						TargetHost:  "worker1.example.com",
						PackagePath: "/tmp/worker.tar.gz",
						Command:     "./worker start",
						CreatedAt:   time.Now(),
					},
				},
			},
			wantErr: true,
			errType: "requests[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.errType, ve.Field)
			} else {
				require.NoError(t, err)
				// Check defaults
				if tt.req.Parallelism == 0 {
					assert.Equal(t, len(tt.req.Requests), tt.req.Parallelism)
				}
				if tt.req.Timeout == 0 {
					assert.Equal(t, 2*time.Hour, tt.req.Timeout)
				}
				if tt.req.CreatedAt.IsZero() {
					assert.False(t, tt.req.CreatedAt.IsZero())
				}
			}
		})
	}
}

func TestWorkerStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  *WorkerStatus
		wantErr bool
		errType string
	}{
		{
			name: "valid worker status",
			status: &WorkerStatus{
				ID:            "worker-001",
				Host:          "worker1.example.com",
				Status:        "idle",
				LastActivity:  time.Now(),
				JobsCompleted: 10,
				JobsFailed:    2,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			status: &WorkerStatus{
				Host:         "worker1.example.com",
				Status:       "idle",
				LastActivity: time.Now(),
			},
			wantErr: true,
			errType: "id",
		},
		{
			name: "missing host",
			status: &WorkerStatus{
				ID:           "worker-001",
				Status:       "idle",
				LastActivity: time.Now(),
			},
			wantErr: true,
			errType: "host",
		},
		{
			name: "empty status uses default",
			status: &WorkerStatus{
				ID:           "worker-001",
				Host:         "worker1.example.com",
				LastActivity: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "zero last_activity uses now",
			status: &WorkerStatus{
				ID:     "worker-001",
				Host:   "worker1.example.com",
				Status: "idle",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.errType, ve.Field)
			} else {
				require.NoError(t, err)
				// Check defaults
				if tt.status.Status == "" {
					assert.Equal(t, "unknown", tt.status.Status)
				}
				if tt.status.LastActivity.IsZero() {
					assert.False(t, tt.status.LastActivity.IsZero())
				}
			}
		})
	}
}

func TestDeploymentResponse_HelperMethods(t *testing.T) {
	tests := []struct {
		name     string
		response *DeploymentResponse
		success  bool
		completed bool
	}{
		{
			name: "successful response",
			response: &DeploymentResponse{
				Status:  "completed",
				Success: true,
				Error:   "",
			},
			success:  true,
			completed: true,
		},
		{
			name: "failed response",
			response: &DeploymentResponse{
				Status:  "failed",
				Success: false,
				Error:   "Command failed",
			},
			success:  false,
			completed: true,
		},
		{
			name: "running response",
			response: &DeploymentResponse{
				Status:  "running",
				Success: true,
				Error:   "",
			},
			success:  true,
			completed: false,
		},
		{
			name: "successful but with error",
			response: &DeploymentResponse{
				Status:  "completed",
				Success: true,
				Error:   "Warning message",
			},
			success:  false,
			completed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.success, tt.response.IsSuccessful(), "IsSuccessful mismatch")
			assert.Equal(t, tt.completed, tt.response.IsCompleted(), "IsCompleted mismatch")
		})
	}
}

func TestBatchDeploymentResponse_HelperMethods(t *testing.T) {
	tests := []struct {
		name     string
		response *BatchDeploymentResponse
		complete bool
		success  bool
	}{
		{
			name: "complete successful",
			response: &BatchDeploymentResponse{
				TotalJobs:    5,
				CompletedJobs: 5,
				FailedJobs:    0,
				SuccessJobs:   5,
			},
			complete: true,
			success:  true,
		},
		{
			name: "complete with failures",
			response: &BatchDeploymentResponse{
				TotalJobs:    5,
				CompletedJobs: 5,
				FailedJobs:    2,
				SuccessJobs:   3,
			},
			complete: true,
			success:  false,
		},
		{
			name: "incomplete",
			response: &BatchDeploymentResponse{
				TotalJobs:    5,
				CompletedJobs: 3,
				FailedJobs:    1,
				SuccessJobs:   2,
			},
			complete: false,
			success:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.complete, tt.response.IsComplete(), "IsComplete mismatch")
			assert.Equal(t, tt.success, tt.response.IsSuccessful(), "IsSuccessful mismatch")
		})
	}
}

func TestSystemResources_IsHealthy(t *testing.T) {
	tests := []struct {
		name      string
		resources *SystemResources
		healthy   bool
	}{
		{
			name: "all healthy metrics",
			resources: &SystemResources{
				CPUUsage:    50.0,
				MemoryUsage: 60.0,
				DiskUsage:   70.0,
			},
			healthy: true,
		},
		{
			name: "CPU usage too high",
			resources: &SystemResources{
				CPUUsage:    85.0,
				MemoryUsage: 60.0,
				DiskUsage:   70.0,
			},
			healthy: false,
		},
		{
			name: "memory usage too high",
			resources: &SystemResources{
				CPUUsage:    50.0,
				MemoryUsage: 90.0,
				DiskUsage:   70.0,
			},
			healthy: false,
		},
		{
			name: "disk usage too high",
			resources: &SystemResources{
				CPUUsage:    50.0,
				MemoryUsage: 60.0,
				DiskUsage:   95.0,
			},
			healthy: false,
		},
		{
			name: "all metrics at threshold",
			resources: &SystemResources{
				CPUUsage:    79.9,
				MemoryUsage: 79.9,
				DiskUsage:   79.9,
			},
			healthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.healthy, tt.resources.IsHealthy())
		})
	}
}

func TestWorkerCapabilities_SupportMethods(t *testing.T) {
	caps := &WorkerCapabilities{
		SupportedPlatforms: []string{"linux", "darwin", "windows"},
		SupportedFormats:   []string{"fb2", "epub", "pdf", "docx"},
		MaxConcurrentJobs:  4,
		HasGPU:            true,
		GPUType:          "nvidia-tesla-v100",
		NetworkSpeed:      1000, // 1 Gbps
	}

	t.Run("SupportsPlatform", func(t *testing.T) {
		assert.True(t, caps.SupportsPlatform("linux"))
		assert.True(t, caps.SupportsPlatform("darwin"))
		assert.True(t, caps.SupportsPlatform("windows"))
		assert.False(t, caps.SupportsPlatform("freebsd"))
		assert.False(t, caps.SupportsPlatform(""))
	})

	t.Run("SupportsFormat", func(t *testing.T) {
		assert.True(t, caps.SupportsFormat("fb2"))
		assert.True(t, caps.SupportsFormat("epub"))
		assert.True(t, caps.SupportsFormat("pdf"))
		assert.True(t, caps.SupportsFormat("docx"))
		assert.False(t, caps.SupportsFormat("txt"))
		assert.False(t, caps.SupportsFormat(""))
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("NewDeploymentRequest", func(t *testing.T) {
		req := NewDeploymentRequest("test-001", "host.com", "/path/pkg.tar.gz", "./start")
		
		assert.Equal(t, "test-001", req.ID)
		assert.Equal(t, "host.com", req.TargetHost)
		assert.Equal(t, "/path/pkg.tar.gz", req.PackagePath)
		assert.Equal(t, "./start", req.Command)
		assert.Equal(t, 30*time.Minute, req.Timeout)
		assert.Equal(t, 3, req.Retries)
		assert.NotNil(t, req.Environment)
		assert.NotNil(t, req.Labels)
		assert.Equal(t, 0, len(req.PreCommands))
		assert.Equal(t, 0, len(req.PostCommands))
	})

	t.Run("NewWorkerStatus", func(t *testing.T) {
		status := NewWorkerStatus("worker-001", "worker1.example.com")
		
		assert.Equal(t, "worker-001", status.ID)
		assert.Equal(t, "worker1.example.com", status.Host)
		assert.Equal(t, "idle", status.Status)
		assert.Equal(t, 0, status.JobsCompleted)
		assert.Equal(t, 0, status.JobsFailed)
	})

	t.Run("NewBatchDeploymentRequest", func(t *testing.T) {
		subReq := NewDeploymentRequest("sub-001", "host.com", "/path/pkg.tar.gz", "./start")
		batchReq := NewBatchDeploymentRequest("batch-001", "Test Batch", []DeploymentRequest{*subReq})
		
		assert.Equal(t, "batch-001", batchReq.ID)
		assert.Equal(t, "Test Batch", batchReq.Name)
		assert.Equal(t, 1, len(batchReq.Requests))
		assert.Equal(t, 1, batchReq.Parallelism)
		assert.Equal(t, "any", batchReq.RollbackMode)
		assert.Equal(t, 2*time.Hour, batchReq.Timeout)
	})
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}
	
	expected := "test message (field: test_field)"
	assert.Equal(t, expected, err.Error())
}

// Performance benchmarks
func BenchmarkDeploymentRequest_Validate(b *testing.B) {
	req := &DeploymentRequest{
		ID:          "test-deploy-001",
		TargetHost:  "worker1.example.com",
		PackagePath: "/tmp/translator-worker.tar.gz",
		Command:     "./worker start",
		Timeout:     30 * time.Minute,
		Retries:     3,
		CreatedAt:   time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

func BenchmarkBatchDeploymentRequest_Validate(b *testing.B) {
	subReq := &DeploymentRequest{
		ID:          "sub-deploy-001",
		TargetHost:  "worker1.example.com",
		PackagePath: "/tmp/translator-worker.tar.gz",
		Command:     "./worker start",
		CreatedAt:   time.Now(),
	}
	
	req := &BatchDeploymentRequest{
		ID:       "batch-001",
		Name:     "Deploy workers",
		Requests: []DeploymentRequest{*subReq, *subReq, *subReq},
		CreatedAt: time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = req.Validate()
	}
}

// Concurrent testing
func TestDeploymentTypes_Concurrent(t *testing.T) {
	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Test deployment request validation
			req := &DeploymentRequest{
				ID:          fmt.Sprintf("req-%d", id),
				TargetHost:  fmt.Sprintf("worker%d.example.com", id),
				PackagePath: "/tmp/worker.tar.gz",
				Command:     "./worker start",
				CreatedAt:   time.Now(),
			}
			
			if err := req.Validate(); err != nil {
				errors <- fmt.Errorf("validation error for req-%d: %w", id, err)
				return
			}
			
			// Test worker status validation
			status := &WorkerStatus{
				ID:           fmt.Sprintf("worker-%d", id),
				Host:         fmt.Sprintf("host%d.example.com", id),
				Status:       "idle",
				LastActivity: time.Now(),
			}
			
			if err := status.Validate(); err != nil {
				errors <- fmt.Errorf("validation error for worker-%d: %w", id, err)
				return
			}
			
			// Test helper methods
			resp := &DeploymentResponse{
				Status:  "completed",
				Success: true,
				Error:   "",
			}
			
			if !resp.IsSuccessful() {
				errors <- fmt.Errorf("helper method failed for req-%d", id)
				return
			}
		}(i)
	}
	
	wg.Wait()
	close(errors)
	
	for err := range errors {
		t.Error(err)
	}
}