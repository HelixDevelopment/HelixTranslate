package models

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
)

func setupMockVerifierServer(t *testing.T) (*httptest.Server, *VerifierRegistry) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/api/models":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]verifier.Model{
				{
					ID: "gpt-4o", ProviderID: "openai", Name: "GPT-4o",
					VerificationStatus: "verified", CanSeeCode: true,
					AffirmativeResponse: true, OverallScore: 0.95,
					Capabilities: map[string]bool{"streaming": true},
				},
				{
					ID: "bad-model", ProviderID: "test", Name: "Bad",
					VerificationStatus: "failed", CanSeeCode: false,
					AffirmativeResponse: false, OverallScore: 0.0,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	cfg := &verifier.Config{
		APIURL:            server.URL,
		APIKey:            "test",
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.5,
	}
	return server, NewVerifierRegistry(cfg)
}

func TestVerifierRegistry_Refresh(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	err := reg.Refresh(context.Background())
	require.NoError(t, err)

	models := reg.ListModels()
	require.Len(t, models, 2)
}

func TestVerifierRegistry_ListVerifiedModels(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	require.NoError(t, reg.Refresh(context.Background()))

	verified := reg.ListVerifiedModels()
	require.Len(t, verified, 1)
	assert.Equal(t, "gpt-4o", verified[0].ID)
}

func TestVerifierRegistry_IsModelVerified(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	require.NoError(t, reg.Refresh(context.Background()))

	assert.True(t, reg.IsModelVerified("gpt-4o"))
	assert.False(t, reg.IsModelVerified("bad-model"))
	assert.False(t, reg.IsModelVerified("unknown"))
}

func TestVerifierRegistry_GetModel(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	m, err := reg.GetModel(context.Background(), "gpt-4o")
	require.NoError(t, err)
	assert.Equal(t, "GPT-4o", m.Name)

	// Second call should use cache
	m2, err := reg.GetModel(context.Background(), "gpt-4o")
	require.NoError(t, err)
	assert.Equal(t, m.Name, m2.Name)
}

func TestVerifierRegistry_HealthCheck(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	err := reg.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestVerifierRegistry_ToModelInfo(t *testing.T) {
	server, reg := setupMockVerifierServer(t)
	defer server.Close()

	info := reg.ToModelInfo(verifier.Model{
		ID:           "test-model",
		Name:         "Test Model",
		ProviderID:   "openai",
		OverallScore: 0.95,
	})

	assert.Equal(t, "test-model", info.ID)
	assert.Equal(t, "Test Model", info.Name)
	assert.Equal(t, "excellent", info.Quality)

	info2 := reg.ToModelInfo(verifier.Model{OverallScore: 0.6})
	assert.Equal(t, "moderate", info2.Quality)

	info3 := reg.ToModelInfo(verifier.Model{OverallScore: 0.3})
	assert.Equal(t, "poor", info3.Quality)
}
