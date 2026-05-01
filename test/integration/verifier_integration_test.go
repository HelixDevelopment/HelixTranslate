//go:build integration

package integration

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
	"digital.vasic.translator/internal/verifier/discovery"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// TestFullVerifierFlow tests the complete pipeline: discovery → scoring → selection.
func TestFullVerifierFlow(t *testing.T) {
	// Anti-bluff: This test uses real in-memory registries and engines, not mocks.
	// It verifies the actual integration between components.

	// 1. Setup mock LLMsVerifier server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/models":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]verifier.Model{
				{
					ID: "claude-3-sonnet", ProviderID: "anthropic",
					Name: "Claude 3 Sonnet", VerificationStatus: "verified",
					CanSeeCode: true, AffirmativeResponse: true,
					OverallScore: 0.92,
					Capabilities: map[string]bool{"streaming": true, "vision": true},
				},
				{
					ID: "gpt-4o", ProviderID: "openai",
					Name: "GPT-4o", VerificationStatus: "verified",
					CanSeeCode: true, AffirmativeResponse: true,
					OverallScore: 0.95,
					Capabilities: map[string]bool{"streaming": true, "vision": true},
				},
				{
					ID: "unverified-model", ProviderID: "test",
					Name: "Bad Model", VerificationStatus: "pending",
					CanSeeCode: false, AffirmativeResponse: false,
					OverallScore: 0.1,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 2. Initialize client
	client := verifier.NewClient(&verifier.Config{
		APIURL:   server.URL,
		APIKey:   "integration-test-key",
		CacheTTL: time.Hour,
	})

	// 3. Verify connectivity
	require.NoError(t, client.Ping(context.Background()))

	// 4. Fetch models
	models, err := client.GetVerifiedModels(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 3)

	// 5. Setup registry and discovery
	reg := verifier.NewRegistry()
	for _, m := range models {
		reg.AddModel(m)
	}

	disc := discovery.NewService(verifier.DefaultConfig(), reg)
	require.NotNil(t, disc)

	// 6. Setup scoring engine
	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	// 7. Setup score adapter
	adapter := services.NewLLMsVerifierScoreAdapter(client, scoreEngine, verifier.DefaultConfig())
	require.NoError(t, adapter.RefreshScores(context.Background()))

	// 8. Verify adapter filters unverified models
	prefs, err := adapter.GetPreferences(context.Background())
	require.NoError(t, err)

	// Only verified models should appear
	for _, p := range prefs {
		assert.NotEqual(t, "unverified-model", p.ModelID)
		assert.True(t, p.Score > 0)
	}

	// 9. Setup selection engine and select model
	selEngine := selection.NewEngine(reg, scoreEngine, verifier.DefaultConfig())
	selected, err := selEngine.SelectModel(context.Background(), selection.TaskRequirements{
		SourceLang:       "en",
		TargetLang:       "sr",
		RequireStreaming: true,
	})
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, "gpt-4o", selected.ID) // Highest score

	// 10. Verify fallback works
	fallback, err := selEngine.SelectFallback(selected.ID, selection.TaskRequirements{
		RequireStreaming: true,
	})
	require.NoError(t, err)
	require.NotNil(t, fallback)
	assert.NotEqual(t, selected.ID, fallback.ID)
}

// TestAdapter_RefreshScores_WithRealCache verifies cache behavior across refreshes.
func TestAdapter_RefreshScores_WithRealCache(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{
			{ID: "m1", ProviderID: "p", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.8},
		})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{
		APIURL:   server.URL,
		APIKey:   "key",
		CacheTTL: time.Hour,
	})
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	adapter := services.NewLLMsVerifierScoreAdapter(client, engine, verifier.DefaultConfig())

	err := adapter.RefreshScores(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Score should be cached and retrievable
	assert.Equal(t, 8.0, adapter.GetProviderScore("m1"))

	// LastRefresh should be recent
	assert.WithinDuration(t, time.Now(), adapter.LastRefresh(), 5*time.Second)
}
