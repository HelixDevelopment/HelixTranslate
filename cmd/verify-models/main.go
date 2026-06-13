// Command verify-models runs HelixTranslate's own in-process LLMsVerifier
// pipeline against the real providers for which an API key is present in the
// environment, then persists the verified models to a local SQLite store.
//
// Usage:
//
//	source "$HOME/api_keys.sh" && go run ./cmd/verify-models -db ./data/verified_models.db -max 3
//
// API-key VALUES are NEVER printed — only provider IDs and real per-provider
// reachability/auth outcomes (§11.4.10).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/persistence"
)

func main() {
	dbPath := flag.String("db", "./data/verified_models.db", "path to the SQLite store for verified models")
	maxModels := flag.Int("max", 3, "max models per provider to run through the full pipeline (0 = no cap)")
	minScore := flag.Float64("min-score", 0.0, "minimum overall score (exclusive) to persist a model as verified")
	timeout := flag.Duration("timeout", 5*time.Minute, "overall verification timeout")
	flag.Parse()

	providers := verifier.ProvidersFromEnv()
	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "no provider API keys set in environment; nothing to verify")
		fmt.Fprintln(os.Stderr, "set e.g. DEEPSEEK_API_KEY / GROQ_API_KEY / OPENROUTER_API_KEY and re-run")
		os.Exit(2)
	}

	fmt.Printf("configured providers (by ID, no key values): %v\n", verifier.EnvProviderIDs())

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create db dir: %v\n", err)
		os.Exit(1)
	}

	store, err := persistence.NewSQLiteStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open sqlite store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	reg := verifier.NewRegistry()
	pipe := verifier.NewPipeline()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	res, err := verifier.RunVerification(ctx, reg, pipe, store, providers, verifier.RunOptions{
		MaxModelsPerProvider: *maxModels,
		MinScoreToPersist:    *minScore,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "verification error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("---- per-provider results ----")
	for _, pv := range res.Providers {
		fmt.Printf("provider=%-12s reachable=%-5v auth=%-5v status=%-3d candidates=%-3d verified=%-3d %s\n",
			pv.ProviderID, pv.ReachabilityPass, pv.AuthPass, pv.AuthStatusCode,
			len(pv.CandidateModels), pv.VerifiedCount, errSuffix(pv.AuthError))
	}

	count, err := store.Count()
	if err != nil {
		fmt.Fprintf(os.Stderr, "count persisted models: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("---- summary ----")
	fmt.Printf("total verified (persisted this run): %d\n", res.TotalVerified)
	fmt.Printf("persisted rows in store:             %d\n", count)
	fmt.Printf("duration:                            %s\n", res.FinishedAt.Sub(res.StartedAt))
	fmt.Printf("db:                                  %s\n", *dbPath)
}

func errSuffix(e string) string {
	if e == "" {
		return ""
	}
	return "err=" + e
}
