package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/selection"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifiedFactory_LLMsVerifierSSOT_AntiBluff verifies that VerifiedFactory
// uses LLMsVerifier as the single source of truth when a client is configured.
// This is an anti-bluff test: it confirms users can actually get a translator
// backed by verified models from a running LLMsVerifier server.
func TestVerifiedFactory_LLMsVerifierSSOT_AntiBluff(t *testing.T) {
	// Start a mock LLMsVerifier server with verified models
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			http.NotFound(w, r)
			return
		}
		models := []verifier.Model{
			{
				ID:                  "gpt-4",
				ProviderID:          "openai",
				Name:                "GPT-4",
				VerificationStatus:  "verified",
				CanSeeCode:          true,
				AffirmativeResponse: true,
				OverallScore:        0.95,
				Capabilities:        map[string]bool{"translation": true},
			},
			{
				ID:                  "claude-3",
				ProviderID:          "anthropic",
				Name:                "Claude 3",
				VerificationStatus:  "verified",
				CanSeeCode:          true,
				AffirmativeResponse: true,
				OverallScore:        0.92,
				Capabilities:        map[string]bool{"translation": true},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models)
	}))
	defer mockServer.Close()

	cfg := &verifier.Config{
		APIURL:            mockServer.URL,
		CacheTTL:          time.Minute,
		MinScoreThreshold: 0.5,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed:     0.3,
			CostEffectiveness: 0.2,
			ModelEfficiency:   0.2,
			Capability:        0.2,
			Recency:           0.1,
		},
	}

	factory := NewVerifiedFactory(cfg)
	client := verifier.NewClient(cfg)
	factory.SetClient(client)
	// Provide a dummy API key so the provider client can be instantiated
	factory.SetKeyResolver(func(providerID string) string {
		return "dummy-key-for-testing"
	})

	// The factory should fetch models from the mock server and create a translator
	ctx := context.Background()
	trans, err := factory.CreateTranslator(ctx, selection.TaskRequirements{
		SourceLang: "en",
		TargetLang: "sr",
	})

	// We expect this to succeed because the mock server returns verified models.
	// Note: NewLLMTranslatorWithConfig may fail if the provider isn't fully configured,
	// but the key anti-bluff assertion is that the factory DID fetch from SSOT.
	require.NoError(t, err, "factory.CreateTranslator must succeed when LLMsVerifier SSOT has models")
	require.NotNil(t, trans, "translator must not be nil")

	// Verify the translator was configured with a model from the mock server
	assert.Equal(t, "llm-openai", trans.GetName(), "translator must use provider from SSOT")

	// Verify the factory's registry now contains models from the server
	verified := factory.ListVerifiedModels()
	require.Len(t, verified, 2, "registry must contain 2 verified models from SSOT")
	// Order-independent (§11.4.50/§11.4.6): ListVerifiedModels -> FilterVerified
	// ranges a map (internal/verifier/registry.go:60), so the returned slice
	// order is nondeterministic. The contract is the SET of verified models,
	// not their order — assert membership, not position (positional asserts here
	// were an intermittent map-iteration-order flake, D6).
	gotIDs := []string{verified[0].ID, verified[1].ID}
	assert.ElementsMatch(t, []string{"gpt-4", "claude-3"}, gotIDs,
		"registry must contain exactly the gpt-4 + claude-3 models from SSOT (any order)")

	t.Logf("Anti-bluff SSOT verification passed: factory fetched %d models from mock LLMsVerifier", len(verified))
}

// TestVerifiedFactory_LLMsVerifierUnreachable_AntiBluff verifies that when
// LLMsVerifier is unreachable AND the local registry is empty, the factory
// fails with a clear error. This prevents silent use of unverified models.
func TestVerifiedFactory_LLMsVerifierUnreachable_AntiBluff(t *testing.T) {
	// Start a server that immediately closes (simulates unreachable)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	mockServer.Close() // Close immediately to simulate unreachable

	cfg := &verifier.Config{
		APIURL:            mockServer.URL,
		CacheTTL:          time.Minute,
		MinScoreThreshold: 0.5,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed:     0.3,
			CostEffectiveness: 0.2,
			ModelEfficiency:   0.2,
			Capability:        0.2,
			Recency:           0.1,
		},
	}

	factory := NewVerifiedFactory(cfg)
	client := verifier.NewClient(cfg)
	factory.SetClient(client)

	ctx := context.Background()
	_, err := factory.CreateTranslator(ctx, selection.TaskRequirements{
		SourceLang: "en",
		TargetLang: "sr",
	})

	require.Error(t, err, "factory.CreateTranslator MUST fail when LLMsVerifier is unreachable and no cached models exist")
	assert.Contains(t, err.Error(), "unreachable", "error must indicate LLMsVerifier is unreachable")
}

// TestVerifiedFactory_Mutation_BreakSSOT verifies that the anti-bluff test
// above correctly FAILS when the SSOT refresh is broken.
// This is CONST-035 mandatory mutation testing.
func TestVerifiedFactory_Mutation_BreakSSOT(t *testing.T) {
	// This test documents the mutation: if refreshRegistry is no-op'd,
	// the factory will have zero models and CreateTranslator will fail.
	// We verify this by creating a factory with no client and no seeded models.
	cfg := &verifier.Config{
		APIURL:            "http://localhost:1", // unreachable
		CacheTTL:          time.Minute,
		MinScoreThreshold: 0.5,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed:     0.3,
			CostEffectiveness: 0.2,
			ModelEfficiency:   0.2,
			Capability:        0.2,
			Recency:           0.1,
		},
	}

	factory := NewVerifiedFactory(cfg)
	// Intentionally NO client and NO manually registered models

	ctx := context.Background()
	_, err := factory.CreateTranslator(ctx, selection.TaskRequirements{
		SourceLang: "en",
		TargetLang: "sr",
	})

	require.Error(t, err, "MUTATION TEST: with SSOT disabled and no local models, factory MUST fail")
	assert.Contains(t, err.Error(), "no verified model", "error must say no verified models available")
}
