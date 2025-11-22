package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
)

func main() {
	var (
		configFile = flag.String("config", "config.distributed.json", "Configuration file")
		action     = flag.String("action", "deploy", "Action: deploy, status, stop, cleanup, update, restart, generate-plan")
		service    = flag.String("service", "", "Service name for update/restart actions")
		image      = flag.String("image", "", "New image for update action")
		planFile   = flag.String("plan", "", "Deployment plan JSON file")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Create event bus
	eventBus := events.NewEventBus()

	// Create deployment orchestrator
	orchestrator := deployment.NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	// Handle actions
	switch *action {
	case "deploy":
		if *planFile == "" {
			log.Fatal("Deployment plan file is required for deploy action")
		}
		handleDeploy(orchestrator, *planFile)

	case "status":
		handleStatus(orchestrator)

	case "stop":
		handleStop(orchestrator)

	case "cleanup":
		handleCleanup(orchestrator)

	case "update":
		handleUpdate(orchestrator, *service, *image)

	case "restart":
		handleRestart(orchestrator, *service)

	case "generate-plan":
		handleGeneratePlan(cfg)

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

func handleDeploy(orchestrator *deployment.DeploymentOrchestrator, planFile string) {
	log.Println("Starting deployment...")

	// Load deployment plan
	plan, err := loadDeploymentPlan(planFile)
	if err != nil {
		log.Fatalf("Failed to load deployment plan: %v", err)
	}

	// Execute deployment
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := orchestrator.DeployDistributedSystem(ctx, plan); err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	log.Println("Deployment completed successfully!")
}

func handleStatus(orchestrator *deployment.DeploymentOrchestrator) {
	instances := orchestrator.GetDeployedInstances()

	fmt.Println("=== Deployment Status ===")
	fmt.Printf("Total instances: %d\n\n", len(instances))

	for id, instance := range instances {
		fmt.Printf("Instance: %s\n", id)
		fmt.Printf("  Host: %s:%d\n", instance.Host, instance.Port)
		fmt.Printf("  Container ID: %s\n", instance.ContainerID[:12])
		fmt.Printf("  Status: %s\n", instance.Status)
		fmt.Printf("  Last Seen: %s\n", instance.LastSeen.Format(time.RFC3339))
		fmt.Println()
	}
}

func handleStop(orchestrator *deployment.DeploymentOrchestrator) {
	log.Println("Stopping deployment...")

	// For now, this would need to be implemented in the orchestrator
	// orchestrator.StopDeployment(context.Background())

	log.Println("Deployment stopped")
}

func handleCleanup(orchestrator *deployment.DeploymentOrchestrator) {
	log.Println("Cleaning up deployment...")

	// Cleanup would be implemented in orchestrator
	// orchestrator.Cleanup()

	log.Println("Cleanup completed")
}

func handleUpdate(orchestrator *deployment.DeploymentOrchestrator, service, image string) {
	if service == "" {
		log.Fatal("Service name is required for update action")
	}

	log.Printf("Updating service %s...", service)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if image != "" {
		// Update specific service to new image
		if err := orchestrator.UpdateService(ctx, service, image); err != nil {
			log.Fatalf("Update failed: %v", err)
		}
	} else {
		// Update all services
		if err := orchestrator.UpdateAllServices(ctx); err != nil {
			log.Fatalf("Update failed: %v", err)
		}
	}

	log.Println("Update completed successfully!")
}

func handleRestart(orchestrator *deployment.DeploymentOrchestrator, service string) {
	log.Printf("Restarting service %s...", service)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if service != "" {
		// Restart specific service
		if err := orchestrator.RestartService(ctx, service); err != nil {
			log.Fatalf("Restart failed: %v", err)
		}
	} else {
		// Restart all services
		if err := orchestrator.RestartAllServices(ctx); err != nil {
			log.Fatalf("Restart failed: %v", err)
		}
	}

	log.Println("Restart completed successfully!")
}

func handleGeneratePlan(cfg *config.Config) {
	log.Println("Generating deployment plan...")

	plan := generateDefaultPlan(cfg)

	// Write plan to file
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal plan: %v", err)
	}

	if err := os.WriteFile("deployment-plan.json", data, 0644); err != nil {
		log.Fatalf("Failed to write plan file: %v", err)
	}

	log.Println("Deployment plan generated: deployment-plan.json")
}

func loadDeploymentPlan(filename string) (*deployment.DeploymentPlan, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var plan deployment.DeploymentPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

func generateDefaultPlan(cfg *config.Config) *deployment.DeploymentPlan {
	plan := &deployment.DeploymentPlan{
		Main: &deployment.DeploymentConfig{
			Host:          "localhost",
			User:          "translator",
			SSHKeyPath:    "/path/to/ssh/key",
			DockerImage:   "translator:latest",
			ContainerName: "translator-main",
			Ports: []deployment.PortMapping{
				{HostPort: 8443, ContainerPort: 8443, Protocol: "tcp"},
			},
			Environment: map[string]string{
				"JWT_SECRET": "main-server-secret",
			},
			Volumes: []deployment.VolumeMapping{
				{HostPath: "./certs", ContainerPath: "/app/certs", ReadOnly: true},
				{HostPath: "./config.distributed.json", ContainerPath: "/app/config.json", ReadOnly: true},
			},
			Networks:      []string{"translator-network"},
			RestartPolicy: "unless-stopped",
			HealthCheck: &deployment.HealthCheckConfig{
				Test:     []string{"CMD", "curl", "-f", "https://localhost:8443/health"},
				Interval: 30 * time.Second,
				Timeout:  10 * time.Second,
				Retries:  3,
			},
		},
		Workers: []*deployment.DeploymentConfig{},
	}

	// Add workers based on configuration
	workerIndex := 1
	for workerID, worker := range cfg.Distributed.Workers {
		workerConfig := &deployment.DeploymentConfig{
			Host:          worker.Host,
			User:          worker.User,
			SSHKeyPath:    worker.KeyFile,
			DockerImage:   "translator:latest",
			ContainerName: fmt.Sprintf("translator-worker-%s", workerID),
			Ports: []deployment.PortMapping{
				{HostPort: 8443 + workerIndex, ContainerPort: 8443, Protocol: "tcp"},
			},
			Environment: map[string]string{
				"JWT_SECRET":   fmt.Sprintf("worker-%s-secret", workerID),
				"WORKER_INDEX": fmt.Sprintf("%d", workerIndex),
			},
			Volumes: []deployment.VolumeMapping{
				{HostPath: "./certs", ContainerPath: "/app/certs", ReadOnly: true},
				{HostPath: "./config.worker.json", ContainerPath: "/app/config.json", ReadOnly: true},
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
		plan.Workers = append(plan.Workers, workerConfig)
		workerIndex++
	}

	return plan
}
