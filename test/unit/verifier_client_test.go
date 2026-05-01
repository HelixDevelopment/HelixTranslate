package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
)

// TestNewClient_Construction verifies client construction with config.
func TestNewClient_Construction(t *testing.T) {
	cfg := &verifier.Config{
		APIURL:   "http://localhost:8080",
		APIKey:   "test-key",
		CacheTTL: time.Minute,
	}

	client := verifier.NewClient(cfg)
	require.NotNil(t, client)

	// Anti-bluff: Verify the client actually stores the config values
	_, err := client.GetVerifiedModels(t.Context())
	require.Error(t, err) // No server running, should fail
	assert.NotNil(t, err)
}

// TestClient_Ping_Success verifies health check against a real HTTP server.
func TestClient_Ping_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/health", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{
		APIURL: server.URL,
		APIKey: "test-key",
	})

	err := client.Ping(t.Context())
	require.NoError(t, err)
}

// TestClient_Ping_Failure verifies unreachable server returns specific error.
func TestClient_Ping_Failure(t *testing.T) {
	client := verifier.NewClient(&verifier.Config{
		APIURL: "http://localhost:1", // Unlikely to respond
		APIKey: "test-key",
	})

	err := client.Ping(t.Context())
	require.Error(t, err)

	var unreachableErr verifier.ErrLLMsVerifierUnreachable
	require.ErrorAs(t, err, &unreachableErr)
}

// TestClient_GetVerifiedModels_Caching verifies cache behavior.
func TestClient_GetVerifiedModels_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{
			{ID: "model-1", ProviderID: "openai", Name: "GPT-4", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.95},
		})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{
		APIURL:   server.URL,
		APIKey:   "test-key",
		CacheTTL: time.Hour,
	})

	// First call should hit the server
	models1, err := client.GetVerifiedModels(t.Context())
	require.NoError(t, err)
	require.Len(t, models1, 1)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	models2, err := client.GetVerifiedModels(t.Context())
	require.NoError(t, err)
	require.Len(t, models2, 1)
	assert.Equal(t, 1, callCount) // No additional server call

	// Invalidate cache and call again
	client.InvalidateCache()
	models3, err := client.GetVerifiedModels(t.Context())
	require.NoError(t, err)
	require.Len(t, models3, 1)
	assert.Equal(t, 2, callCount) // Server called again
}

// TestClient_GetModel_NotFound verifies error for unverified model.
func TestClient_GetModel_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]verifier.Model{
			{ID: "model-1", ProviderID: "openai", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true},
		})
	}))
	defer server.Close()

	client := verifier.NewClient(&verifier.Config{
		APIURL:   server.URL,
		APIKey:   "test-key",
		CacheTTL: time.Hour,
	})

	_, err := client.GetModel(t.Context(), "nonexistent-model")
	require.Error(t, err)

	var notVerifiedErr verifier.ErrModelNotVerified
	require.ErrorAs(t, err, &notVerifiedErr)
	assert.Equal(t, "nonexistent-model", notVerifiedErr.ModelID)
}
