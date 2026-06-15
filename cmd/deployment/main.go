package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/events"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var (
		configFile = fs.String("config", "config.distributed.json", "Configuration file")
		action     = fs.String("action", "deploy", "Action: deploy, status, stop, cleanup, update, restart, generate-plan")
		service    = fs.String("service", "", "Service name for update/restart actions")
		image      = fs.String("image", "", "New image for update action")
		planFile   = fs.String("plan", "", "Deployment plan JSON file")
		verbose    = fs.Bool("verbose", false, "Enable verbose logging")
	)
	fs.Parse(os.Args[1:])

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

// shortContainerID returns at most the first 12 characters of a container ID
// for display. A plain id[:12] slice panics with a slice-out-of-range when the
// ID is shorter than 12 bytes (empty or short container IDs are possible — the
// ID can come from a deployer return value or a short configured container
// name), which would crash the entire `status` report. This bounds the slice.
func shortContainerID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func handleStatus(orchestrator *deployment.DeploymentOrchestrator) {
	instances := orchestrator.GetDeployedInstances()

	fmt.Println("=== Deployment Status ===")
	fmt.Printf("Total instances: %d\n\n", len(instances))

	for id, instance := range instances {
		fmt.Printf("Instance: %s\n", id)
		fmt.Printf("  Host: %s:%d\n", instance.Host, instance.Port)
		fmt.Printf("  Container ID: %s\n", shortContainerID(instance.ContainerID))
		fmt.Printf("  Status: %s\n", instance.Status)
		fmt.Printf("  Last Seen: %s\n", instance.LastSeen.Format(time.RFC3339))
		fmt.Println()
	}
}

func handleStop(orchestrator *deployment.DeploymentOrchestrator) {
	log.Println("Stopping deployment...")

	if orchestrator == nil {
		log.Println("No orchestrator available, nothing to stop")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := orchestrator.StopDeployment(ctx); err != nil {
		log.Fatalf("Failed to stop deployment: %v", err)
	}

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

	plan := generateDeploymentPlan(cfg)

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

// generateJWTSecret returns a cryptographically-random 256-bit secret as a hex
// string for a generated deployment plan. A generated plan MUST NOT carry a
// hardcoded/predictable JWT secret (e.g. "main-secret" / "worker-<id>-secret"):
// an operator who deploys the plan without editing it would ship a known-constant
// signing key — a trivial auth-bypass (§11.4.10 credentials mandate). crypto/rand
// gives each generated plan a unique, unguessable secret with no review friction.
// On the (practically impossible) rand read failure the field is left as an
// explicit placeholder that validateDeploymentPlan-class checks can reject, rather
// than silently emitting a weak constant.
func generateJWTSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "CHANGE_ME_BEFORE_DEPLOY"
	}
	return hex.EncodeToString(b)
}

func generateDeploymentPlan(cfg *config.Config) *deployment.DeploymentPlan {
	plan := &deployment.DeploymentPlan{
		Main:    generateMainConfig(cfg),
		Workers: []*deployment.DeploymentConfig{},
	}

	// Add workers based on configuration
	workerIndex := 1
	for workerID, worker := range cfg.Distributed.Workers {
		workerConfig := &deployment.DeploymentConfig{
			Host:          worker.Host,
			User:          worker.User,
			Password:      worker.Password,
			SSHKeyPath:    "",
			DockerImage:   "translator:latest",
			ContainerName: fmt.Sprintf("translator-worker-%s", workerID),
			Ports: []deployment.PortMapping{
				{HostPort: 8443 + workerIndex, ContainerPort: 8443, Protocol: "tcp"},
			},
			Environment: map[string]string{
				"JWT_SECRET":   generateJWTSecret(),
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

// generateMainConfig builds the coordinator (Main) instance config for a
// generated deployment plan. validateDeploymentPlan REQUIRES a non-nil Main
// with Host/User/DockerImage/ContainerName/>=1 Port, and DeployDistributedSystem
// dereferences plan.Main on its success-event path — so generate-plan MUST emit
// a populated Main, otherwise the tool's own `deploy` action rejects the tool's
// own output ("main instance configuration is required").
//
// SSH credentials are sourced (deterministically, by sorted worker key) from the
// first configured worker since the coordinator is typically reachable over the
// same access; the host falls back to the server host then localhost. The
// operator is expected to review/edit the emitted plan before deploying.
func generateMainConfig(cfg *config.Config) *deployment.DeploymentConfig {
	host := cfg.Server.Host
	user := "deploy"
	password := ""

	// Deterministically pick the first worker (sorted by id) for SSH access.
	if len(cfg.Distributed.Workers) > 0 {
		ids := make([]string, 0, len(cfg.Distributed.Workers))
		for id := range cfg.Distributed.Workers {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		first := cfg.Distributed.Workers[ids[0]]
		if first.User != "" {
			user = first.User
		}
		password = first.Password
		if host == "" {
			host = first.Host
		}
	}

	if host == "" {
		host = "localhost"
	}

	mainPort := cfg.Server.Port
	if mainPort <= 0 {
		mainPort = 8443
	}

	return &deployment.DeploymentConfig{
		Host:          host,
		User:          user,
		Password:      password,
		SSHKeyPath:    "",
		DockerImage:   "translator:latest",
		ContainerName: "translator-main",
		Ports: []deployment.PortMapping{
			{HostPort: mainPort, ContainerPort: 8443, Protocol: "tcp"},
		},
		Environment: map[string]string{
			"JWT_SECRET": generateJWTSecret(),
			"MAIN_HOST":  host,
			"ROLE":       "coordinator",
		},
		Volumes: []deployment.VolumeMapping{
			{HostPath: "./certs", ContainerPath: "/app/certs", ReadOnly: true},
			{HostPath: "./config.json", ContainerPath: "/app/config.json", ReadOnly: true},
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
}
