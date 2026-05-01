package unit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// TestEngine_SelectModel_Basic verifies model selection works.
func TestEngine_SelectModel_Basic(t *testing.T) {
	reg := verifier.NewRegistry()
	reg.AddModel(verifier.Model{
		ID: "gpt-4", ProviderID: "openai", Name: "GPT-4",
		VerificationStatus: "verified", CanSeeCode: true,
		AffirmativeResponse: true, OverallScore: 0.95,
		Capabilities: map[string]bool{"streaming": true},
	})

	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})
	_, _ = scoreEngine.CalculateScore("gpt-4", 0.95, 0, 0, 0, 0)

	cfg := verifier.DefaultConfig()
	engine := selection.NewEngine(reg, scoreEngine, cfg)

	model, err := engine.SelectModel(context.Background(), selection.TaskRequirements{
		SourceLang: "en", TargetLang: "sr",
	})
	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "gpt-4", model.ID)
}

// TestEngine_SelectModel_NoModels verifies error when registry is empty.
func TestEngine_SelectModel_NoModels(t *testing.T) {
	reg := verifier.NewRegistry()
	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{})
	cfg := verifier.DefaultConfig()
	engine := selection.NewEngine(reg, scoreEngine, cfg)

	_, err := engine.SelectModel(context.Background(), selection.TaskRequirements{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no verified models available")
}

// TestEngine_SelectModel_RequirementsNotMet verifies capability filtering.
func TestEngine_SelectModel_RequirementsNotMet(t *testing.T) {
	reg := verifier.NewRegistry()
	reg.AddModel(verifier.Model{
		ID: "no-stream", ProviderID: "p", VerificationStatus: "verified",
		CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.9,
		Capabilities: map[string]bool{"streaming": false},
	})

	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})
	_, _ = scoreEngine.CalculateScore("no-stream", 0.9, 0, 0, 0, 0)

	cfg := verifier.DefaultConfig()
	engine := selection.NewEngine(reg, scoreEngine, cfg)

	_, err := engine.SelectModel(context.Background(), selection.TaskRequirements{
		RequireStreaming: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no model meets task requirements")
}

// TestEngine_SelectFallback verifies fallback chain construction.
func TestEngine_SelectFallback(t *testing.T) {
	reg := verifier.NewRegistry()
	reg.AddModel(verifier.Model{
		ID: "primary", ProviderID: "p1", VerificationStatus: "verified",
		CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.9,
	})
	reg.AddModel(verifier.Model{
		ID: "fallback", ProviderID: "p2", VerificationStatus: "verified",
		CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.8,
	})

	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})
	_, _ = scoreEngine.CalculateScore("primary", 0.9, 0, 0, 0, 0)
	_, _ = scoreEngine.CalculateScore("fallback", 0.8, 0, 0, 0, 0)

	cfg := verifier.DefaultConfig()
	engine := selection.NewEngine(reg, scoreEngine, cfg)

	fb, err := engine.SelectFallback("primary", selection.TaskRequirements{})
	require.NoError(t, err)
	require.NotNil(t, fb)
	assert.NotEqual(t, "primary", fb.ID)
}

// TestEngine_SelectFallback_NoFallback verifies error when no alternatives exist.
func TestEngine_SelectFallback_NoFallback(t *testing.T) {
	reg := verifier.NewRegistry()
	reg.AddModel(verifier.Model{
		ID: "only", ProviderID: "p", VerificationStatus: "verified",
		CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.9,
	})

	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})
	_, _ = scoreEngine.CalculateScore("only", 0.9, 0, 0, 0, 0)

	cfg := verifier.DefaultConfig()
	engine := selection.NewEngine(reg, scoreEngine, cfg)

	_, err := engine.SelectFallback("only", selection.TaskRequirements{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fallback model available")
}
