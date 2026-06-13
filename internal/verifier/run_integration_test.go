//go:build integration

package verifier_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/persistence"
)

// TestRunVerification_RealProviders runs the System's own in-process verifier
// against the REAL providers for which an API key is present in the
// environment. It is gated behind the `integration` build tag so the default
// `go test ./...` run boots nothing and makes no network calls.
//
// Anti-bluff (§11.4 / §11.4.3 / §11.4.123): when NO provider key is present the
// test SKIPs with a reason (never a faked PASS). When keys ARE present it makes
// real HTTP calls, asserts at least one provider authenticated, persists the
// verified models to a real SQLite store, and asserts the persisted row count
// matches the verified count. API-key VALUES are never logged.
func TestRunVerification_RealProviders(t *testing.T) {
	providers := verifier.ProvidersFromEnv()
	if len(providers) == 0 {
		t.Skip("SKIP: no provider API keys set in environment (§11.4.3 topology-aware skip)")
	}

	// Log which providers are configured by ID only — never the key value.
	t.Logf("configured providers (by ID, no key values): %v", verifier.EnvProviderIDs())

	dbPath := filepath.Join(t.TempDir(), "verified_models.db")
	store, err := persistence.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	reg := verifier.NewRegistry()
	pipe := verifier.NewPipeline()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	res, err := verifier.RunVerification(ctx, reg, pipe, store, providers, verifier.RunOptions{
		MaxModelsPerProvider: 3,   // bound wall-clock/token spend
		MinScoreToPersist:    0.0, // persist any model that passes the hard gates
	})
	if err != nil {
		t.Fatalf("RunVerification: %v", err)
	}

	anyAuth := false
	for _, pv := range res.Providers {
		// Per-provider real outcomes — key values never printed.
		t.Logf("provider=%s reachability=%v auth=%v authStatus=%d candidates=%d verified=%d authErr=%q",
			pv.ProviderID, pv.ReachabilityPass, pv.AuthPass, pv.AuthStatusCode,
			len(pv.CandidateModels), pv.VerifiedCount, pv.AuthError)
		if pv.AuthPass {
			anyAuth = true
		}
	}

	if !anyAuth {
		t.Fatalf("no provider authenticated successfully — every configured key was rejected or unreachable")
	}

	// Assert the persisted row count is a real, consistent number.
	persisted, err := store.Count()
	if err != nil {
		t.Fatalf("count persisted models: %v", err)
	}
	t.Logf("RunResult.TotalVerified=%d persisted_rows=%d duration=%s",
		res.TotalVerified, persisted, res.FinishedAt.Sub(res.StartedAt))

	if persisted != res.TotalVerified {
		t.Fatalf("persisted row count %d != reported verified count %d", persisted, res.TotalVerified)
	}

	// If at least one provider authed, we expect at least one verified+persisted
	// model with a real model ID (no fabrication).
	if res.TotalVerified == 0 {
		t.Logf("WARNING: a provider authenticated but no model passed the full pipeline; inspect per-provider results above")
	} else {
		loaded, err := store.LoadModels()
		if err != nil {
			t.Fatalf("load persisted models: %v", err)
		}
		if len(loaded) == 0 {
			t.Fatalf("TotalVerified=%d but LoadModels returned 0 rows", res.TotalVerified)
		}
		for _, m := range loaded {
			if m.ID == "" || m.ProviderID == "" {
				t.Errorf("persisted model has empty ID/ProviderID: %+v", m)
			}
			t.Logf("persisted verified model: provider=%s id=%s score=%.3f", m.ProviderID, m.ID, m.OverallScore)
		}
	}
}
