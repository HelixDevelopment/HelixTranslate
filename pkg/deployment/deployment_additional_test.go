package deployment

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Network Discovery Tests ====================

func TestNetworkDiscoverer_sendBroadcastMessage(t *testing.T) {
	t.Run("Send broadcast message", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		// Create a mock connection (won't actually send)
		msg := BroadcastMessage{
			ServiceID: "test-service",
			Type:      "worker",
			Host:      "localhost",
			Port:      8443,
			Protocol:  "https",
			Timestamp: time.Now(),
		}

		// Without a real connection, this should fail but not panic
		// Skip if broadcast connection cannot be established
		err := nd.StartBroadcasting(context.Background(), nil)
		if err != nil {
			t.Skip("Broadcast connection not available in test environment")
		}

		err = nd.sendBroadcastMessage(msg)
		// May fail or succeed depending on network
		_ = err
	})
}

func TestNetworkDiscoverer_handleDiscoveryMessage(t *testing.T) {
	t.Run("Handle valid discovery message", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		msg := BroadcastMessage{
			ServiceID: "external-service",
			Type:      "worker",
			Host:      "192.168.1.100",
			Port:      8443,
			Protocol:  "https",
			Capabilities: map[string]interface{}{
				"version": "1.0.0",
			},
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		addr := &net.UDPAddr{
			IP:   net.ParseIP("192.168.1.100"),
			Port: 9999,
		}

		err = nd.handleDiscoveryMessage(data, addr)
		require.NoError(t, err)

		// Service should be registered
		services := nd.GetDiscoveredServices()
		assert.Contains(t, services, "external-service")
	})

	t.Run("Handle invalid JSON", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		addr := &net.UDPAddr{
			IP:   net.ParseIP("192.168.1.100"),
			Port: 9999,
		}

		err := nd.handleDiscoveryMessage([]byte("invalid json"), addr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
	})

	t.Run("Handle own service message", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		msg := BroadcastMessage{
			ServiceID: "own-service",
			Type:      "coordinator",
			Host:      "localhost",
			Port:      8443,
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(msg)
		require.NoError(t, err)

		// The default isOwnService returns false, so this will be registered.
		// We verify the behavior of the default implementation.
		addr := &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 9999,
		}

		err = nd.handleDiscoveryMessage(data, addr)
		require.NoError(t, err)

		// Default isOwnService returns false, so it will be registered
		services := nd.GetDiscoveredServices()
		assert.Contains(t, services, "own-service")
	})
}

func TestNetworkDiscoverer_isOwnService(t *testing.T) {
	cfg := &config.Config{}
	testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	nd := NewNetworkDiscoverer(cfg, testLogger)
	defer nd.Close()

	// Default implementation always returns false
	msg1 := BroadcastMessage{ServiceID: "any-service", Type: "coordinator"}
	assert.False(t, nd.isOwnService(msg1))
	msg2 := BroadcastMessage{ServiceID: "", Type: "worker"}
	assert.False(t, nd.isOwnService(msg2))
}

func TestNetworkDiscoverer_QueryService(t *testing.T) {
	cfg := &config.Config{}
	testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	nd := NewNetworkDiscoverer(cfg, testLogger)
	defer nd.Close()

	service := &NetworkService{
		ID:   "test-service",
		Name: "Test",
		Capabilities: map[string]interface{}{
			"version": "1.0.0",
			"workers": 5,
		},
	}

	caps, err := nd.QueryService(context.Background(), service)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", caps["version"])
	assert.Equal(t, 5, caps["workers"])
}

func TestNetworkDiscoverer_broadcastServices(t *testing.T) {
	t.Run("Broadcast multiple services", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		instances := map[string]*DeployedInstance{
			"main-1": {
				ID:          "main-1",
				Host:        "host1.example.com",
				Port:        8443,
				ContainerID: "abc123",
				Status:      "running",
			},
			"worker-1": {
				ID:          "worker-1",
				Host:        "host2.example.com",
				Port:        8444,
				ContainerID: "def456",
				Status:      "running",
			},
		}

		// Should not panic - need to start broadcasting first
		err := nd.StartBroadcasting(context.Background(), instances)
		if err != nil {
			t.Skip("Broadcast connection not available in test environment")
		}
		nd.broadcastServices(instances)
	})
}

func TestNetworkDiscoverer_broadcastLoop(t *testing.T) {
	t.Run("Broadcast loop respects context cancellation", func(t *testing.T) {
		cfg := &config.Config{}
		testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		nd := NewNetworkDiscoverer(cfg, testLogger)
		defer nd.Close()

		ctx, cancel := context.WithCancel(context.Background())
		instances := map[string]*DeployedInstance{}

		// Start broadcast loop in goroutine
		nd.wg.Add(1)
		done := make(chan struct{})
		go func() {
			nd.broadcastLoop(ctx, instances)
			close(done)
		}()

		// Cancel immediately
		cancel()

		// Should exit quickly
		select {
		case <-done:
			// Success
		case <-time.After(2 * time.Second):
			t.Error("broadcastLoop did not exit after context cancellation")
		}
	})
}

// ==================== Orchestrator Tests ====================

func TestDeploymentOrchestrator_emitEvent(t *testing.T) {
	t.Run("Emit event with valid event bus", func(t *testing.T) {
		eventBus := events.NewEventBus()
		received := make(chan events.Event, 1)

		eventBus.SubscribeAll(func(e events.Event) {
			received <- e
		})

		cfg := &config.Config{}
		orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
		defer orchestrator.Close()

		event := events.Event{
			Type:      "test.event",
			SessionID: "test-session",
			Message:   "Test message",
		}

		orchestrator.emitEvent(event)

		select {
		case e := <-received:
			assert.Equal(t, events.EventType("test.event"), e.Type)
			assert.Equal(t, "test-session", e.SessionID)
		case <-time.After(500 * time.Millisecond):
			t.Error("Event was not received")
		}
	})

	t.Run("Emit event with nil event bus does not panic", func(t *testing.T) {
		cfg := &config.Config{}
		orchestrator := NewDeploymentOrchestrator(cfg, nil)
		defer orchestrator.Close()

		event := events.Event{
			Type:      "test.event",
			SessionID: "test-session",
			Message:   "Test message",
		}

		assert.NotPanics(t, func() {
			orchestrator.emitEvent(event)
		})
	})
}

func TestDeploymentOrchestrator_validateInstanceConfig(t *testing.T) {
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	tests := []struct {
		name    string
		config  *DeploymentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: &DeploymentConfig{
				Host:          "worker.example.com",
				User:          "deploy",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: false,
		},
		{
			name: "Missing host",
			config: &DeploymentConfig{
				User:          "deploy",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "Missing user",
			config: &DeploymentConfig{
				Host:          "worker.example.com",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errMsg:  "user is required",
		},
		{
			name: "Missing docker image",
			config: &DeploymentConfig{
				Host:          "worker.example.com",
				User:          "deploy",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errMsg:  "docker image is required",
		},
		{
			name: "Missing container name",
			config: &DeploymentConfig{
				Host:        "worker.example.com",
				User:        "deploy",
				DockerImage: "translator:latest",
				Ports:       []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errMsg:  "container name is required",
		},
		{
			name: "No ports",
			config: &DeploymentConfig{
				Host:          "worker.example.com",
				User:          "deploy",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{},
			},
			wantErr: true,
			errMsg:  "at least one port mapping is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orchestrator.validateInstanceConfig(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeploymentOrchestrator_validateDeploymentPlan(t *testing.T) {
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	validConfig := &DeploymentConfig{
		Host:          "worker.example.com",
		User:          "deploy",
		DockerImage:   "translator:latest",
		ContainerName: "main",
		Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
	}

	tests := []struct {
		name    string
		plan    *DeploymentPlan
		wantErr bool
	}{
		{
			name: "Valid plan",
			plan: &DeploymentPlan{
				Main:    validConfig,
				Workers: []*DeploymentConfig{validConfig},
			},
			wantErr: false,
		},
		{
			name: "Nil main",
			plan: &DeploymentPlan{
				Main:    nil,
				Workers: []*DeploymentConfig{validConfig},
			},
			wantErr: true,
		},
		{
			name: "No workers",
			plan: &DeploymentPlan{
				Main:    validConfig,
				Workers: []*DeploymentConfig{},
			},
			wantErr: true,
		},
		{
			name: "Invalid worker config",
			plan: &DeploymentPlan{
				Main: validConfig,
				Workers: []*DeploymentConfig{
					{Host: "", User: "deploy", DockerImage: "test", ContainerName: "w1", Ports: []PortMapping{{HostPort: 8081, ContainerPort: 8081}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orchestrator.validateDeploymentPlan(tt.plan)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentOrchestrator_isPortAvailable(t *testing.T) {
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	// Test with a port that's likely available
	available := orchestrator.isPortAvailable("127.0.0.1", 54321)
	assert.True(t, available)

	// Test with a port that's likely in use (if ssh is running)
	// This might be false if port 22 is in use, which is expected
	_ = orchestrator.isPortAvailable("127.0.0.1", 22)
}

func TestDeploymentOrchestrator_GetDeployedInstances(t *testing.T) {
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	// Should be empty initially
	instances := orchestrator.GetDeployedInstances()
	assert.Empty(t, instances)

	// Add an instance directly
	orchestrator.mu.Lock()
	orchestrator.deployed["test-instance"] = &DeployedInstance{
		ID:   "test-instance",
		Host: "test.example.com",
		Port: 8443,
	}
	orchestrator.mu.Unlock()

	instances = orchestrator.GetDeployedInstances()
	assert.Len(t, instances, 1)
	assert.Contains(t, instances, "test-instance")

	// Verify it's a copy, not the original map
	instances["new-instance"] = &DeployedInstance{ID: "new-instance"}
	originalInstances := orchestrator.GetDeployedInstances()
	assert.NotContains(t, originalInstances, "new-instance")
}

func TestDeploymentOrchestrator_Close(t *testing.T) {
	t.Run("Close cleans up resources", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDeploymentOrchestrator(cfg, eventBus)

		err := orchestrator.Close()
		assert.NoError(t, err)
	})
}

// ==================== Docker Orchestrator Tests ====================

func TestDockerOrchestrator_addServiceToCompose(t *testing.T) {
	t.Run("Add service with UDP port", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		composeConfig := &DockerComposeConfig{
			Version:  "3.8",
			Services: make(map[string]*DockerServiceConfig),
		}

		config := &DeploymentConfig{
			DockerImage:   "test:latest",
			ContainerName: "test-container",
			Ports: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080, Protocol: "udp"},
			},
			Environment:   map[string]string{"ENV": "test"},
			Networks:      []string{"test-network"},
			RestartPolicy: "always",
		}

		err := orchestrator.addServiceToCompose(composeConfig, config, "test-service")
		require.NoError(t, err)

		service, exists := composeConfig.Services["test-service"]
		require.True(t, exists)
		assert.Equal(t, "test:latest", service.Image)
		assert.Equal(t, "test-container", service.ContainerName)
		assert.Contains(t, service.Ports, "8080:8080/udp")
		assert.Equal(t, "always", service.Restart)
	})

	t.Run("Add service with read-only volume", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		composeConfig := &DockerComposeConfig{
			Version:  "3.8",
			Services: make(map[string]*DockerServiceConfig),
		}

		config := &DeploymentConfig{
			DockerImage:   "test:latest",
			ContainerName: "test-container",
			Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			Volumes: []VolumeMapping{
				{HostPath: "/host/data", ContainerPath: "/data", ReadOnly: true},
				{HostPath: "/host/config", ContainerPath: "/config", ReadOnly: false},
			},
		}

		err := orchestrator.addServiceToCompose(composeConfig, config, "test-service")
		require.NoError(t, err)

		service := composeConfig.Services["test-service"]
		assert.Contains(t, service.Volumes, "/host/data:/data:ro")
		assert.Contains(t, service.Volumes, "/host/config:/config")
	})

	t.Run("Add service with health check", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		composeConfig := &DockerComposeConfig{
			Version:  "3.8",
			Services: make(map[string]*DockerServiceConfig),
		}

		config := &DeploymentConfig{
			DockerImage:   "test:latest",
			ContainerName: "test-container",
			Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			HealthCheck: &HealthCheckConfig{
				Test:        []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 60 * time.Second,
			},
		}

		err := orchestrator.addServiceToCompose(composeConfig, config, "test-service")
		require.NoError(t, err)

		service := composeConfig.Services["test-service"]
		require.NotNil(t, service.HealthCheck)
		assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost:8080/health"}, service.HealthCheck.Test)
		assert.Equal(t, "30s", service.HealthCheck.Interval)
		assert.Equal(t, "10s", service.HealthCheck.Timeout)
		assert.Equal(t, 3, service.HealthCheck.Retries)
		assert.Equal(t, "1m0s", service.HealthCheck.StartPeriod)
	})
}

func TestDockerOrchestrator_addSupportingServices(t *testing.T) {
	t.Run("Adds postgres and redis", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		composeConfig := &DockerComposeConfig{
			Version:  "3.8",
			Services: make(map[string]*DockerServiceConfig),
			Volumes:  make(map[string]*DockerVolumeConfig),
		}

		orchestrator.addSupportingServices(composeConfig)

		assert.Contains(t, composeConfig.Services, "postgres")
		assert.Contains(t, composeConfig.Services, "redis")
		assert.Contains(t, composeConfig.Volumes, "postgres-data")
		assert.Contains(t, composeConfig.Volumes, "redis-data")

		postgres := composeConfig.Services["postgres"]
		assert.Equal(t, "postgres:15-alpine", postgres.Image)
		assert.Equal(t, "translator", postgres.Environment["POSTGRES_USER"])

		redis := composeConfig.Services["redis"]
		assert.Equal(t, "redis:7-alpine", redis.Image)
		assert.Equal(t, []string{"redis-server", "--requirepass", "redis_secure_password"}, redis.Command)
	})
}

func TestDockerOrchestrator_formatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Zero", 0, ""},
		{"1 second", 1 * time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"1 minute", 1 * time.Minute, "1m0s"},
		{"Complex", 1*time.Hour + 30*time.Minute + 15*time.Second, "1h30m15s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== SSH Deployer Tests ====================

func TestSSHDeployer_InstanceOperations(t *testing.T) {
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	deployer := NewSSHDeployer(config)

	t.Run("DeployInstance returns mock ID", func(t *testing.T) {
		ctx := context.Background()
		deploymentConfig := &DeploymentConfig{
			Host:        "test.example.com",
			DockerImage: "test:latest",
		}
		id, err := deployer.DeployInstance(ctx, deploymentConfig)
		require.NoError(t, err)
		assert.Equal(t, "mock-container-id", id)
	})

	t.Run("CheckInstanceHealth returns nil", func(t *testing.T) {
		ctx := context.Background()
		err := deployer.CheckInstanceHealth(ctx, "test-instance")
		assert.NoError(t, err)
	})

	t.Run("UpdateInstance returns nil", func(t *testing.T) {
		ctx := context.Background()
		err := deployer.UpdateInstance(ctx, "test-instance", &DeploymentConfig{})
		assert.NoError(t, err)
	})

	t.Run("RestartInstance returns nil", func(t *testing.T) {
		ctx := context.Background()
		err := deployer.RestartInstance(ctx, "test-instance")
		assert.NoError(t, err)
	})

	t.Run("Close returns nil", func(t *testing.T) {
		err := deployer.Close()
		assert.NoError(t, err)
	})
}

func TestMockSSHClient(t *testing.T) {
	client := NewMockSSHClient(true)

	t.Run("SetAuthFail and SetExecFail", func(t *testing.T) {
		client.SetAuthFail(true)
		assert.True(t, client.shouldFailAuth)

		client.SetExecFail(true)
		assert.True(t, client.shouldFailExec)

		client.SetAuthFail(false)
		assert.False(t, client.shouldFailAuth)
	})

	t.Run("IsConnected and GetExecutedCommands", func(t *testing.T) {
		assert.False(t, client.IsConnected("test"))
		assert.Empty(t, client.GetExecutedCommands("test"))
	})
}

// ==================== API Logger Tests ====================

func TestAPICommunicationLogger_getStatusText(t *testing.T) {
	logger, err := NewAPICommunicationLogger(t.TempDir() + "/test.log")
	require.NoError(t, err)
	defer logger.Close()

	tests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{409, "Conflict"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{504, "Gateway Timeout"},
		{999, ""},
		{0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := logger.getStatusText(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPICommunicationLogger_getProtocol(t *testing.T) {
	logger, err := NewAPICommunicationLogger(t.TempDir() + "/test.log")
	require.NoError(t, err)
	defer logger.Close()

	tests := []struct {
		port     int
		expected string
	}{
		{443, "https"},
		{8443, "https"},
		{80, "http"},
		{8080, "http"},
		{5000, "http"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := logger.getProtocol(tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPICommunicationLogger_formatDuration(t *testing.T) {
	logger, err := NewAPICommunicationLogger(t.TempDir() + "/test.log")
	require.NoError(t, err)
	defer logger.Close()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{50 * time.Millisecond, "50ms"},
		{150 * time.Millisecond, "150ms"},
		{1 * time.Second, "1.0s"},
		{2500 * time.Millisecond, "2.5s"},
		{100 * time.Second, "100.0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := logger.formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAPICommunicationLogger_LogCommunication_Response(t *testing.T) {
	tmpFile := t.TempDir() + "/api.log"
	logger, err := NewAPICommunicationLogger(tmpFile)
	require.NoError(t, err)
	defer logger.Close()

	// Log a response entry
	entry := &APICommunicationLog{
		Timestamp:    time.Now(),
		SourceHost:   "client",
		SourcePort:   12345,
		TargetHost:   "server",
		TargetPort:   443,
		Method:       "POST",
		URL:          "/api/test",
		StatusCode:   200,
		RequestSize:  1024,
		ResponseSize: 2048,
		Duration:     150 * time.Millisecond,
		UserAgent:    "TestClient/1.0",
	}

	err = logger.LogCommunication(entry)
	require.NoError(t, err)

	// Read log file
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "200")
	assert.Contains(t, string(content), "OK")
}

func TestAPICommunicationLogger_LogCommunication_Request(t *testing.T) {
	tmpFile := t.TempDir() + "/api.log"
	logger, err := NewAPICommunicationLogger(tmpFile)
	require.NoError(t, err)
	defer logger.Close()

	// Log a request entry (status code 0)
	entry := &APICommunicationLog{
		Timestamp:   time.Now(),
		SourceHost:  "client",
		SourcePort:  12345,
		TargetHost:  "server",
		TargetPort:  8443,
		Method:      "GET",
		URL:         "/api/health",
		RequestSize: 0,
	}

	err = logger.LogCommunication(entry)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "GET")
	assert.Contains(t, string(content), "https")
}

func TestAPICommunicationLogger_LogCommunication_Error(t *testing.T) {
	tmpFile := t.TempDir() + "/api.log"
	logger, err := NewAPICommunicationLogger(tmpFile)
	require.NoError(t, err)
	defer logger.Close()

	entry := &APICommunicationLog{
		Timestamp:  time.Now(),
		TargetHost: "server",
		TargetPort: 8080,
		Method:     "POST",
		URL:        "/api/fail",
		StatusCode: 500,
		Error:      "Internal Server Error",
	}

	err = logger.LogCommunication(entry)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "HTTP FAILED")
	assert.Contains(t, string(content), "Internal Server Error")
}

// ==================== Types Tests ====================

func TestValidationError_Error(t *testing.T) {
	t.Run("Error with field and message", func(t *testing.T) {
		err := &ValidationError{
			Field:   "host",
			Message: "host is required",
		}
		assert.Equal(t, "host is required (field: host)", err.Error())
	})

	t.Run("Error implements error interface", func(t *testing.T) {
		var err error = &ValidationError{Field: "test", Message: "test error"}
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "test error")
	})
}

func TestConnectionError_Error(t *testing.T) {
	t.Run("Error with underlying error", func(t *testing.T) {
		underlying := &ValidationError{Field: "host", Message: "missing"}
		err := &ConnectionError{
			Type: "config_validation",
			Err:  underlying,
		}
		assert.Contains(t, err.Error(), "config_validation")
		assert.Contains(t, err.Error(), "missing")
		assert.Equal(t, underlying, err.Unwrap())
	})

	t.Run("Error with nil underlying", func(t *testing.T) {
		err := &ConnectionError{
			Type: "timeout",
			Err:  nil,
		}
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestDockerComposeConfig_Struct(t *testing.T) {
	config := &DockerComposeConfig{
		Version: "3.8",
		Services: map[string]*DockerServiceConfig{
			"app": {
				Image:         "app:latest",
				ContainerName: "myapp",
				Ports:         []string{"8080:8080"},
				Environment:   map[string]string{"ENV": "production"},
				Volumes:       []string{"data:/data"},
				Networks:      []string{"app-network"},
				Restart:       "unless-stopped",
				DependsOn:     []string{"db"},
				HealthCheck: &DockerHealthCheck{
					Test:     []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
					Interval: "30s",
					Timeout:  "10s",
					Retries:  3,
				},
				Command: []string{"/app/start"},
			},
		},
		Networks: map[string]*DockerNetworkConfig{
			"app-network": {Driver: "bridge"},
		},
		Volumes: map[string]*DockerVolumeConfig{
			"data": {Driver: "local"},
		},
	}

	assert.Equal(t, "3.8", config.Version)
	assert.Len(t, config.Services, 1)
	assert.Len(t, config.Networks, 1)
	assert.Len(t, config.Volumes, 1)

	app := config.Services["app"]
	assert.Equal(t, "app:latest", app.Image)
	assert.Equal(t, "myapp", app.ContainerName)
	assert.Equal(t, []string{"8080:8080"}, app.Ports)
	assert.Equal(t, "unless-stopped", app.Restart)
	assert.Equal(t, []string{"db"}, app.DependsOn)
	assert.Equal(t, []string{"/app/start"}, app.Command)

	hc := app.HealthCheck
	require.NotNil(t, hc)
	assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost:8080/health"}, hc.Test)
	assert.Equal(t, "30s", hc.Interval)
	assert.Equal(t, "10s", hc.Timeout)
	assert.Equal(t, 3, hc.Retries)
}

func TestDeployedInstance_Mutex(t *testing.T) {
	instance := &DeployedInstance{
		ID:       "test",
		Host:     "localhost",
		Port:     8443,
		Status:   "running",
		LastSeen: time.Now(),
	}

	// Test concurrent access
	done := make(chan bool, 2)

	go func() {
		instance.mu.Lock()
		instance.Status = "updating"
		instance.mu.Unlock()
		done <- true
	}()

	go func() {
		instance.mu.RLock()
		_ = instance.Status
		instance.mu.RUnlock()
		done <- true
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for mutex operations")
		}
	}
}

func TestDeploymentPlan_Struct(t *testing.T) {
	plan := &DeploymentPlan{
		Main: &DeploymentConfig{
			Host:          "main.example.com",
			ContainerName: "main",
		},
		Workers: []*DeploymentConfig{
			{Host: "worker1.example.com", ContainerName: "worker1"},
			{Host: "worker2.example.com", ContainerName: "worker2"},
		},
	}

	assert.NotNil(t, plan.Main)
	assert.Equal(t, "main.example.com", plan.Main.Host)
	assert.Len(t, plan.Workers, 2)
	assert.Equal(t, "worker1", plan.Workers[0].ContainerName)
}

func TestNetworkDiscoverer_GetDiscoveredServices_Cleanup(t *testing.T) {
	cfg := &config.Config{}
	testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	nd := NewNetworkDiscoverer(cfg, testLogger)
	defer nd.Close()

	// Add services with different TTLs
	nd.mu.Lock()
	nd.services["expired"] = &NetworkService{
		ID:       "expired",
		LastSeen: time.Now().Add(-5 * time.Minute),
		TTL:      1 * time.Minute,
	}
	nd.services["active"] = &NetworkService{
		ID:       "active",
		LastSeen: time.Now(),
		TTL:      5 * time.Minute,
	}
	nd.mu.Unlock()

	services := nd.GetDiscoveredServices()
	assert.NotContains(t, services, "expired")
	assert.Contains(t, services, "active")
}

func TestDockerOrchestrator_writeComposeFile(t *testing.T) {
	t.Run("Write to valid path", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "compose.yml")

		composeConfig := &DockerComposeConfig{
			Version: "3.8",
			Services: map[string]*DockerServiceConfig{
				"test": {Image: "nginx"},
			},
		}

		err := orchestrator.writeComposeFile(composeConfig, path)
		require.NoError(t, err)

		// Verify file exists and is valid YAML
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "version")
		assert.Contains(t, string(content), "nginx")
	})

	t.Run("Write to invalid path", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		orchestrator := NewDockerOrchestrator(cfg, eventBus)

		composeConfig := &DockerComposeConfig{Version: "3.8"}
		err := orchestrator.writeComposeFile(composeConfig, "/nonexistent/path/compose.yml")
		assert.Error(t, err)
	})
}

func TestSSHDeployConfig_Validate_Defaults(t *testing.T) {
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Username: "user",
		Password: "pass",
	}

	err := config.Validate()
	require.NoError(t, err)

	// Verify defaults are set
	assert.Equal(t, 22, config.Port)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.ConnectRetries)
	assert.Equal(t, 10*time.Minute, config.CommandTimeout)
}
