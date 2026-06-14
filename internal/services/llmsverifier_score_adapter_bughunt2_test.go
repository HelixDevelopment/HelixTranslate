package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

// TestGetPreferences_WithoutPriorRefresh_UsesNormalizedScore is a REPRODUCE-FIRST
// RED test for a state/scale-confusion defect in GetPreferences.
//
// GetPreferences is a self-contained public method: it fetches verified models
// itself via the client. But to compute each preference's Score/Weight and to
// apply MinScoreThreshold it calls GetProviderScore(m.ID), which only reads the
// a.scores cache populated by a SEPARATE call to RefreshScores. When a caller
// invokes GetPreferences WITHOUT having first called RefreshScores (a legitimate
// usage — GetPreferences already does its own fetch), GetProviderScore returns 0
// for every model.
//
// Consequences with a production-style positive MinScoreThreshold: every verified,
// high-scoring model is silently dropped (0 < threshold), so GetPreferences returns
// an EMPTY preference list while the registry is full of qualified models. This is
// the "tests pass but feature unusable" failure mode — every existing test happens
// to call RefreshScores first, masking it.
//
// The correct behaviour: GetPreferences must score each model from the model data
// it already fetched (normalize OverallScore) rather than depending on a side-channel
// cache that may be empty.
func TestGetPreferences_WithoutPriorRefresh_UsesNormalizedScore(t *testing.T) {
	models := []verifier.Model{
		{ID: "gpt-4", ProviderID: "openai", Name: "GPT-4", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
		{ID: "claude-3", ProviderID: "anthropic", Name: "Claude 3", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 92.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	cfg.MinScoreThreshold = 5.0 // production-style positive threshold (0-10 scale)
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	// NO RefreshScores() call here — GetPreferences must stand on its own.
	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)

	// Both models are verified and score 9.5 / 9.2 (normalized), comfortably above
	// the 5.0 threshold. They MUST appear.
	require.Len(t, prefs, 2, "verified high-scoring models must not be dropped just because RefreshScores was not called first")

	assert.Equal(t, "gpt-4", prefs[0].ModelID)
	assert.InDelta(t, 9.5, prefs[0].Score, 0.01, "Score must reflect the model's normalized OverallScore, not 0")
	assert.InDelta(t, 0.95, prefs[0].Weight, 0.01, "Weight must be score/10, not 0")
	assert.InDelta(t, 9.2, prefs[1].Score, 0.01)
}
