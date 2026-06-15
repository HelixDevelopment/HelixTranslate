package deployment

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
)

// TestSSHDeployConfig_Validate_NoSharedMutationRace is a reproduce-first RED test
// for a data race in SSHDeployConfig.Validate (ssh_deployer.go:75-89). Validate
// MUTATES its receiver as a side effect of defaulting (Port=22, Timeout=30s,
// ConnectRetries=3, CommandTimeout=10m). SSHDeployer.Connect calls Validate, and a
// single *SSHDeployConfig is routinely shared across deployers/goroutines (see
// TestSSHDeployer_Concurrent). Concurrent Connect/Validate then write+read the
// shared config fields without synchronization -> data race. A validator must be
// safe for concurrent callers (no hidden mutation of shared state). Run under -race.
func TestSSHDeployConfig_Validate_NoSharedMutationRace(t *testing.T) {
	// All goroutines share ONE config pointer, with the defaulted fields left zero
	// so every Validate call takes the write path.
	cfg := &SSHDeployConfig{
		Host:     "localhost",
		Username: "testuser",
		Password: "testpass",
	}

	const goroutines = 8
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = cfg.Validate()
			}
		}()
	}
	wg.Wait()
}

// TestUpdateAllServices_DoesNotDoubleTagImage is a reproduce-first RED test for the
// image-reference construction bug in DeploymentOrchestrator.UpdateAllServices
// (orchestrator.go:479). It builds the "update to latest" image as
// instance.Config.DockerImage + ":latest". When the deployed image ALREADY carries
// a tag (the normal case — pinned versions for deploy/rollback decisions), this
// produces an invalid double-tag reference like "repo/img:v1.2.3:latest", which
// Docker rejects ("invalid reference format"). The deployed instance is then left
// recording a broken image string.
func TestUpdateAllServices_DoesNotDoubleTagImage(t *testing.T) {
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	orchestrator := NewDeploymentOrchestrator(cfg, eventBus)
	defer orchestrator.Close()

	// Seed a deployed instance whose image is pinned to an explicit version tag.
	orchestrator.mu.Lock()
	orchestrator.deployed["translator-main"] = &DeployedInstance{
		ID:     "translator-main",
		Host:   "test.example.com",
		Port:   8443,
		Status: "healthy",
		Config: &DeploymentConfig{
			Host:          "test.example.com",
			DockerImage:   "myrepo/translator:v1.2.3",
			ContainerName: "translator-main",
			Ports:         []PortMapping{{HostPort: 8443, ContainerPort: 8443}},
		},
	}
	orchestrator.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// deployer stubs make UpdateInstance/CheckInstanceHealth succeed, so the only
	// thing exercised here is the image-string construction + storage.
	if err := orchestrator.UpdateAllServices(ctx); err != nil {
		t.Fatalf("UpdateAllServices returned error: %v", err)
	}

	orchestrator.mu.RLock()
	got := orchestrator.deployed["translator-main"].Config.DockerImage
	orchestrator.mu.RUnlock()

	// A valid Docker image reference has at most one tag after the final path
	// component, i.e. at most one ':' in the part after the last '/'.
	lastSlash := strings.LastIndex(got, "/")
	namePart := got
	if lastSlash >= 0 {
		namePart = got[lastSlash+1:]
	}
	if strings.Count(namePart, ":") > 1 {
		t.Fatalf("UpdateAllServices produced a double-tagged (invalid) image reference: %q", got)
	}
}
