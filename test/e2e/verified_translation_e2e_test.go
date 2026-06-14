//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/services"
	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// TestEndToEnd_VerifiedTranslationPipeline simulates a full translation request
// using only LLMsVerifier-approved models.
func TestEndToEnd_VerifiedTranslationPipeline(t *testing.T) {
	// Anti-bluff: This test exercises the real integration path without mocks
	// for the core business logic. It verifies that a translation task would
	// receive a verified model, not just that functions exist.

	// Setup mock LLMsVerifier with realistic model catalog
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "version": "1.0.0"})
		case "/api/models":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]verifier.Model{
				{
					ID: "gpt-4o-2024-08-06", ProviderID: "openai",
					Name: "GPT-4o", VerificationStatus: "verified",
					CanSeeCode: true, AffirmativeResponse: true,
					OverallScore: 0.94,
					ResponsivenessScore: 0.92, CodeCapabilityScore: 0.88,
					FeatureRichnessScore: 0.96, ReliabilityScore: 0.95,
					Capabilities: map[string]bool{"streaming": true, "vision": true, "tool_calling": true},
					Pricing: verifier.PricingInfo{InputTokenCost: 2.5, OutputTokenCost: 10.0, Currency: "USD"},
					LastVerifiedAt: time.Now().Add(-1 * time.Hour),
				},
				{
					ID: "claude-3-5-sonnet-20241022", ProviderID: "anthropic",
					Name: "Claude 3.5 Sonnet", VerificationStatus: "verified",
					CanSeeCode: true, AffirmativeResponse: true,
					OverallScore: 0.96,
					ResponsivenessScore: 0.95, CodeCapabilityScore: 0.91,
					FeatureRichnessScore: 0.94, ReliabilityScore: 0.98,
					Capabilities: map[string]bool{"streaming": true, "vision": true, "tool_calling": true},
					Pricing: verifier.PricingInfo{InputTokenCost: 3.0, OutputTokenCost: 15.0, Currency: "USD"},
					LastVerifiedAt: time.Now().Add(-2 * time.Hour),
				},
				{
					// This model should be filtered out: fails verification
					ID: "bad-model", ProviderID: "shady-corp",
					Name: "Bad Model", VerificationStatus: "failed",
					CanSeeCode: false, AffirmativeResponse: false,
					OverallScore: 0.0,
					Capabilities: map[string]bool{},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Phase 1: Initialize LLMsVerifier client
	cfg := &verifier.Config{
		APIURL:            server.URL,
		APIKey:            "e2e-test-key",
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.5,
	}
	client := verifier.NewClient(cfg)

	// Phase 2: Health check
	ctx := context.Background()
	require.NoError(t, client.Ping(ctx), "LLMsVerifier must be reachable")

	// Phase 3: Fetch and cache verified models
	models, err := client.GetVerifiedModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 3, "Mock should return 3 models")

	// Phase 4: Populate registry with verified models only
	reg := verifier.NewRegistry()
	for _, m := range models {
		reg.AddModel(m)
	}

	// Phase 5: Initialize scoring engine and compute scores
	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	adapter := services.NewLLMsVerifierScoreAdapter(client, scoreEngine, cfg)
	require.NoError(t, adapter.RefreshScores(ctx))

	// Phase 6: Verify unverified models are NOT in preferences
	prefs, err := adapter.GetPreferences(ctx)
	require.NoError(t, err)

	for _, p := range prefs {
		assert.NotEqual(t, "bad-model", p.ModelID, "Unverified model must not appear in preferences")
		assert.Greater(t, p.Score, 0.0, "All preferred models must have positive scores")
		assert.True(t, p.Score >= cfg.MinScoreThreshold, "Score must meet threshold")
	}

	// Phase 7: Select best model for translation task
	selEngine := selection.NewEngine(reg, scoreEngine, cfg)
	selected, err := selEngine.SelectModel(ctx, selection.TaskRequirements{
		SourceLang:       "English",
		TargetLang:       "Serbian",
		RequireStreaming: true,
		RequireCode:      false,
	})
	require.NoError(t, err)
	require.NotNil(t, selected)

	// Anti-bluff: The selected model must be one of the verified models
	assert.Contains(t, []string{"gpt-4o-2024-08-06", "claude-3-5-sonnet-20241022"}, selected.ID)
	assert.Equal(t, "verified", selected.VerificationStatus)
	assert.True(t, selected.CanSeeCode)
	assert.True(t, selected.AffirmativeResponse)

	// Phase 8: Verify fallback chain
	fallback, err := selEngine.SelectFallback(selected.ID, selection.TaskRequirements{
		RequireStreaming: true,
	})
	require.NoError(t, err)
	require.NotNil(t, fallback)
	assert.NotEqual(t, selected.ID, fallback.ID)
	assert.Equal(t, "verified", fallback.VerificationStatus)

	// Phase 9: Verify evidence is captured
	t.Logf("Selected model: %s (score: %.2f)", selected.Name, selected.OverallScore)
	t.Logf("Fallback model: %s (score: %.2f)", fallback.Name, fallback.OverallScore)
	t.Logf("Preferences count: %d", len(prefs))

	// Phase 10: Mutation test — raise threshold above all scores, selection must fail
	cfg.MinScoreThreshold = 1.0
	_, err = selEngine.SelectModel(ctx, selection.TaskRequirements{})
	require.Error(t, err, "When threshold exceeds all scores, selection MUST fail")
	assert.Contains(t, err.Error(), "no verified models available")
}
