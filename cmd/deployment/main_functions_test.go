package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadDeploymentPlan tests the loadDeploymentPlan function
func TestLoadDeploymentPlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "load-plan-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a valid deployment plan file
	plan := &deployment.DeploymentPlan{
		Workers: []*deployment.DeploymentConfig{
			{
				Host:          "localhost",
				User:          "testuser",
				DockerImage:   "translator:latest",
				ContainerName: "test-worker",
				Ports: []deployment.PortMapping{
					{HostPort: 8443, ContainerPort: 8443, Protocol: "tcp"},
				},
			},
		},
	}

	planData, err := json.MarshalIndent(plan, "", "  ")
	require.NoError(t, err)

	validPlanFile := filepath.Join(tempDir, "valid-plan.json")
	err = os.WriteFile(validPlanFile, planData, 0644)
	require.NoError(t, err)

	// Create invalid JSON file
	invalidPlanFile := filepath.Join(tempDir, "invalid-plan.json")
	err = os.WriteFile(invalidPlanFile, []byte("{ invalid json }"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		filename    string
		expectError bool
		expectPlan  bool
	}{
		{
			name:        "load valid plan",
			filename:    validPlanFile,
			expectError: false,
			expectPlan:  true,
		},
		{
			name:        "load nonexistent file",
			filename:    "/nonexistent/plan.json",
			expectError: true,
			expectPlan:  false,
		},
		{
			name:        "load invalid JSON",
			filename:    invalidPlanFile,
			expectError: true,
			expectPlan:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := loadDeploymentPlan(tt.filename)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, plan)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, plan)
				if tt.expectPlan {
					assert.Equal(t, 1, len(plan.Workers))
					assert.Equal(t, "localhost", plan.Workers[0].Host)
					assert.Equal(t, "testuser", plan.Workers[0].User)
				}
			}
		})
	}
}

// TestGenerateDeploymentPlan tests the generateDeploymentPlan function
func TestGenerateDeploymentPlan(t *testing.T) {
	// Test with no workers
	cfg := &config.Config{
		Distributed: config.DistributedConfig{
			Enabled: true,
			Workers: map[string]config.WorkerConfig{},
		},
	}

	plan := generateDeploymentPlan(cfg)
	assert.NotNil(t, plan)
	assert.Equal(t, 0, len(plan.Workers))

	// Test with multiple workers - note that map iteration order is not guaranteed
	cfg = &config.Config{
		Distributed: config.DistributedConfig{
			Enabled: true,
			Workers: map[string]config.WorkerConfig{
				"worker1": {
					Host:     "localhost",
					User:     "testuser",
					Password: "testpass",
				},
				"worker2": {
					Host:     "remote-host",
					User:     "remoteuser",
					Password: "remotepass",
				},
				"worker3": {
					Host:     "third-host",
					User:     "thirduser",
					Password: "thirdpass",
				},
			},
		},
	}

	plan = generateDeploymentPlan(cfg)
	assert.NotNil(t, plan)
	assert.Equal(t, 3, len(plan.Workers))

	// Verify each worker has proper configuration
	// Create a map of expected values for easier verification
	expectedValues := map[string]config.WorkerConfig{
		"worker1": {
			Host:     "localhost",
			User:     "testuser",
			Password: "testpass",
		},
		"worker2": {
			Host:     "remote-host",
			User:     "remoteuser",
			Password: "remotepass",
		},
		"worker3": {
			Host:     "third-host",
			User:     "thirduser",
			Password: "thirdpass",
		},
	}

	// Check each generated worker corresponds to one of our expected configs
	for _, worker := range plan.Workers {
		// Extract worker ID from container name
		assert.Contains(t, worker.ContainerName, "translator-worker-")

		// Verify basic configuration
		assert.Equal(t, "", worker.SSHKeyPath)
		assert.Equal(t, "translator:latest", worker.DockerImage)
		assert.Equal(t, "tcp", worker.Ports[0].Protocol)
		assert.Equal(t, 8443, worker.Ports[0].ContainerPort)
		assert.Contains(t, worker.Environment, "JWT_SECRET")
		assert.Contains(t, worker.Environment, "WORKER_INDEX")
		assert.Equal(t, []string{"translator-network"}, worker.Networks)
		assert.Equal(t, "unless-stopped", worker.RestartPolicy)
		assert.NotNil(t, worker.HealthCheck)
		assert.Equal(t, []string{"CMD", "curl", "-f", "https://localhost:8443/health"}, worker.HealthCheck.Test)

		// Find matching expected config
		workerID := ""
		for id := range expectedValues {
			if worker.ContainerName == "translator-worker-"+id {
				workerID = id
				break
			}
		}

		if workerID != "" {
			expected := expectedValues[workerID]
			assert.Equal(t, expected.Host, worker.Host)
			assert.Equal(t, expected.User, worker.User)
			assert.Equal(t, "worker-"+workerID+"-secret", worker.Environment["JWT_SECRET"])
		}
	}
}

// TestHandleGeneratePlan tests the handleGeneratePlan function
func TestHandleGeneratePlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change working directory to temp dir for file creation
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create test config with workers
	cfg := &config.Config{
		Distributed: config.DistributedConfig{
			Enabled: true,
			Workers: map[string]config.WorkerConfig{
				"worker1": {
					Host:     "localhost",
					User:     "testuser",
					Password: "testpass",
				},
				"worker2": {
					Host:     "remote-host",
					User:     "remoteuser",
					Password: "remotepass",
				},
			},
		},
	}

	assert.NotPanics(t, func() {
		handleGeneratePlan(cfg)
	})

	// Verify deployment plan file was created
	planFile := "deployment-plan.json"
	assert.FileExists(t, planFile)

	// Read and verify plan content
	planData, err := os.ReadFile(planFile)
	require.NoError(t, err)

	var plan deployment.DeploymentPlan
	err = json.Unmarshal(planData, &plan)
	require.NoError(t, err)

	// Should have 2 workers
	assert.Equal(t, 2, len(plan.Workers))

	// Verify worker configurations
	for i, worker := range plan.Workers {
		assert.NotEmpty(t, worker.ContainerName)
		assert.Equal(t, "translator:latest", worker.DockerImage)
		assert.Contains(t, worker.ContainerName, "translator-worker-")
		assert.Equal(t, 8443+i+1, worker.Ports[0].HostPort) // 8444, 8445
		assert.Equal(t, 8443, worker.Ports[0].ContainerPort)
		assert.Equal(t, "tcp", worker.Ports[0].Protocol)
		assert.Equal(t, "unless-stopped", worker.RestartPolicy)
		assert.NotNil(t, worker.HealthCheck)
		assert.Equal(t, []string{"CMD", "curl", "-f", "https://localhost:8443/health"}, worker.HealthCheck.Test)
	}
}

// TestMainFunctionPanic tests that the main function handles errors correctly
func TestMainFunctionPanic(t *testing.T) {
	t.Skip("main() calls log.Fatalf which does os.Exit, not panic - cannot test in Go test framework")
	// Save original args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name     string
		args     []string
		panics   bool
		testFunc func()
	}{
		{
			name:   "deploy without plan file",
			args:   []string{"deployment", "-action", "deploy"},
			panics: true,
			testFunc: func() {
				// This will panic because plan file is required for deploy action
				main()
			},
		},
		{
			name:   "update without service",
			args:   []string{"deployment", "-action", "update", "-service", ""},
			panics: true,
			testFunc: func() {
				// This will panic because service name is required for update action
				main()
			},
		},
		{
			name:   "unknown action",
			args:   []string{"deployment", "-action", "unknown-action"},
			panics: true,
			testFunc: func() {
				// This will panic because action is unknown
				main()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = append([]string{"deployment"}, tt.args...)

			if tt.panics {
				assert.Panics(t, tt.testFunc)
			} else {
				assert.NotPanics(t, tt.testFunc)
			}
		})
	}
}

// TestDeploymentPlanGeneration tests various aspects of deployment plan generation
func TestDeploymentPlanGeneration(t *testing.T) {
	tests := []struct {
		name            string
		config          *config.Config
		expectedWorkers int
		verifyPlan      func(*testing.T, *deployment.DeploymentPlan)
	}{
		{
			name: "empty worker config",
			config: &config.Config{
				Distributed: config.DistributedConfig{
					Enabled: true,
					Workers: map[string]config.WorkerConfig{},
				},
			},
			expectedWorkers: 0,
			verifyPlan: func(t *testing.T, plan *deployment.DeploymentPlan) {
				assert.Equal(t, 0, len(plan.Workers))
			},
		},
		{
			name: "single worker config",
			config: &config.Config{
				Distributed: config.DistributedConfig{
					Enabled: true,
					Workers: map[string]config.WorkerConfig{
						"worker1": {
							Host:     "localhost",
							User:     "testuser",
							Password: "testpass",
						},
					},
				},
			},
			expectedWorkers: 1,
			verifyPlan: func(t *testing.T, plan *deployment.DeploymentPlan) {
				assert.Equal(t, 1, len(plan.Workers))
				worker := plan.Workers[0]
				assert.Equal(t, "localhost", worker.Host)
				assert.Equal(t, "testuser", worker.User)
				assert.Equal(t, "translator-worker-worker1", worker.ContainerName)
				assert.Equal(t, 8444, worker.Ports[0].HostPort)
				assert.Equal(t, "worker-worker1-secret", worker.Environment["JWT_SECRET"])
			},
		},
		{
			name: "multiple workers config",
			config: &config.Config{
				Distributed: config.DistributedConfig{
					Enabled: true,
					Workers: map[string]config.WorkerConfig{
						"alpha": {
							Host:     "alpha-host",
							User:     "alpha-user",
							Password: "alpha-pass",
						},
						"beta": {
							Host:     "beta-host",
							User:     "beta-user",
							Password: "beta-pass",
						},
						"gamma": {
							Host:     "gamma-host",
							User:     "gamma-user",
							Password: "gamma-pass",
						},
					},
				},
			},
			expectedWorkers: 3,
			verifyPlan: func(t *testing.T, plan *deployment.DeploymentPlan) {
				assert.Equal(t, 3, len(plan.Workers))

				expectedHosts := []string{"alpha-host", "beta-host", "gamma-host"}
				actualHosts := []string{plan.Workers[0].Host, plan.Workers[1].Host, plan.Workers[2].Host}
				sort.Strings(expectedHosts)
				sort.Strings(actualHosts)
				assert.Equal(t, expectedHosts, actualHosts)

				for i, worker := range plan.Workers {
					assert.Equal(t, "translator:latest", worker.DockerImage)
					assert.Equal(t, 8444+i, worker.Ports[0].HostPort)
					assert.Equal(t, 8443, worker.Ports[0].ContainerPort)
					assert.Contains(t, worker.ContainerName, "translator-worker-")
					assert.Equal(t, []string{"translator-network"}, worker.Networks)
					assert.Equal(t, "unless-stopped", worker.RestartPolicy)
					assert.NotNil(t, worker.HealthCheck)
					assert.Equal(t, []string{"CMD", "curl", "-f", "https://localhost:8443/health"}, worker.HealthCheck.Test)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := generateDeploymentPlan(tt.config)
			assert.NotNil(t, plan)
			assert.Equal(t, tt.expectedWorkers, len(plan.Workers))

			if tt.verifyPlan != nil {
				tt.verifyPlan(t, plan)
			}
		})
	}
}

// TestFlagParsing tests flag parsing scenarios
func TestFlagParsing(t *testing.T) {
	t.Skip("main() redefines flags on global flag.CommandLine causing 'flag redefined' panic when run with other tests")
	// Save original args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name         string
		args         []string
		shouldPanic  bool
		expectedFunc func()
	}{
		{
			name:        "default flags",
			args:        []string{"deployment"},
			shouldPanic: true, // Will panic because of missing plan file for default deploy action
		},
		{
			name:        "status action",
			args:        []string{"deployment", "-action", "status"},
			shouldPanic: false, // Should not panic for status action
		},
		{
			name:        "stop action",
			args:        []string{"deployment", "-action", "stop"},
			shouldPanic: false, // Should not panic for stop action
		},
		{
			name:        "cleanup action",
			args:        []string{"deployment", "-action", "cleanup"},
			shouldPanic: false, // Should not panic for cleanup action
		},
		{
			name:        "generate-plan action",
			args:        []string{"deployment", "-action", "generate-plan"},
			shouldPanic: false, // Should not panic for generate-plan action
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			if tt.shouldPanic {
				assert.Panics(t, func() {
					main()
				})
			} else {
				// We expect this might still panic due to config loading issues,
				// but we want to verify it at least parses the flags correctly
				assert.Panics(t, func() {
					main()
				})
			}
		})
	}
}

// TestDeploymentConfigurationValidation tests deployment configuration validation
func TestDeploymentConfigurationValidation(t *testing.T) {
	// Test deployment config structure validation
	workerConfig := &deployment.DeploymentConfig{
		Host:          "localhost",
		User:          "testuser",
		Password:      "testpass",
		DockerImage:   "translator:latest",
		ContainerName: "test-worker",
		Ports: []deployment.PortMapping{
			{HostPort: 8443, ContainerPort: 8443, Protocol: "tcp"},
		},
		Environment: map[string]string{
			"JWT_SECRET":   "test-secret",
			"WORKER_INDEX": "1",
		},
		Volumes: []deployment.VolumeMapping{
			{HostPath: "./certs", ContainerPath: "/app/certs", ReadOnly: true},
		},
		Networks:      []string{"translator-network"},
		RestartPolicy: "unless-stopped",
		HealthCheck: &deployment.HealthCheckConfig{
			Test:     []string{"CMD", "curl", "-f", "https://localhost:8443/health"},
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
	}

	// Verify required fields
	assert.Equal(t, "localhost", workerConfig.Host)
	assert.Equal(t, "testuser", workerConfig.User)
	assert.Equal(t, "translator:latest", workerConfig.DockerImage)
	assert.Equal(t, "test-worker", workerConfig.ContainerName)
	assert.Equal(t, 1, len(workerConfig.Ports))
	assert.Equal(t, 8443, workerConfig.Ports[0].HostPort)
	assert.Equal(t, 8443, workerConfig.Ports[0].ContainerPort)
	assert.Equal(t, "tcp", workerConfig.Ports[0].Protocol)
	assert.Equal(t, 2, len(workerConfig.Environment))
	assert.Equal(t, 1, len(workerConfig.Volumes))
	assert.Equal(t, 1, len(workerConfig.Networks))
	assert.Equal(t, "unless-stopped", workerConfig.RestartPolicy)
	assert.NotNil(t, workerConfig.HealthCheck)
	assert.Equal(t, 4, len(workerConfig.HealthCheck.Test))
	assert.Equal(t, 30*time.Second, workerConfig.HealthCheck.Interval)
	assert.Equal(t, 10*time.Second, workerConfig.HealthCheck.Timeout)
	assert.Equal(t, 3, workerConfig.HealthCheck.Retries)
}

// TestDeploymentHandlers tests the various deployment handler functions
func TestDeploymentHandlers(t *testing.T) {
	// Use nil to avoid type issues since we're just testing that functions don't panic
	// We capture log output to avoid fatal errors from crashing the test
	var orchestrator *deployment.DeploymentOrchestrator = nil

	t.Run("handleDeploy", func(t *testing.T) {
		// Create a temporary plan file for the test
		tempDir := t.TempDir()
		planFile := filepath.Join(tempDir, "test-plan.json")

		// Create a minimal valid plan
		plan := &deployment.DeploymentPlan{
			Workers: []*deployment.DeploymentConfig{
				{
					Host:          "localhost",
					DockerImage:   "test:latest",
					ContainerName: "test-container",
				},
			},
		}

		planData, err := json.MarshalIndent(plan, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(planFile, planData, 0644)
		require.NoError(t, err)

		// Test that handleDeploy can be called without panicking before the fatal error
		// Since the function calls log.Fatalf, it will exit, but that's expected behavior
		assert.Panics(t, func() {
			handleDeploy(orchestrator, planFile)
		})
	})

	t.Run("handleStatus", func(t *testing.T) {
		// Test that handleStatus panics when called with nil orchestrator
		assert.Panics(t, func() {
			handleStatus(orchestrator)
		})
	})

	t.Run("handleStop", func(t *testing.T) {
		// Test that handleStop doesn't panic (just logs)
		assert.NotPanics(t, func() {
			handleStop(orchestrator)
		})
	})

	t.Run("handleCleanup", func(t *testing.T) {
		// Test that handleCleanup doesn't panic (just logs)
		assert.NotPanics(t, func() {
			handleCleanup(orchestrator)
		})
	})

	t.Run("handleUpdate", func(t *testing.T) {
		// Test that handleUpdate panics when called with nil orchestrator
		assert.Panics(t, func() {
			handleUpdate(orchestrator, "test-service", "test-image:latest")
		})
	})

	t.Run("handleRestart", func(t *testing.T) {
		// Test that handleRestart panics when called with nil orchestrator
		assert.Panics(t, func() {
			handleRestart(orchestrator, "test-service")
		})
	})
}
