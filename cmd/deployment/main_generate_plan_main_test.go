package main

import (
	"testing"

	"digital.vasic.translator/internal/config"
)

// TestGenerateDeploymentPlan_ProducesDeployableMainInstance is an anti-bluff
// regression test (§11.4.115). The `generate-plan` action writes a
// DeploymentPlan to disk that the operator then feeds to the `deploy` action.
// DeployDistributedSystem -> validateDeploymentPlan REQUIRES a non-nil Main
// instance with Host/User/DockerImage/ContainerName/>=1 Port; if Main is nil
// the deploy fails immediately with "main instance configuration is required",
// and DeployDistributedSystem dereferences deploymentPlan.Main (nil-deref panic
// risk on the success-event path at orchestrator.go:149). A generate-plan that
// emits an undeployable plan is a CLI orchestration bug: the tool's own output
// is rejected by the tool's own deploy path.
//
// This test reproduces the defect on the current code (Main is left nil) and
// becomes the GREEN regression guard once generateDeploymentPlan populates Main.
func TestGenerateDeploymentPlan_ProducesDeployableMainInstance(t *testing.T) {
	cfg := &config.Config{
		Distributed: config.DistributedConfig{
			Enabled: true,
			Workers: map[string]config.WorkerConfig{
				"worker1": {Host: "10.0.0.2", User: "deploy", Password: "pw"},
			},
		},
	}

	plan := generateDeploymentPlan(cfg)

	if plan.Main == nil {
		t.Fatalf("generate-plan produced a plan with nil Main: the deploy action " +
			"validateDeploymentPlan rejects this with \"main instance configuration " +
			"is required\" and DeployDistributedSystem nil-derefs plan.Main")
	}

	// Mirror the orchestrator's validateInstanceConfig required-field contract so
	// the generated Main is actually deployable, not merely non-nil.
	m := plan.Main
	if m.Host == "" {
		t.Errorf("Main.Host empty: validateInstanceConfig requires a host")
	}
	if m.User == "" {
		t.Errorf("Main.User empty: validateInstanceConfig requires a user")
	}
	if m.DockerImage == "" {
		t.Errorf("Main.DockerImage empty: validateInstanceConfig requires a docker image")
	}
	if m.ContainerName == "" {
		t.Errorf("Main.ContainerName empty: validateInstanceConfig requires a container name")
	}
	if len(m.Ports) == 0 {
		t.Errorf("Main.Ports empty: validateInstanceConfig requires >=1 port mapping")
	}
}
