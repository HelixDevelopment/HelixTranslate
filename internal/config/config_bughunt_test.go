package config

import (
	"strings"
	"testing"
	"time"
)

// baseValidLLMsVerifierEnabledConfig returns a Config with LLMsVerifier enabled
// and otherwise valid so that Validate() reaches the scoring-weights check.
func baseValidLLMsVerifierEnabledConfig() *Config {
	c := DefaultConfig()
	c.Security.EnableAuth = false // avoid unrelated JWT-secret requirement
	c.LLMsVerifier.Enabled = true
	c.LLMsVerifier.APIURL = "http://localhost:8080"
	c.LLMsVerifier.APIKey = "test-key"
	c.LLMsVerifier.CacheTTL = time.Hour
	return c
}

// TestValidate_RejectsAllZeroScoringWeights is a REPRODUCE-FIRST RED test for the
// fail-open validation gap: when LLMsVerifier is enabled, a scoring-weight set in
// which every component is 0 sums to 0, and the `total > 0` guard in
// validateLLMsVerifierConfig SKIPS the sum check entirely — silently accepting a
// degenerate configuration that produces a meaningless (all-zero) scoring engine.
// An all-zero weight set is obviously invalid and MUST be rejected when the
// verifier is enabled.
func TestValidate_RejectsAllZeroScoringWeights(t *testing.T) {
	c := baseValidLLMsVerifierEnabledConfig()
	c.LLMsVerifier.ScoringWeights = ScoreWeights{
		ResponseSpeed:     0,
		CostEffectiveness: 0,
		ModelEfficiency:   0,
		Capability:        0,
		Recency:           0,
	}

	err := c.Validate()
	if err == nil {
		t.Fatalf("expected Validate() to reject all-zero scoring weights when verifier enabled, got nil")
	}
	if !strings.Contains(err.Error(), "scoring weights") {
		t.Fatalf("expected scoring-weights error, got: %v", err)
	}
}

// TestValidate_AcceptsValidScoringWeights guards the GREEN side: a weight set that
// sums to 1.0 must still pass.
func TestValidate_AcceptsValidScoringWeights(t *testing.T) {
	c := baseValidLLMsVerifierEnabledConfig()
	// DefaultConfig already sums to 1.0; assert it still validates.
	if err := c.Validate(); err != nil {
		t.Fatalf("expected valid default scoring weights to pass, got: %v", err)
	}
}
