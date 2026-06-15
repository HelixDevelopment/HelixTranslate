package main

import (
	"testing"

	"digital.vasic.translator/internal/config"
)

// TestGenerateDeploymentPlan_JWTSecretsAreNotHardcoded is an anti-bluff security
// regression test (§11.4.10 / §11.4.115). generate-plan previously emitted a
// hardcoded, predictable JWT_SECRET for both the Main coordinator ("main-secret")
// and every worker ("worker-<id>-secret"). An operator deploying the generated
// plan without editing it would ship a known-constant JWT signing key — a trivial
// auth bypass (anyone can mint valid tokens). The fix generates a
// cryptographically-random secret per plan via crypto/rand.
//
// This test reproduces the defect on the pre-fix code (the well-known constants)
// and becomes the GREEN regression guard once generateJWTSecret is wired in:
// the emitted secrets must NOT equal the old constants, must be long+hex
// (256-bit => 64 hex chars), and two independent generations must differ
// (proving they are random, not a fixed value).
func TestGenerateDeploymentPlan_JWTSecretsAreNotHardcoded(t *testing.T) {
	cfg := &config.Config{
		Distributed: config.DistributedConfig{
			Enabled: true,
			Workers: map[string]config.WorkerConfig{
				"worker1": {Host: "10.0.0.2", User: "deploy", Password: "pw"},
			},
		},
	}

	plan := generateDeploymentPlan(cfg)

	mainSecret := plan.Main.Environment["JWT_SECRET"]
	if mainSecret == "main-secret" {
		t.Errorf("Main JWT_SECRET is the hardcoded constant %q — an operator deploying "+
			"the plan unedited ships a known signing key (auth bypass, §11.4.10)", mainSecret)
	}
	if len(mainSecret) < 64 {
		t.Errorf("Main JWT_SECRET %q is too short (%d chars) to be a 256-bit random secret",
			mainSecret, len(mainSecret))
	}

	if len(plan.Workers) == 0 {
		t.Fatalf("expected >=1 worker in the generated plan")
	}
	workerSecret := plan.Workers[0].Environment["JWT_SECRET"]
	if workerSecret == "worker-worker1-secret" {
		t.Errorf("worker JWT_SECRET is the hardcoded constant %q — known signing key "+
			"(auth bypass, §11.4.10)", workerSecret)
	}
	if len(workerSecret) < 64 {
		t.Errorf("worker JWT_SECRET %q too short (%d chars) for a 256-bit random secret",
			workerSecret, len(workerSecret))
	}

	// Randomness: Main and worker secrets must differ, and a second plan must
	// produce different secrets again (a fixed constant would repeat).
	if mainSecret == workerSecret {
		t.Errorf("Main and worker JWT_SECRET are identical (%q) — not independently random", mainSecret)
	}
	plan2 := generateDeploymentPlan(cfg)
	if plan2.Main.Environment["JWT_SECRET"] == mainSecret {
		t.Errorf("two generate-plan runs produced the SAME Main JWT_SECRET (%q) — not random", mainSecret)
	}
}
