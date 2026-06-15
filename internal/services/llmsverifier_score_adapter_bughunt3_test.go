package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

// TestGetPreferences_ThresholdScaleMatchesRegistryAndHandler is a §11.4.115
// RED-on-broken-artifact reproduction of a cross-seam scale defect:
// MinScoreThreshold is interpreted on the RAW 0-100 score scale by every other
// consumer of the config field — the public /api/v1/verified-models handler
// (listVerifiedModels: `m.OverallScore <= MinScoreThreshold`, proven by
// TestVerifierServing_EnvelopeRespectsThreshold which uses score=95 / threshold=96)
// AND the live model-selection path (selection.Engine.SelectModel ->
// registry.FilterVerified -> `m.OverallScore > minScore`). Only
// LLMsVerifierScoreAdapter.GetPreferences diverged by NORMALIZING the score to
// 0-10 BEFORE comparing it against the same raw-scale threshold.
//
// Consequence for the end user: an operator who sets the documented-convention
// threshold MinScoreThreshold=50 ("keep models scoring above 50/100") gets the
// /verified-models list and the selection engine behaving correctly, but
// GetPreferences silently drops EVERY model (normalizeScore(95)=9.5 < 50), so
// any consumer of the adapter's preference list sees zero candidates while the
// rest of the system reports the models as available.
func TestGetPreferences_ThresholdScaleMatchesRegistryAndHandler(t *testing.T) {
	models := []verifier.Model{
		{ID: "high", ProviderID: "p", Name: "High", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
		{ID: "mid", ProviderID: "p", Name: "Mid", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 60.0},
		{ID: "low", ProviderID: "p", Name: "Low", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 40.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	// Documented / registry / handler convention: threshold on the RAW 0-100 scale.
	cfg.MinScoreThreshold = 50.0

	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})

	// Ground truth: the registry path that the live selection engine actually uses
	// (registry.FilterVerified compares raw OverallScore > MinScoreThreshold).
	registry := verifier.NewRegistry()
	for _, m := range models {
		registry.AddModel(m)
	}
	wantIDs := map[string]bool{}
	for _, m := range registry.FilterVerified(cfg.MinScoreThreshold) {
		wantIDs[m.ID] = true
	}
	// Sanity: registry keeps the two models above 50 (95, 60), drops 40.
	require.Equal(t, map[string]bool{"high": true, "mid": true}, wantIDs,
		"registry/selection ground truth: raw threshold 50 keeps 95 and 60, drops 40")

	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)
	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)

	gotIDs := map[string]bool{}
	for _, p := range prefs {
		gotIDs[p.ModelID] = true
	}

	// The adapter MUST admit the same set the registry/handler admit for the
	// identical config threshold. Before the fix, GetPreferences normalized to
	// 0-10 first (9.5/6.0/4.0 all < 50) and returned ZERO preferences.
	assert.Equal(t, wantIDs, gotIDs,
		"GetPreferences must apply MinScoreThreshold on the same raw 0-100 scale as the registry/selection engine and the /verified-models handler")
	assert.Len(t, prefs, 2, "raw threshold 50 must keep exactly the two models scoring above 50")
}
