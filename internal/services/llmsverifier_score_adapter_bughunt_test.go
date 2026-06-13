package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

// TestGetPreferences_SortedByScoreDescending is a REPRODUCE-FIRST RED test for the
// bug where GetPreferences documents "sorted by score descending" but performs no
// sort. The upstream API returns models in arbitrary order (here ascending score);
// the documented contract — and the FallbackOrder semantics that depend on it —
// require descending-by-score output.
func TestGetPreferences_SortedByScoreDescending(t *testing.T) {
	// API returns models in ASCENDING score order (a realistic upstream ordering).
	models := []verifier.Model{
		{ID: "low", ProviderID: "test", Name: "Low", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 60.0},
		{ID: "mid", ProviderID: "test", Name: "Mid", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 80.0},
		{ID: "high", ProviderID: "test", Name: "High", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	require.NoError(t, adapter.RefreshScores(context.Background()))

	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs, 3)

	// Documented contract: descending by score.
	assert.Equal(t, "high", prefs[0].ModelID, "highest score must be first")
	assert.Equal(t, "mid", prefs[1].ModelID)
	assert.Equal(t, "low", prefs[2].ModelID, "lowest score must be last")

	// Scores must be monotonically non-increasing.
	for i := 1; i < len(prefs); i++ {
		assert.GreaterOrEqual(t, prefs[i-1].Score, prefs[i].Score,
			"prefs must be sorted by score descending")
	}
}

// TestGetPreferences_FallbackOrderIsContiguousRank is a REPRODUCE-FIRST RED test
// for the bug where FallbackOrder = i+1 uses the index in the UNFILTERED models
// slice. When a leading model is filtered out (unverified), the first accepted
// preference must still be FallbackOrder=1 (a contiguous 1-based rank), not the
// raw upstream index.
func TestGetPreferences_FallbackOrderIsContiguousRank(t *testing.T) {
	models := []verifier.Model{
		// First model is filtered out (unverified) -> must NOT consume rank 1.
		{ID: "filtered", ProviderID: "test", Name: "Filtered", VerificationStatus: "pending", CanSeeCode: false, AffirmativeResponse: false, OverallScore: 99.0},
		{ID: "best", ProviderID: "test", Name: "Best", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
		{ID: "second", ProviderID: "test", Name: "Second", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 90.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	require.NoError(t, adapter.RefreshScores(context.Background()))

	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs, 2)

	// FallbackOrder must be a contiguous 1-based rank among accepted prefs.
	assert.Equal(t, 1, prefs[0].FallbackOrder, "first accepted pref must be rank 1, not the raw upstream index")
	assert.Equal(t, 2, prefs[1].FallbackOrder)
}
