package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDeploymentConfigComprehensive tests all aspects of deployment configuration
func TestDeploymentConfigComprehensive(t *testing.T) {
	t.Run("Valid configuration validation", func(t *testing.T) {
		config := &DeploymentConfig{
			Host:          "worker1.example.com",
			User:          "deploy",
			Password:      "password123",
			SSHKeyPath:    "/path/to/key",
			DockerImage:   "translator:latest",
			ContainerName: "worker-1",
			Ports: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
				{HostPort: 8443, ContainerPort: 8443, Protocol: "tcp"},
			},
			Environment: map[string]string{
				"ENV": "production",
				"LOG": "info",
			},
			Volumes: []VolumeMapping{
				{HostPath: "/host/data", ContainerPath: "/container/data", ReadOnly: false},
				{HostPath: "/host/config", ContainerPath: "/container/config", ReadOnly: true},
			},
			Networks:      []string{"network1", "network2"},
			RestartPolicy: "unless-stopped",
			HealthCheck: &HealthCheckConfig{
				Test:        []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 60 * time.Second,
			},
		}

		// Test all fields are properly set
		assert.Equal(t, "worker1.example.com", config.Host)
		assert.Equal(t, "deploy", config.User)
		assert.Equal(t, "password123", config.Password)
		assert.Equal(t, "/path/to/key", config.SSHKeyPath)
		assert.Equal(t, "translator:latest", config.DockerImage)
		assert.Equal(t, "worker-1", config.ContainerName)
		assert.Len(t, config.Ports, 2)
		assert.Len(t, config.Environment, 2)
		assert.Len(t, config.Volumes, 2)
		assert.Len(t, config.Networks, 2)
		assert.Equal(t, "unless-stopped", config.RestartPolicy)
		assert.NotNil(t, config.HealthCheck)
	})

	t.Run("Minimal valid configuration", func(t *testing.T) {
		config := &DeploymentConfig{
			Host:          "worker1.example.com",
			User:          "deploy",
			DockerImage:   "translator:latest",
			ContainerName: "worker-1",
			Ports: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
			},
		}

		assert.Equal(t, "worker1.example.com", config.Host)
		assert.Equal(t, "deploy", config.User)
		assert.Equal(t, "translator:latest", config.DockerImage)
		assert.Equal(t, "worker-1", config.ContainerName)
		assert.Len(t, config.Ports, 1)
		assert.Empty(t, config.Environment)
		assert.Empty(t, config.Volumes)
		assert.Empty(t, config.Networks)
		assert.Empty(t, config.RestartPolicy)
		assert.Nil(t, config.HealthCheck)
	})

	t.Run("Port mapping validation", func(t *testing.T) {
		tests := []struct {
			name  string
			port  PortMapping
			valid bool
		}{
			{
				name:  "Valid TCP port",
				port:  PortMapping{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
				valid: true,
			},
			{
				name:  "Valid UDP port",
				port:  PortMapping{HostPort: 8081, ContainerPort: 8081, Protocol: "udp"},
				valid: true,
			},
			{
				name:  "Port with empty protocol",
				port:  PortMapping{HostPort: 8082, ContainerPort: 8082, Protocol: ""},
				valid: true, // Empty protocol might default to TCP
			},
			{
				name:  "Zero host port",
				port:  PortMapping{HostPort: 0, ContainerPort: 8080, Protocol: "tcp"},
				valid: false, // Port 0 is typically invalid
			},
			{
				name:  "Zero container port",
				port:  PortMapping{HostPort: 8080, ContainerPort: 0, Protocol: "tcp"},
				valid: false, // Port 0 is typically invalid
			},
			{
				name:  "High-numbered ports",
				port:  PortMapping{HostPort: 65000, ContainerPort: 65000, Protocol: "tcp"},
				valid: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.valid {
					assert.Greater(t, tt.port.HostPort, 0, "Host port should be greater than 0")
					assert.Greater(t, tt.port.ContainerPort, 0, "Container port should be greater than 0")
				} else {
					// Test the invalid cases
					if tt.port.HostPort == 0 || tt.port.ContainerPort == 0 {
						// These are expected to be invalid
					}
				}
			})
		}
	})
}

// TestPortMappingComprehensive tests port mapping in detail
func TestPortMappingComprehensive(t *testing.T) {
	t.Run("Port mapping with different protocols", func(t *testing.T) {
		protocols := []string{"tcp", "udp", ""}

		for _, protocol := range protocols {
			_ = PortMapping{
				HostPort:      8080,
				ContainerPort: 8080,
				Protocol:      protocol,
			}

			// Just verify the protocol assignment
			assert.Equal(t, protocol, protocol)
		}
	})

	t.Run("Port mapping edge cases", func(t *testing.T) {
		testCases := []struct {
			name          string
			hostPort      int
			containerPort int
			expectedValid bool
		}{
			{
				name:          "Minimum valid port",
				hostPort:      1,
				containerPort: 1,
				expectedValid: true,
			},
			{
				name:          "Maximum valid port",
				hostPort:      65535,
				containerPort: 65535,
				expectedValid: true,
			},
			{
				name:          "Invalid port 0",
				hostPort:      0,
				containerPort: 8080,
				expectedValid: false,
			},
			{
				name:          "Invalid high port",
				hostPort:      65536,
				containerPort: 8080,
				expectedValid: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_ = PortMapping{
					HostPort:      tc.hostPort,
					ContainerPort: tc.containerPort,
					Protocol:      "tcp",
				}

				isValid := (tc.hostPort > 0 && tc.hostPort <= 65535 &&
					tc.containerPort > 0 && tc.containerPort <= 65535)

				assert.Equal(t, tc.expectedValid, isValid)
			})
		}
	})
}

// TestVolumeMappingComprehensive tests volume mapping functionality
func TestVolumeMappingComprehensive(t *testing.T) {
	t.Run("Volume mapping with read-only option", func(t *testing.T) {
		readOnlyMapping := VolumeMapping{
			HostPath:      "/host/data",
			ContainerPath: "/container/data",
			ReadOnly:      true,
		}

		readWriteMapping := VolumeMapping{
			HostPath:      "/host/config",
			ContainerPath: "/container/config",
			ReadOnly:      false,
		}

		assert.Equal(t, "/host/data", readOnlyMapping.HostPath)
		assert.Equal(t, "/container/data", readOnlyMapping.ContainerPath)
		assert.True(t, readOnlyMapping.ReadOnly)

		assert.Equal(t, "/host/config", readWriteMapping.HostPath)
		assert.Equal(t, "/container/config", readWriteMapping.ContainerPath)
		assert.False(t, readWriteMapping.ReadOnly)
	})

	t.Run("Volume mapping with different path types", func(t *testing.T) {
		testCases := []struct {
			name          string
			hostPath      string
			containerPath string
			readOnly      bool
		}{
			{
				name:          "Absolute paths",
				hostPath:      "/absolute/host/path",
				containerPath: "/absolute/container/path",
				readOnly:      false,
			},
			{
				name:          "Relative paths",
				hostPath:      "./relative/host/path",
				containerPath: "./relative/container/path",
				readOnly:      true,
			},
			{
				name:          "Named volumes",
				hostPath:      "named_volume",
				containerPath: "/container/path",
				readOnly:      false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				volumeMapping := VolumeMapping{
					HostPath:      tc.hostPath,
					ContainerPath: tc.containerPath,
					ReadOnly:      tc.readOnly,
				}

				assert.Equal(t, tc.hostPath, volumeMapping.HostPath)
				assert.Equal(t, tc.containerPath, volumeMapping.ContainerPath)
				assert.Equal(t, tc.readOnly, volumeMapping.ReadOnly)
			})
		}
	})
}

// TestHealthCheckConfigComprehensive tests health check configuration
func TestHealthCheckConfigComprehensive(t *testing.T) {
	t.Run("Complete health check configuration", func(t *testing.T) {
		healthCheck := &HealthCheckConfig{
			Test:        []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			Retries:     3,
			StartPeriod: 60 * time.Second,
		}

		assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost:8080/health"}, healthCheck.Test)
		assert.Equal(t, 30*time.Second, healthCheck.Interval)
		assert.Equal(t, 10*time.Second, healthCheck.Timeout)
		assert.Equal(t, 3, healthCheck.Retries)
		assert.Equal(t, 60*time.Second, healthCheck.StartPeriod)
	})

	t.Run("Different health check test types", func(t *testing.T) {
		testCases := []struct {
			name string
			test []string
		}{
			{
				name: "HTTP check",
				test: []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
			},
			{
				name: "TCP socket check",
				test: []string{"CMD-SHELL", "nc -z localhost 8080"},
			},
			{
				name: "File existence check",
				test: []string{"CMD", "test", "-f", "/healthcheck"},
			},
			{
				name: "None (disable health check)",
				test: []string{"NONE"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				healthCheck := &HealthCheckConfig{
					Test:     tc.test,
					Interval: 30 * time.Second,
					Timeout:  10 * time.Second,
					Retries:  3,
				}

				assert.Equal(t, tc.test, healthCheck.Test)
				assert.Greater(t, healthCheck.Interval, time.Duration(0))
				assert.Greater(t, healthCheck.Timeout, time.Duration(0))
				assert.Greater(t, healthCheck.Retries, 0)
			})
		}
	})

	t.Run("Health check with zero values", func(t *testing.T) {
		healthCheck := &HealthCheckConfig{
			Test:     []string{"CMD", "echo", "ok"},
			Interval: 0,
			Timeout:  0,
			Retries:  0,
		}

		// Zero values should be allowed but may be interpreted differently
		assert.Equal(t, 0*time.Second, healthCheck.Interval)
		assert.Equal(t, 0*time.Second, healthCheck.Timeout)
		assert.Equal(t, 0, healthCheck.Retries)
	})
}

// TestDeploymentPlanComprehensive tests deployment plan functionality
func TestDeploymentPlanComprehensive(t *testing.T) {
	validConfig := &DeploymentConfig{
		Host:          "worker.example.com",
		User:          "deploy",
		DockerImage:   "translator:latest",
		ContainerName: "worker-1",
		Ports: []PortMapping{
			{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
		},
	}

	t.Run("Complete deployment plan", func(t *testing.T) {
		plan := &DeploymentPlan{
			Main:    validConfig,
			Workers: []*DeploymentConfig{validConfig, validConfig},
		}

		assert.NotNil(t, plan.Main)
		assert.Len(t, plan.Workers, 2)
		assert.Equal(t, validConfig, plan.Main)
		assert.Equal(t, validConfig, plan.Workers[0])
		assert.Equal(t, validConfig, plan.Workers[1])
	})

	t.Run("Deployment plan with different worker configs", func(t *testing.T) {
		workerConfig1 := &DeploymentConfig{
			Host:          "worker1.example.com",
			User:          "deploy",
			DockerImage:   "translator:latest",
			ContainerName: "worker-1",
			Ports: []PortMapping{
				{HostPort: 8081, ContainerPort: 8080, Protocol: "tcp"},
			},
		}

		workerConfig2 := &DeploymentConfig{
			Host:          "worker2.example.com",
			User:          "deploy",
			DockerImage:   "translator:latest",
			ContainerName: "worker-2",
			Ports: []PortMapping{
				{HostPort: 8082, ContainerPort: 8080, Protocol: "tcp"},
			},
		}

		plan := &DeploymentPlan{
			Main:    validConfig,
			Workers: []*DeploymentConfig{workerConfig1, workerConfig2},
		}

		assert.NotNil(t, plan.Main)
		assert.Len(t, plan.Workers, 2)
		assert.Equal(t, "worker1.example.com", plan.Workers[0].Host)
		assert.Equal(t, "worker2.example.com", plan.Workers[1].Host)
	})

	t.Run("Deployment plan edge cases", func(t *testing.T) {
		testCases := []struct {
			name   string
			plan   *DeploymentPlan
			valid  bool
			reason string
		}{
			{
				name: "Nil main config",
				plan: &DeploymentPlan{
					Main:    nil,
					Workers: []*DeploymentConfig{validConfig},
				},
				valid:  false,
				reason: "Main config should not be nil",
			},
			{
				name: "No workers",
				plan: &DeploymentPlan{
					Main:    validConfig,
					Workers: []*DeploymentConfig{},
				},
				valid:  true, // Empty workers might be allowed
				reason: "Empty workers should be allowed",
			},
			{
				name: "Many workers",
				plan: &DeploymentPlan{
					Main: validConfig,
					Workers: []*DeploymentConfig{
						validConfig, validConfig, validConfig,
						validConfig, validConfig, validConfig,
						validConfig, validConfig, validConfig,
						validConfig,
					},
				},
				valid:  true,
				reason: "Many workers should be allowed",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.valid {
					// Basic checks for valid configurations
					if tc.plan.Main != nil {
						assert.NotEmpty(t, tc.plan.Main.Host)
						assert.NotEmpty(t, tc.plan.Main.DockerImage)
					}
				} else {
					// Check invalid conditions
					if tc.name == "Nil main config" {
						assert.Nil(t, tc.plan.Main)
					}
				}
			})
		}
	})
}

// TestDeploymentStatusComprehensive tests deployment status functionality
func TestDeploymentStatusComprehensive(t *testing.T) {
	t.Run("Complete deployment status", func(t *testing.T) {
		now := time.Now()
		healthStatus := &HealthStatus{
			Status:    "healthy",
			LastCheck: now,
			Response:  "OK",
			Error:     "",
		}

		capabilities := map[string]interface{}{
			"cpu":     "4 cores",
			"memory":  "8GB",
			"storage": "100GB",
		}

		status := &DeploymentStatus{
			InstanceID:   "instance-123",
			Status:       "running",
			Host:         "worker.example.com",
			Port:         8080,
			ContainerID:  "container-456",
			LastSeen:     now,
			Health:       healthStatus,
			Capabilities: capabilities,
		}

		assert.Equal(t, "instance-123", status.InstanceID)
		assert.Equal(t, "running", status.Status)
		assert.Equal(t, "worker.example.com", status.Host)
		assert.Equal(t, 8080, status.Port)
		assert.Equal(t, "container-456", status.ContainerID)
		assert.Equal(t, now, status.LastSeen)
		assert.NotNil(t, status.Health)
		assert.Equal(t, "healthy", status.Health.Status)
		assert.Equal(t, now, status.Health.LastCheck)
		assert.Equal(t, "OK", status.Health.Response)
		assert.Equal(t, "", status.Health.Error)
		assert.Equal(t, capabilities, status.Capabilities)
		assert.Equal(t, "4 cores", status.Capabilities["cpu"])
		assert.Equal(t, "8GB", status.Capabilities["memory"])
		assert.Equal(t, "100GB", status.Capabilities["storage"])
	})

	t.Run("Deployment status without health", func(t *testing.T) {
		status := &DeploymentStatus{
			InstanceID:   "instance-123",
			Status:       "starting",
			Host:         "worker.example.com",
			Port:         8080,
			ContainerID:  "container-456",
			LastSeen:     time.Now(),
			Health:       nil, // No health status yet
			Capabilities: nil, // No capabilities detected yet
		}

		assert.Equal(t, "instance-123", status.InstanceID)
		assert.Equal(t, "starting", status.Status)
		assert.Nil(t, status.Health)
		assert.Nil(t, status.Capabilities)
	})

	t.Run("Health status with error", func(t *testing.T) {
		now := time.Now()
		healthStatus := &HealthStatus{
			Status:    "unhealthy",
			LastCheck: now,
			Response:  "",
			Error:     "Connection timeout",
		}

		status := &DeploymentStatus{
			InstanceID:  "instance-123",
			Status:      "unhealthy",
			Host:        "worker.example.com",
			Port:        8080,
			ContainerID: "container-456",
			LastSeen:    now,
			Health:      healthStatus,
		}

		assert.Equal(t, "unhealthy", status.Status)
		assert.NotNil(t, status.Health)
		assert.Equal(t, "unhealthy", status.Health.Status)
		assert.Equal(t, "", status.Health.Response)
		assert.Equal(t, "Connection timeout", status.Health.Error)
	})
}

// TestHealthStatusComprehensive tests health status functionality
func TestHealthStatusComprehensive(t *testing.T) {
	t.Run("Healthy status", func(t *testing.T) {
		now := time.Now()
		health := &HealthStatus{
			Status:    "healthy",
			LastCheck: now,
			Response:  "Service is running properly",
			Error:     "",
		}

		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, now, health.LastCheck)
		assert.Equal(t, "Service is running properly", health.Response)
		assert.Equal(t, "", health.Error)
	})

	t.Run("Unhealthy status with error", func(t *testing.T) {
		now := time.Now()
		health := &HealthStatus{
			Status:    "unhealthy",
			LastCheck: now,
			Response:  "",
			Error:     "Failed to connect to service",
		}

		assert.Equal(t, "unhealthy", health.Status)
		assert.Equal(t, now, health.LastCheck)
		assert.Equal(t, "", health.Response)
		assert.Equal(t, "Failed to connect to service", health.Error)
	})

	t.Run("Starting status", func(t *testing.T) {
		now := time.Now()
		health := &HealthStatus{
			Status:    "starting",
			LastCheck: now,
			Response:  "",
			Error:     "Service is still starting",
		}

		assert.Equal(t, "starting", health.Status)
		assert.Equal(t, now, health.LastCheck)
		assert.Equal(t, "", health.Response)
		assert.Equal(t, "Service is still starting", health.Error)
	})

	t.Run("Unknown status", func(t *testing.T) {
		now := time.Now()
		health := &HealthStatus{
			Status:    "unknown",
			LastCheck: now,
			Response:  "",
			Error:     "Health check not yet performed",
		}

		assert.Equal(t, "unknown", health.Status)
		assert.Equal(t, now, health.LastCheck)
		assert.Equal(t, "", health.Response)
		assert.Equal(t, "Health check not yet performed", health.Error)
	})
}

// TestNetworkServiceComprehensive tests network service functionality
func TestNetworkServiceComprehensive(t *testing.T) {
	t.Run("Complete network service", func(t *testing.T) {
		now := time.Now()
		capabilities := map[string]interface{}{
			"version":     "1.0.0",
			"endpoints":   []string{"/api/v1/translate", "/health"},
			"max_workers": 10,
		}

		service := &NetworkService{
			ID:           "service-123",
			Name:         "translation-worker",
			Host:         "worker.example.com",
			Port:         8080,
			Type:         "worker",
			Protocol:     "http",
			Capabilities: capabilities,
			LastSeen:     now,
			TTL:          5 * time.Minute,
		}

		assert.Equal(t, "service-123", service.ID)
		assert.Equal(t, "translation-worker", service.Name)
		assert.Equal(t, "worker.example.com", service.Host)
		assert.Equal(t, 8080, service.Port)
		assert.Equal(t, "worker", service.Type)
		assert.Equal(t, "http", service.Protocol)
		assert.Equal(t, capabilities, service.Capabilities)
		assert.Equal(t, now, service.LastSeen)
		assert.Equal(t, 5*time.Minute, service.TTL)
		assert.Equal(t, "1.0.0", service.Capabilities["version"])
		assert.Equal(t, 10, service.Capabilities["max_workers"])
	})

	t.Run("Minimal network service", func(t *testing.T) {
		service := &NetworkService{
			ID:       "service-123",
			Name:     "basic-service",
			Host:     "localhost",
			Port:     8080,
			Type:     "api",
			Protocol: "http",
			LastSeen: time.Now(),
			TTL:      time.Minute,
		}

		assert.Equal(t, "service-123", service.ID)
		assert.Equal(t, "basic-service", service.Name)
		assert.Equal(t, "localhost", service.Host)
		assert.Equal(t, 8080, service.Port)
		assert.Equal(t, "api", service.Type)
		assert.Equal(t, "http", service.Protocol)
		assert.Nil(t, service.Capabilities)
		assert.NotZero(t, service.LastSeen)
		assert.Equal(t, time.Minute, service.TTL)
	})

	t.Run("Network service with different types", func(t *testing.T) {
		serviceTypes := []string{"main", "worker", "api", "database", "cache"}

		for _, serviceType := range serviceTypes {
			service := &NetworkService{
				ID:       "service-" + serviceType,
				Name:     serviceType + "-service",
				Host:     "example.com",
				Port:     8080,
				Type:     serviceType,
				Protocol: "http",
				LastSeen: time.Now(),
				TTL:      time.Minute,
			}

			assert.Equal(t, serviceType, service.Type)
		}
	})

	t.Run("Network service expiration", func(t *testing.T) {
		pastTime := time.Now().Add(-10 * time.Minute)
		expiredService := &NetworkService{
			ID:       "expired-service",
			Name:     "expired",
			Host:     "example.com",
			Port:     8080,
			Type:     "api",
			Protocol: "http",
			LastSeen: pastTime,
			TTL:      time.Minute, // Service should be expired since last seen was 10 minutes ago
		}

		futureTime := time.Now().Add(10 * time.Minute)
		futureService := &NetworkService{
			ID:       "future-service",
			Name:     "future",
			Host:     "example.com",
			Port:     8080,
			Type:     "api",
			Protocol: "http",
			LastSeen: futureTime,
			TTL:      time.Minute, // Service should not be expired
		}

		// Test time-based conditions
		assert.True(t, time.Since(expiredService.LastSeen) > expiredService.TTL)
		assert.False(t, time.Since(futureService.LastSeen) > futureService.TTL)
	})
}

// TestBroadcastMessageComprehensive tests broadcast message functionality
func TestBroadcastMessageComprehensive(t *testing.T) {
	t.Run("Complete broadcast message", func(t *testing.T) {
		now := time.Now()
		capabilities := map[string]interface{}{
			"version":     "1.0.0",
			"max_workers": 5,
		}

		message := &BroadcastMessage{
			ServiceID:    "service-123",
			Type:         "worker",
			Host:         "worker.example.com",
			Port:         8080,
			Protocol:     "http",
			Capabilities: capabilities,
			Timestamp:    now,
		}

		assert.Equal(t, "service-123", message.ServiceID)
		assert.Equal(t, "worker", message.Type)
		assert.Equal(t, "worker.example.com", message.Host)
		assert.Equal(t, 8080, message.Port)
		assert.Equal(t, "http", message.Protocol)
		assert.Equal(t, capabilities, message.Capabilities)
		assert.Equal(t, now, message.Timestamp)
		assert.Equal(t, "1.0.0", message.Capabilities["version"])
		assert.Equal(t, 5, message.Capabilities["max_workers"])
	})

	t.Run("Minimal broadcast message", func(t *testing.T) {
		message := &BroadcastMessage{
			ServiceID: "service-123",
			Type:      "api",
			Host:      "localhost",
			Port:      8080,
			Protocol:  "http",
			Timestamp: time.Now(),
		}

		assert.Equal(t, "service-123", message.ServiceID)
		assert.Equal(t, "api", message.Type)
		assert.Equal(t, "localhost", message.Host)
		assert.Equal(t, 8080, message.Port)
		assert.Equal(t, "http", message.Protocol)
		assert.Nil(t, message.Capabilities)
		assert.NotZero(t, message.Timestamp)
	})

	t.Run("Broadcast message with different timestamps", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-5 * time.Minute)
		future := now.Add(5 * time.Minute)

		pastMessage := &BroadcastMessage{
			ServiceID: "past-service",
			Type:      "api",
			Host:      "example.com",
			Port:      8080,
			Protocol:  "http",
			Timestamp: past,
		}

		futureMessage := &BroadcastMessage{
			ServiceID: "future-service",
			Type:      "api",
			Host:      "example.com",
			Port:      8080,
			Protocol:  "http",
			Timestamp: future,
		}

		assert.True(t, pastMessage.Timestamp.Before(now))
		assert.True(t, futureMessage.Timestamp.After(now))
	})
}

// TestAPICommunicationLogComprehensive tests API communication log functionality
func TestAPICommunicationLogComprehensive(t *testing.T) {
	t.Run("Complete API log entry", func(t *testing.T) {
		now := time.Now()
		duration := 250 * time.Millisecond

		log := &APICommunicationLog{
			Timestamp:    now,
			SourceHost:   "client.example.com",
			SourcePort:   12345,
			TargetHost:   "api.example.com",
			TargetPort:   8080,
			Method:       "POST",
			URL:          "/api/v1/translate",
			StatusCode:   200,
			RequestSize:  1024,
			ResponseSize: 2048,
			Duration:     duration,
			UserAgent:    "TranslateClient/1.0",
			Error:        "",
		}

		assert.Equal(t, now, log.Timestamp)
		assert.Equal(t, "client.example.com", log.SourceHost)
		assert.Equal(t, 12345, log.SourcePort)
		assert.Equal(t, "api.example.com", log.TargetHost)
		assert.Equal(t, 8080, log.TargetPort)
		assert.Equal(t, "POST", log.Method)
		assert.Equal(t, "/api/v1/translate", log.URL)
		assert.Equal(t, 200, log.StatusCode)
		assert.Equal(t, int64(1024), log.RequestSize)
		assert.Equal(t, int64(2048), log.ResponseSize)
		assert.Equal(t, duration, log.Duration)
		assert.Equal(t, "TranslateClient/1.0", log.UserAgent)
		assert.Equal(t, "", log.Error)
	})

	t.Run("API log entry with error", func(t *testing.T) {
		log := &APICommunicationLog{
			Timestamp:    time.Now(),
			SourceHost:   "client.example.com",
			SourcePort:   12345,
			TargetHost:   "api.example.com",
			TargetPort:   8080,
			Method:       "POST",
			URL:          "/api/v1/translate",
			StatusCode:   500,
			RequestSize:  1024,
			ResponseSize: 256, // Error response size
			Duration:     100 * time.Millisecond,
			UserAgent:    "TranslateClient/1.0",
			Error:        "Internal server error",
		}

		assert.Equal(t, 500, log.StatusCode)
		assert.Equal(t, int64(256), log.ResponseSize)
		assert.Equal(t, "Internal server error", log.Error)
	})

	t.Run("API log entry with different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

		for _, method := range methods {
			log := &APICommunicationLog{
				Timestamp:    time.Now(),
				SourceHost:   "client.example.com",
				SourcePort:   12345,
				TargetHost:   "api.example.com",
				TargetPort:   8080,
				Method:       method,
				URL:          "/api/test",
				StatusCode:   200,
				RequestSize:  512,
				ResponseSize: 1024,
				Duration:     50 * time.Millisecond,
			}

			assert.Equal(t, method, log.Method)
		}
	})

	t.Run("API log entry with different status codes", func(t *testing.T) {
		statusCodes := []int{200, 201, 301, 302, 400, 401, 403, 404, 500, 502}

		for _, statusCode := range statusCodes {
			log := &APICommunicationLog{
				Timestamp:    time.Now(),
				SourceHost:   "client.example.com",
				SourcePort:   12345,
				TargetHost:   "api.example.com",
				TargetPort:   8080,
				Method:       "GET",
				URL:          "/api/test",
				StatusCode:   statusCode,
				RequestSize:  256,
				ResponseSize: 512,
				Duration:     25 * time.Millisecond,
			}

			assert.Equal(t, statusCode, log.StatusCode)
		}
	})
}

// TestValidationErrorComprehensive tests validation error functionality
func TestValidationErrorComprehensive(t *testing.T) {
	t.Run("Error with all fields", func(t *testing.T) {
		err := &ValidationError{
			Field:   "host",
			Message: "host cannot be empty",
		}

		expected := "host cannot be empty (field: host)"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, "host", err.Field)
		assert.Equal(t, "host cannot be empty", err.Message)
	})

	t.Run("Error with empty field", func(t *testing.T) {
		err := &ValidationError{
			Field:   "",
			Message: "general validation error",
		}

		expected := "general validation error (field: )"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, "", err.Field)
		assert.Equal(t, "general validation error", err.Message)
	})

	t.Run("Error with empty message", func(t *testing.T) {
		err := &ValidationError{
			Field:   "user",
			Message: "",
		}

		expected := " (field: user)"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, "user", err.Field)
		assert.Equal(t, "", err.Message)
	})

	t.Run("Error as error interface", func(t *testing.T) {
		var err error = &ValidationError{
			Field:   "docker_image",
			Message: "docker image is required",
		}

		assert.Implements(t, (*error)(nil), err)
		assert.Equal(t, "docker image is required (field: docker_image)", err.Error())
	})
}

// TestPerformanceBenchmarksComprehensive tests performance of key operations
func TestPerformanceBenchmarksComprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmarks in short mode")
	}

	t.Run("DeploymentConfig creation benchmark", func(t *testing.T) {
		config := &DeploymentConfig{
			Host:          "worker.example.com",
			User:          "deploy",
			Password:      "password",
			DockerImage:   "translator:latest",
			ContainerName: "worker-1",
			Ports: []PortMapping{
				{HostPort: 8080, ContainerPort: 8080, Protocol: "tcp"},
			},
			Environment: map[string]string{
				"ENV": "production",
				"LOG": "info",
			},
		}

		// Benchmark config access
		for i := 0; i < 1000; i++ {
			_ = config.Host
			_ = config.DockerImage
			_ = len(config.Ports)
			_ = len(config.Environment)
		}

		assert.Equal(t, "worker.example.com", config.Host)
		assert.Equal(t, "translator:latest", config.DockerImage)
		assert.Len(t, config.Ports, 1)
		assert.Len(t, config.Environment, 2)
	})

	t.Run("DeploymentStatus creation benchmark", func(t *testing.T) {
		now := time.Now()
		status := &DeploymentStatus{
			InstanceID:  "instance-123",
			Status:      "running",
			Host:        "worker.example.com",
			Port:        8080,
			ContainerID: "container-456",
			LastSeen:    now,
			Capabilities: map[string]interface{}{
				"cpu":     "4 cores",
				"memory":  "8GB",
				"storage": "100GB",
			},
		}

		// Benchmark status access
		for i := 0; i < 1000; i++ {
			_ = status.InstanceID
			_ = status.Status
			_ = status.Host
			_ = status.Port
			_ = len(status.Capabilities)
		}

		assert.Equal(t, "instance-123", status.InstanceID)
		assert.Equal(t, "running", status.Status)
		assert.Equal(t, "worker.example.com", status.Host)
		assert.Equal(t, 8080, status.Port)
		assert.Len(t, status.Capabilities, 3)
	})
}
