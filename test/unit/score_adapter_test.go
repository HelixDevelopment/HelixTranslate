package unit

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
)

// TestScoreAdapter_RefreshScores verifies score fetching and normalization.
func TestScoreAdapter_RefreshScores(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{
			{ID: "model-100", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 100},
			{ID: "model-10", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 10},
			{ID: "model-1", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 1},
			{ID: "model-0", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0},
		})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{APIURL: server.URL, APIKey: "k", CacheTTL: time.Hour})
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := services.NewLLMsVerifierScoreAdapter(client, engine, verifier.DefaultConfig())

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	// Anti-bluff: Verify normalization logic
	// 100 -> 10, 10 -> 10, 1 -> 10 (clamp), 0 -> 0
	assert.Equal(t, 10.0, adapter.GetProviderScore("model-100"))
	assert.Equal(t, 10.0, adapter.GetProviderScore("model-10"))
	assert.Equal(t, 10.0, adapter.GetProviderScore("model-1"))
	assert.Equal(t, 0.0, adapter.GetProviderScore("model-0"))
}

// TestScoreAdapter_GetPreferences_Filtering verifies threshold filtering.
func TestScoreAdapter_GetPreferences_Filtering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{
			{ID: "high", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 9.0},
			{ID: "low", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 3.0},
		})
	}))
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.MinScoreThreshold = 5.0

	client := verifier.NewClient(&verifier.Config{APIURL: server.URL, APIKey: "k", CacheTTL: time.Hour})
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := services.NewLLMsVerifierScoreAdapter(client, engine, cfg)

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs, 1)
	assert.Equal(t, "high", prefs[0].ModelID)
}

// TestScoreAdapter_LastRefresh verifies timestamp tracking.
func TestScoreAdapter_LastRefresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{APIURL: server.URL, APIKey: "k", CacheTTL: time.Hour})
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := services.NewLLMsVerifierScoreAdapter(client, engine, verifier.DefaultConfig())

	before := time.Now()
	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)
	after := time.Now()

	refresh := adapter.LastRefresh()
	assert.True(t, refresh.After(before) || refresh.Equal(before))
	assert.True(t, refresh.Before(after) || refresh.Equal(after))
}
