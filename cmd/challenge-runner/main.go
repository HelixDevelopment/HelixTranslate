package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.translator/pkg/challenge_runner"
)

func main() {
	var (
		parallel       = flag.Bool("parallel", true, "Run challenges in parallel")
		maxConcurrency = flag.Int("max-concurrency", 4, "Max concurrent challenges")
		stopOnFailure  = flag.Bool("stop-on-failure", false, "Stop on first failure")
		verbose        = flag.Bool("verbose", false, "Verbose output")
		timeout        = flag.Duration("timeout", 5*time.Minute, "Per-challenge timeout")
		filter         = flag.String("filter", "", "Comma-separated challenge IDs to run")
		category       = flag.String("category", "", "Limit to specific category")
		resultsDir     = flag.String("results-dir", "challenge-results", "Results directory")
	)
	flag.Parse()

	config := challenge_runner.OrchestratorConfig{
		Parallel:       *parallel,
		MaxConcurrency: *maxConcurrency,
		StopOnFailure:  *stopOnFailure,
		Verbose:        *verbose,
		Timeout:        *timeout,
		ResultsDir:     *resultsDir,
		Category:       *category,
	}

	if *filter != "" {
		config.Filter = strings.Split(*filter, ",")
	}

	orchestrator := challenge_runner.NewOrchestrator(config)

	fmt.Println("=== HelixTranslate Challenge Runner ===")
	fmt.Println("Registering challenges...")

	if err := orchestrator.RegisterAllChallenges(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register challenges: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running challenges...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	results, err := orchestrator.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Challenge run failed: %v\n", err)
		os.Exit(1)
	}

	var passed, failed, skipped int
	for _, r := range results {
		switch r.Status {
		case challenge.StatusPassed:
			passed++
		case challenge.StatusFailed:
			failed++
		case challenge.StatusSkipped:
			skipped++
		}
	}

	fmt.Println()
	fmt.Println("=== Challenge Results ===")
	fmt.Printf("Total:    %d\n", len(results))
	fmt.Printf("Passed:   %d\n", passed)
	fmt.Printf("Failed:   %d\n", failed)
	fmt.Printf("Skipped:  %d\n", skipped)

	for _, r := range results {
		status := r.Status
		fmt.Printf("[%s] %s (%s)\n", status, r.ChallengeName, r.Duration)
	}

	if failed > 0 {
		fmt.Println()
		fmt.Printf("Results written to: %s\n", *resultsDir)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== All Challenges Passed ===")
}
