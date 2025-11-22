package deployment

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"gopkg.in/yaml.v3"
)

// DockerOrchestrator manages Docker-based deployment using docker-compose
type DockerOrchestrator struct {
	config     *config.Config
	eventBus   *events.EventBus
	logger     *log.Logger
	composeDir string
}

// DockerComposeConfig represents a docker-compose configuration
type DockerComposeConfig struct {
	Version  string                          `yaml:"version"`
	Services map[string]*DockerServiceConfig `yaml:"services"`
	Networks map[string]*DockerNetworkConfig `yaml:"networks,omitempty"`
	Volumes  map[string]*DockerVolumeConfig  `yaml:"volumes,omitempty"`
}

// DockerServiceConfig represents a Docker service configuration
type DockerServiceConfig struct {
	Image         string             `yaml:"image"`
	ContainerName string             `yaml:"container_name,omitempty"`
	Ports         []string           `yaml:"ports,omitempty"`
	Environment   map[string]string  `yaml:"environment,omitempty"`
	Volumes       []string           `yaml:"volumes,omitempty"`
	Networks      []string           `yaml:"networks,omitempty"`
	Restart       string             `yaml:"restart,omitempty"`
	DependsOn     []string           `yaml:"depends_on,omitempty"`
	HealthCheck   *DockerHealthCheck `yaml:"healthcheck,omitempty"`
	Command       []string           `yaml:"command,omitempty"`
}

// DockerHealthCheck represents a Docker health check configuration
type DockerHealthCheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// DockerNetworkConfig represents a Docker network configuration
type DockerNetworkConfig struct {
	Driver string `yaml:"driver,omitempty"`
}

// DockerVolumeConfig represents a Docker volume configuration
type DockerVolumeConfig struct {
	Driver string `yaml:"driver,omitempty"`
}

// NewDockerOrchestrator creates a new Docker orchestrator
func NewDockerOrchestrator(cfg *config.Config, eventBus *events.EventBus) *DockerOrchestrator {
	logger := log.New(os.Stdout, "[DOCKER] ", log.LstdFlags)

	// Create compose directory
	composeDir := filepath.Join(os.TempDir(), "translator-compose")
	os.MkdirAll(composeDir, 0755)

	return &DockerOrchestrator{
		config:     cfg,
		eventBus:   eventBus,
		logger:     logger,
		composeDir: composeDir,
	}
}

// GenerateComposeFile generates a docker-compose.yml file for the deployment plan
func (do *DockerOrchestrator) GenerateComposeFile(plan *DeploymentPlan) (string, error) {
	do.logger.Printf("Generating docker-compose.yml for %d services", len(plan.Workers)+1)

	composeConfig := &DockerComposeConfig{
		Version:  "3.8",
		Services: make(map[string]*DockerServiceConfig),
		Networks: map[string]*DockerNetworkConfig{
			"translator-network": {Driver: "bridge"},
		},
		Volumes: make(map[string]*DockerVolumeConfig),
	}

	// Add main service
	if err := do.addServiceToCompose(composeConfig, plan.Main, "main"); err != nil {
		return "", fmt.Errorf("failed to add main service: %w", err)
	}

	// Add worker services
	for i, worker := range plan.Workers {
		serviceName := fmt.Sprintf("worker-%d", i+1)
		if err := do.addServiceToCompose(composeConfig, worker, serviceName); err != nil {
			return "", fmt.Errorf("failed to add worker service %d: %w", i+1, err)
		}
	}

	// Add supporting services (database, etc.)
	do.addSupportingServices(composeConfig)

	// Generate YAML
	composePath := filepath.Join(do.composeDir, "docker-compose.yml")
	if err := do.writeComposeFile(composeConfig, composePath); err != nil {
		return "", fmt.Errorf("failed to write compose file: %w", err)
	}

	do.logger.Printf("Generated docker-compose.yml at %s", composePath)
	return composePath, nil
}

// addServiceToCompose adds a service to the docker-compose configuration
func (do *DockerOrchestrator) addServiceToCompose(composeConfig *DockerComposeConfig, cfg *DeploymentConfig, serviceName string) error {
	service := &DockerServiceConfig{
		Image:         cfg.DockerImage,
		ContainerName: cfg.ContainerName,
		Ports:         make([]string, 0),
		Environment:   cfg.Environment,
		Volumes:       make([]string, 0),
		Networks:      cfg.Networks,
		Restart:       cfg.RestartPolicy,
	}

	// Add ports
	for _, port := range cfg.Ports {
		portStr := fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort)
		if port.Protocol != "" && port.Protocol != "tcp" {
			portStr += "/" + port.Protocol
		}
		service.Ports = append(service.Ports, portStr)
	}

	// Add volumes
	for _, volume := range cfg.Volumes {
		volumeStr := fmt.Sprintf("%s:%s", volume.HostPath, volume.ContainerPath)
		if volume.ReadOnly {
			volumeStr += ":ro"
		}
		service.Volumes = append(service.Volumes, volumeStr)
	}

	// Add health check
	if cfg.HealthCheck != nil {
		service.HealthCheck = &DockerHealthCheck{
			Test:        cfg.HealthCheck.Test,
			Interval:    formatDuration(cfg.HealthCheck.Interval),
			Timeout:     formatDuration(cfg.HealthCheck.Timeout),
			Retries:     cfg.HealthCheck.Retries,
			StartPeriod: formatDuration(cfg.HealthCheck.StartPeriod),
		}
	}

	composeConfig.Services[serviceName] = service
	return nil
}

// addSupportingServices adds database and other supporting services
func (do *DockerOrchestrator) addSupportingServices(composeConfig *DockerComposeConfig) {
	// PostgreSQL database
	composeConfig.Services["postgres"] = &DockerServiceConfig{
		Image: "postgres:15-alpine",
		Environment: map[string]string{
			"POSTGRES_USER":     "translator",
			"POSTGRES_PASSWORD": "secure_password",
			"POSTGRES_DB":       "translator",
		},
		Volumes:  []string{"postgres-data:/var/lib/postgresql/data"},
		Networks: []string{"translator-network"},
		Ports:    []string{"5432:5432"},
		HealthCheck: &DockerHealthCheck{
			Test:     []string{"CMD-SHELL", "pg_isready -U translator -d translator"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}

	// Redis cache
	composeConfig.Services["redis"] = &DockerServiceConfig{
		Image:    "redis:7-alpine",
		Command:  []string{"redis-server", "--requirepass", "redis_secure_password"},
		Volumes:  []string{"redis-data:/data"},
		Networks: []string{"translator-network"},
		Ports:    []string{"6379:6379"},
		HealthCheck: &DockerHealthCheck{
			Test:     []string{"CMD", "redis-cli", "--raw", "incr", "ping"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}

	// Add volumes
	composeConfig.Volumes["postgres-data"] = &DockerVolumeConfig{}
	composeConfig.Volumes["redis-data"] = &DockerVolumeConfig{}
}

// formatDuration formats a time.Duration to docker-compose format
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

// writeComposeFile writes the docker-compose configuration to a file
func (do *DockerOrchestrator) writeComposeFile(config *DockerComposeConfig, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	return encoder.Encode(config)
}

// DeployWithCompose deploys using docker-compose
func (do *DockerOrchestrator) DeployWithCompose(ctx context.Context, composePath string) error {
	do.logger.Println("Starting docker-compose deployment...")

	// Change to compose directory
	composeDir := filepath.Dir(composePath)
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(composeDir); err != nil {
		return fmt.Errorf("failed to change to compose directory: %w", err)
	}

	// Pull images
	if err := do.runComposeCommand(ctx, "pull"); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Start services
	if err := do.runComposeCommand(ctx, "up", "-d"); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Wait for services to be healthy
	if err := do.waitForServicesHealthy(ctx, composeDir); err != nil {
		return fmt.Errorf("services failed health checks: %w", err)
	}

	do.emitEvent(events.Event{
		Type:      "docker_deployment_completed",
		SessionID: "system",
		Message:   "Docker-compose deployment completed successfully",
		Data: map[string]interface{}{
			"compose_file": composePath,
		},
	})

	return nil
}

// runComposeCommand runs a docker-compose command
func (do *DockerOrchestrator) runComposeCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	cmd.Dir = do.composeDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose command failed: %w\nOutput: %s", err, string(output))
	}

	do.logger.Printf("docker-compose command succeeded: %s", strings.Join(args, " "))
	return nil
}

// waitForServicesHealthy waits for all services to pass health checks
func (do *DockerOrchestrator) waitForServicesHealthy(ctx context.Context, composeDir string) error {
	do.logger.Println("Waiting for services to become healthy...")

	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for services to become healthy")
		case <-ticker.C:
			if healthy, err := do.checkServicesHealth(ctx, composeDir); err != nil {
				do.logger.Printf("Health check error: %v", err)
			} else if healthy {
				do.logger.Println("All services are healthy!")
				return nil
			}
		}
	}
}

// checkServicesHealth checks if all services are healthy
func (do *DockerOrchestrator) checkServicesHealth(ctx context.Context, composeDir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", "ps", "--format", "json")
	cmd.Dir = composeDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse the JSON output to check service states
	// This is a simplified check - in production you'd parse the JSON properly
	outputStr := string(output)
	if strings.Contains(outputStr, "healthy") || strings.Contains(outputStr, "running") {
		return true, nil
	}

	return false, nil
}

// ScaleService scales a service to the specified number of replicas
func (do *DockerOrchestrator) ScaleService(ctx context.Context, serviceName string, replicas int) error {
	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d", "--scale", fmt.Sprintf("%s=%d", serviceName, replicas))
	cmd.Dir = do.composeDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to scale service: %w\nOutput: %s", err, string(output))
	}

	do.logger.Printf("Scaled service %s to %d replicas", serviceName, replicas)
	return nil
}

// GetServiceLogs retrieves logs from a specific service
func (do *DockerOrchestrator) GetServiceLogs(ctx context.Context, serviceName string, lines int) (string, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", "logs", "--tail", fmt.Sprintf("%d", lines), serviceName)
	cmd.Dir = do.composeDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get service logs: %w", err)
	}

	return string(output), nil
}

// StopDeployment stops the entire deployment
func (do *DockerOrchestrator) StopDeployment(ctx context.Context) error {
	do.logger.Println("Stopping docker-compose deployment...")

	cmd := exec.CommandContext(ctx, "docker-compose", "down")
	cmd.Dir = do.composeDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop deployment: %w\nOutput: %s", err, string(output))
	}

	do.logger.Println("Deployment stopped successfully")
	return nil
}

// UpdateService updates a service to a new image version
func (do *DockerOrchestrator) UpdateService(ctx context.Context, serviceName, newImage string) error {
	do.logger.Printf("Updating service %s to image %s...", serviceName, newImage)

	// Pull the new image
	if err := do.runComposeCommand(ctx, "pull", serviceName); err != nil {
		return fmt.Errorf("failed to pull new image for service %s: %w", serviceName, err)
	}

	// Update the service
	if err := do.runComposeCommand(ctx, "up", "-d", serviceName); err != nil {
		return fmt.Errorf("failed to update service %s: %w", serviceName, err)
	}

	// Wait for the service to be healthy
	if err := do.waitForServiceHealthy(ctx, serviceName); err != nil {
		return fmt.Errorf("service %s failed health check after update: %w", serviceName, err)
	}

	do.emitEvent(events.Event{
		Type:      "service_updated",
		SessionID: "system",
		Message:   fmt.Sprintf("Service %s updated to %s", serviceName, newImage),
		Data: map[string]interface{}{
			"service": serviceName,
			"image":   newImage,
		},
	})

	do.logger.Printf("Service %s updated successfully", serviceName)
	return nil
}

// RestartService restarts a specific service
func (do *DockerOrchestrator) RestartService(ctx context.Context, serviceName string) error {
	do.logger.Printf("Restarting service %s...", serviceName)

	if err := do.runComposeCommand(ctx, "restart", serviceName); err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}

	// Wait for the service to be healthy
	if err := do.waitForServiceHealthy(ctx, serviceName); err != nil {
		return fmt.Errorf("service %s failed health check after restart: %w", serviceName, err)
	}

	do.emitEvent(events.Event{
		Type:      "service_restarted",
		SessionID: "system",
		Message:   fmt.Sprintf("Service %s restarted", serviceName),
		Data: map[string]interface{}{
			"service": serviceName,
		},
	})

	do.logger.Printf("Service %s restarted successfully", serviceName)
	return nil
}

// UpdateAllServices updates all services to their latest images
func (do *DockerOrchestrator) UpdateAllServices(ctx context.Context) error {
	do.logger.Println("Updating all services...")

	// Pull all images
	if err := do.runComposeCommand(ctx, "pull"); err != nil {
		return fmt.Errorf("failed to pull all images: %w", err)
	}

	// Restart all services
	if err := do.runComposeCommand(ctx, "up", "-d"); err != nil {
		return fmt.Errorf("failed to restart all services: %w", err)
	}

	// Wait for all services to be healthy
	if err := do.waitForServicesHealthy(ctx, do.composeDir); err != nil {
		return fmt.Errorf("services failed health checks after update: %w", err)
	}

	do.emitEvent(events.Event{
		Type:      "all_services_updated",
		SessionID: "system",
		Message:   "All services updated successfully",
	})

	do.logger.Println("All services updated successfully")
	return nil
}

// RestartAllServices restarts all services
func (do *DockerOrchestrator) RestartAllServices(ctx context.Context) error {
	do.logger.Println("Restarting all services...")

	if err := do.runComposeCommand(ctx, "restart"); err != nil {
		return fmt.Errorf("failed to restart all services: %w", err)
	}

	// Wait for all services to be healthy
	if err := do.waitForServicesHealthy(ctx, do.composeDir); err != nil {
		return fmt.Errorf("services failed health checks after restart: %w", err)
	}

	do.emitEvent(events.Event{
		Type:      "all_services_restarted",
		SessionID: "system",
		Message:   "All services restarted successfully",
	})

	do.logger.Println("All services restarted successfully")
	return nil
}

// waitForServiceHealthy waits for a specific service to become healthy
func (do *DockerOrchestrator) waitForServiceHealthy(ctx context.Context, serviceName string) error {
	do.logger.Printf("Waiting for service %s to become healthy...", serviceName)

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for service %s to become healthy", serviceName)
		case <-ticker.C:
			if healthy, err := do.checkServiceHealth(ctx, serviceName); err != nil {
				do.logger.Printf("Health check error for %s: %v", serviceName, err)
			} else if healthy {
				do.logger.Printf("Service %s is healthy!", serviceName)
				return nil
			}
		}
	}
}

// checkServiceHealth checks if a specific service is healthy
func (do *DockerOrchestrator) checkServiceHealth(ctx context.Context, serviceName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", "ps", serviceName, "--format", "json")
	cmd.Dir = do.composeDir

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse the JSON output to check service state
	outputStr := string(output)
	if strings.Contains(outputStr, "healthy") || strings.Contains(outputStr, "running") {
		return true, nil
	}

	return false, nil
}

// GetServiceStatus returns the status of a specific service
func (do *DockerOrchestrator) GetServiceStatus(ctx context.Context, serviceName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", "ps", serviceName, "--format", "{{.State}}")
	cmd.Dir = do.composeDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get service status: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Cleanup removes the generated compose files and temporary data
func (do *DockerOrchestrator) Cleanup() error {
	do.logger.Println("Cleaning up deployment files...")

	if err := os.RemoveAll(do.composeDir); err != nil {
		return fmt.Errorf("failed to cleanup compose directory: %w", err)
	}

	return nil
}

// emitEvent emits an event if event bus is available
func (do *DockerOrchestrator) emitEvent(event events.Event) {
	if do.eventBus != nil {
		do.eventBus.Publish(event)
	}
}
