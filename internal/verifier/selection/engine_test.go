package selection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

func makeRegistryWithModels(t *testing.T) *verifier.Registry {
	t.Helper()
	r := verifier.NewRegistry()
	r.AddModel(verifier.Model{
		ID:                  "gpt-4",
		ProviderID:          "openai",
		Name:                "GPT-4",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        9.5,
		Capabilities:        map[string]bool{"streaming": true, "vision": true},
	})
	r.AddModel(verifier.Model{
		ID:                  "claude-3",
		ProviderID:          "anthropic",
		Name:                "Claude 3",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        9.2,
		Capabilities:        map[string]bool{"streaming": true},
	})
	r.AddModel(verifier.Model{
		ID:                  "llama-3",
		ProviderID:          "meta",
		Name:                "Llama 3",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        8.0,
		Capabilities:        map[string]bool{"streaming": true, "vision": true},
	})
	r.AddModel(verifier.Model{
		ID:                  "unverified",
		ProviderID:          "test",
		Name:                "Unverified",
		VerificationStatus:  "pending",
		CanSeeCode:          false,
		AffirmativeResponse: false,
		OverallScore:        9.9,
	})
	return r
}

func newTestSelectionEngine(t *testing.T) (*Engine, *verifier.Registry) {
	t.Helper()
	registry := makeRegistryWithModels(t)
	scoringEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})
	config := verifier.DefaultConfig()
	config.MinScoreThreshold = 0.0
	return NewEngine(registry, scoringEngine, config), registry
}

func TestSelectModel(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	model, err := engine.SelectModel(context.Background(), TaskRequirements{
		SourceLang: "en",
		TargetLang: "sr",
	})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", model.ID) // Highest score
}

func TestSelectModelWithRequirements(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	// Require vision - only gpt-4 and llama-3 support it
	model, err := engine.SelectModel(context.Background(), TaskRequirements{
		SourceLang:    "en",
		TargetLang:    "sr",
		RequireVision: true,
	})
	require.NoError(t, err)
	assert.True(t, model.Capabilities["vision"])
	assert.NotEqual(t, "claude-3", model.ID)
}

func TestSelectModelRequireCode(t *testing.T) {
	engine, registry := newTestSelectionEngine(t)
	registry.AddModel(verifier.Model{
		ID:                  "no-code-model",
		ProviderID:          "test",
		Name:                "No Code",
		VerificationStatus:  "verified",
		CanSeeCode:          false,
		AffirmativeResponse: true,
		OverallScore:        9.9,
		Capabilities:        map[string]bool{},
	})

	model, err := engine.SelectModel(context.Background(), TaskRequirements{
		SourceLang:   "en",
		TargetLang:   "sr",
		RequireCode:  true,
	})
	require.NoError(t, err)
	assert.True(t, model.CanSeeCode)
}

func TestSelectModelNoCandidates(t *testing.T) {
	registry := verifier.NewRegistry()
	scoringEngine := scoring.NewEngine(scoring.ScoreWeights{})
	config := verifier.DefaultConfig()
	config.MinScoreThreshold = 10.0 // Impossibly high
	engine := NewEngine(registry, scoringEngine, config)

	_, err := engine.SelectModel(context.Background(), TaskRequirements{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no verified models")
}

func TestSelectFallback(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	fallback, err := engine.SelectFallback("gpt-4", TaskRequirements{})
	require.NoError(t, err)
	assert.NotEqual(t, "gpt-4", fallback.ID)
}

func TestSelectFallbackNoOptions(t *testing.T) {
	registry := verifier.NewRegistry()
	registry.AddModel(verifier.Model{
		ID:                  "only-model",
		ProviderID:          "test",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        5.0,
		Capabilities:        map[string]bool{},
	})
	scoringEngine := scoring.NewEngine(scoring.ScoreWeights{})
	config := verifier.DefaultConfig()
	engine := NewEngine(registry, scoringEngine, config)

	_, err := engine.SelectFallback("only-model", TaskRequirements{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fallback")
}

func TestBuildFallbackChain(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	chain := engine.buildFallbackChain("gpt-4")
	require.Len(t, chain, 2)
	assert.NotContains(t, chain, "gpt-4")
	assert.NotContains(t, chain, "unverified")
}

// TestBuildFallbackChainScoreOrdered is a §11.4.115 reproduce-first guard for
// the fallback-ordering bug: buildFallbackChain must return remaining verified
// models in score-DESCENDING order so a primary failure falls back to the
// genuine next-best model (the SelectFallback doc contract: "the next best
// model when the primary fails"). The pre-fix code shuffled the chain randomly,
// so a low-score model could be tried before a high-score one, and the order
// was non-deterministic across runs (§11.4.50). Registry scores:
// gpt-4=9.5 > claude-3=9.2 > llama-3=8.0. Excluding gpt-4, the next-best order
// is exactly [claude-3, llama-3]; any other order is the bug.
func TestBuildFallbackChainScoreOrdered(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	// Run repeatedly: a random shuffle would (with overwhelming probability over
	// many iterations) produce the reversed order at least once. The correct
	// score-ordered implementation yields the identical deterministic order every
	// time.
	for i := 0; i < 50; i++ {
		chain := engine.buildFallbackChain("gpt-4")
		require.Equal(t, []string{"claude-3", "llama-3"}, chain,
			"fallback chain must be score-descending (next-best-first), deterministic across runs")
	}
}

func TestCalculateTaskScore(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	model := verifier.Model{
		ID:           "test",
		OverallScore: 10.0,
		Capabilities: map[string]bool{"streaming": false, "vision": false},
		CanSeeCode:   false,
	}

	// Base score with no penalties
	score := engine.calculateTaskScore(model, TaskRequirements{})
	assert.Equal(t, 10.0, score)

	// Penalty for missing streaming
	score = engine.calculateTaskScore(model, TaskRequirements{RequireStreaming: true})
	assert.Equal(t, 5.0, score)

	// Penalty for missing vision
	score = engine.calculateTaskScore(model, TaskRequirements{RequireVision: true})
	assert.Equal(t, 5.0, score)

	// Penalty for missing code
	score = engine.calculateTaskScore(model, TaskRequirements{RequireCode: true})
	assert.Equal(t, 3.0, score)
}

func TestMeetsRequirements(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	model := verifier.Model{
		Capabilities: map[string]bool{"streaming": true, "vision": false},
		CanSeeCode:   true,
	}

	assert.True(t, engine.meetsRequirements(model, TaskRequirements{RequireStreaming: true}))
	assert.False(t, engine.meetsRequirements(model, TaskRequirements{RequireVision: true}))
	assert.True(t, engine.meetsRequirements(model, TaskRequirements{RequireCode: true}))
	assert.False(t, engine.meetsRequirements(model, TaskRequirements{RequireCode: true, RequireVision: true}))
}

func TestSelectModelCachesFallback(t *testing.T) {
	engine, _ := newTestSelectionEngine(t)

	// First call builds the chain
	_, err := engine.SelectFallback("gpt-4", TaskRequirements{})
	require.NoError(t, err)

	// Second call uses cached chain
	_, err = engine.SelectFallback("gpt-4", TaskRequirements{})
	require.NoError(t, err)
}
