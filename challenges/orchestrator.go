package challenges

import (
	"context"
	"fmt"
	"os"
	"time"

	"digital.vasic.challenges/pkg/challenge"
	"digital.vasic.challenges/pkg/registry"
	"digital.vasic.challenges/pkg/runner"
)

// Orchestrator runs all HelixTranslate challenges.
type Orchestrator struct {
	reg    registry.Registry
	config OrchestratorConfig
}

// OrchestratorConfig holds challenge execution settings.
type OrchestratorConfig struct {
	Parallel       bool
	MaxConcurrency int
	StopOnFailure  bool
	Verbose        bool
	Timeout        time.Duration
	Filter         []string
	Category       string
	ResultsDir     string
}

// NewOrchestrator creates a challenge orchestrator.
func NewOrchestrator(config OrchestratorConfig) *Orchestrator {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 4
	}
	if config.ResultsDir == "" {
		config.ResultsDir = "challenge-results"
	}

	return &Orchestrator{
		reg:    registry.NewRegistry(),
		config: config,
	}
}

// RegisterAllChallenges registers the complete challenge suite.
func (o *Orchestrator) RegisterAllChallenges() error {
	// Core translation challenges
	challenges := []challenge.Challenge{
		NewBuildChallenge(),
		NewFormatDetectionChallenge(),
		NewFB2ParsingChallenge(),
		NewEPUBParsingChallenge(),
		NewMarkdownConversionChallenge(),
		NewLLMProviderChallenge(),
		NewOpenAIProviderChallenge(),
		NewAnthropicProviderChallenge(),
		NewEventBusChallenge(),
		NewWebSocketChallenge(),
		NewAPIEndpointsChallenge(),
		NewGRPCServerChallenge(),
		NewTranslationPipelineChallenge(),
		NewVerificationChallenge(),
		NewDistributedSSHChallenge(),
		NewSecurityAuthChallenge(),
		NewRateLimitingChallenge(),
		NewE2ETranslationChallenge(),
	}

	for _, c := range challenges {
		if err := o.reg.Register(c); err != nil {
			return fmt.Errorf("register challenge %s: %w", c.ID(), err)
		}
	}

	return nil
}

// Run executes all registered challenges and returns results.
func (o *Orchestrator) Run(ctx context.Context) ([]*challenge.Result, error) {
	if o.reg.Count() == 0 {
		return nil, fmt.Errorf("no challenges registered")
	}

	// Build runner with options
	opts := []runner.RunnerOption{
		runner.WithRegistry(o.reg),
		runner.WithResultsDir(o.config.ResultsDir),
		runner.WithTimeout(o.config.Timeout),
	}

	r := runner.NewRunner(opts...)

	config := &challenge.Config{
		ResultsDir: o.config.ResultsDir,
		LogsDir:    o.config.ResultsDir + "/logs",
		Timeout:    o.config.Timeout,
	}

	_ = os.MkdirAll(config.ResultsDir, 0755)
	_ = os.MkdirAll(config.LogsDir, 0755)

	results, err := r.RunAll(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("challenge run failed: %w", err)
	}

	// Write summary report
	reportPath := fmt.Sprintf("%s/summary-%s.md", o.config.ResultsDir, time.Now().Format("20060102-150405"))
	report := generateSummaryReport(results)
	_ = os.WriteFile(reportPath, []byte(report), 0644)

	return results, nil
}

func generateSummaryReport(results []*challenge.Result) string {
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

	report := "# HelixTranslate Challenge Results\n\n"
	report += fmt.Sprintf("**Total**: %d | **Passed**: %d | **Failed**: %d | **Skipped**: %d\n\n", len(results), passed, failed, skipped)
	report += "## Details\n\n"
	report += "| Challenge | Status | Duration |\n"
	report += "|-----------|--------|----------|\n"

	for _, r := range results {
		status := r.Status
		report += fmt.Sprintf("| %s | %s | %s |\n", r.ChallengeName, status, r.Duration)
	}

	return report
}
