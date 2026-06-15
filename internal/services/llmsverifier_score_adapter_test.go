package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

func makeTestServer(models []verifier.Model) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	}))
}

func TestNewLLMsVerifierScoreAdapter(t *testing.T) {
	client := verifier.NewClient(verifier.DefaultConfig())
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, verifier.DefaultConfig())
	assert.NotNil(t, adapter)
}

func TestRefreshScores(t *testing.T) {
	models := []verifier.Model{
		{ID: "gpt-4", ProviderID: "openai", Name: "GPT-4", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
		{ID: "claude-3", ProviderID: "anthropic", Name: "Claude 3", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 92.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	assert.InDelta(t, 9.5, adapter.GetProviderScore("gpt-4"), 0.01)
	assert.InDelta(t, 9.2, adapter.GetProviderScore("claude-3"), 0.01)
	assert.Equal(t, 0.0, adapter.GetProviderScore("unknown"))
	assert.False(t, adapter.LastRefresh().IsZero())
}

func TestRefreshScoresAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	err := adapter.RefreshScores(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch")
}

func TestGetPreferences(t *testing.T) {
	models := []verifier.Model{
		{ID: "gpt-4", ProviderID: "openai", Name: "GPT-4", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0, Capabilities: map[string]bool{"streaming": true}},
		{ID: "claude-3", ProviderID: "anthropic", Name: "Claude 3", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 92.0, Capabilities: map[string]bool{"streaming": true}},
		{ID: "unverified", ProviderID: "test", Name: "Bad", VerificationStatus: "pending", CanSeeCode: false, AffirmativeResponse: false, OverallScore: 99.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	// Refresh scores first so they're cached
	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs, 2)

	// CONST-034: unverified model filtered out
	for _, p := range prefs {
		assert.NotEqual(t, "unverified", p.ModelID)
	}

	// Should be sorted by score (gpt-4 first, then claude-3)
	assert.Equal(t, "gpt-4", prefs[0].ModelID)
	assert.Equal(t, "claude-3", prefs[1].ModelID)
}

func TestGetPreferencesWithMinThreshold(t *testing.T) {
	models := []verifier.Model{
		{ID: "high", ProviderID: "test", Name: "High", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
		{ID: "low", ProviderID: "test", Name: "Low", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 50.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	// MinScoreThreshold is on the RAW score scale (same convention as the
	// /verified-models handler and the registry/selection engine). Threshold 5
	// keeps both raw scores 95 and 50.
	cfg.MinScoreThreshold = 5.0
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs, 2)

	// Now with a higher RAW threshold that excludes the 50-scored model but
	// keeps the 95-scored one.
	cfg.MinScoreThreshold = 90.0
	adapter2 := NewLLMsVerifierScoreAdapter(client, engine, cfg)
	_ = adapter2.RefreshScores(context.Background())

	prefs2, err := adapter2.GetPreferences(context.Background())
	require.NoError(t, err)
	require.Len(t, prefs2, 1)
	assert.Equal(t, "high", prefs2[0].ModelID)
}

func TestNormalizeScore(t *testing.T) {
	// 0-1 scale -> multiply by 10
	assert.InDelta(t, 5.0, normalizeScore(0.5), 0.01)
	// 0-10 scale -> keep
	assert.InDelta(t, 7.5, normalizeScore(7.5), 0.01)
	// 0-100 scale -> divide by 10
	assert.InDelta(t, 9.5, normalizeScore(95.0), 0.01)
	// <= 0
	assert.Equal(t, 0.0, normalizeScore(0))
	assert.Equal(t, 0.0, normalizeScore(-5))
	// > 100
	assert.Equal(t, 10.0, normalizeScore(150))
}

func TestGetPreferencesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	_, err := adapter.GetPreferences(context.Background())
	require.Error(t, err)
}

func TestLastRefresh(t *testing.T) {
	models := []verifier.Model{
		{ID: "gpt-4", ProviderID: "openai", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 95.0},
	}
	server := makeTestServer(models)
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.APIURL = server.URL
	client := verifier.NewClient(cfg)
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := NewLLMsVerifierScoreAdapter(client, engine, cfg)

	assert.True(t, adapter.LastRefresh().IsZero())

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)

	assert.WithinDuration(t, time.Now(), adapter.LastRefresh(), time.Second)
}
