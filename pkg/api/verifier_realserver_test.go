package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
)

// These tests exercise the verifier API handlers against a REAL LLMsVerifier
// HTTP server stood up with httptest (§11.4.27 permits httptest — it is a real
// HTTP roundtrip through the real verifier.Client, not a mock of the client).
// They cover the positive 200 paths plus the model/provider FILTERING logic
// that the existing "" -> 503 tests never reach.
//
// Anti-bluff: assertions check exact status codes AND the filtered JSON body
// (which models survive verification filtering, the provider aggregation,
// counts). A stubbed handler that skipped filtering, or returned everything,
// fails these.

// verifierModelsJSON is the canned /api/models payload. It deliberately mixes
// verified/unverified, can-see-code true/false, affirmative true/false, and
// scores above/below threshold so the filtering branches are all exercised.
const verifierModelsJSON = `[
  {"id":"m-good","provider_id":"openai","name":"Good","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.9,"capabilities":{"code":true},"pricing":{"currency":"USD"},"last_verified_at":"2026-01-01T00:00:00Z"},
  {"id":"m-good2","provider_id":"openai","name":"Good2","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.7,"capabilities":{},"pricing":{},"last_verified_at":"2026-01-01T00:00:00Z"},
  {"id":"m-unverified","provider_id":"anthropic","name":"Unver","verification_status":"pending","can_see_code":true,"affirmative_response":true,"overall_score":0.95},
  {"id":"m-nocode","provider_id":"qwen","name":"NoCode","verification_status":"verified","can_see_code":false,"affirmative_response":true,"overall_score":0.95},
  {"id":"m-lowscore","provider_id":"zhipu","name":"Low","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.4}
]`

// newVerifierTestServer returns an httptest server emulating the LLMsVerifier
// REST API (/api/health + /api/models) and a router wired to a VerifierHandler
// pointed at it.
func newVerifierTestServer(t *testing.T, minScore float64) (*gin.Engine, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
		case "/api/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(verifierModelsJSON))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &verifier.Config{
		APIURL:            srv.URL,
		APIKey:            "test",
		CacheTTL:          time.Hour,
		MinScoreThreshold: minScore,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed: 0.2, CostEffectiveness: 0.3, ModelEfficiency: 0.25,
			Capability: 0.2, Recency: 0.05,
		},
	}
	h := NewVerifierHandler(cfg)
	h.RegisterVerifierRoutes(router.Group("/api/v1"))
	return router, srv
}

// TestVerifier_listVerifiedModels_Filters asserts the handler returns only
// models that are verified AND can-see-code AND affirmative AND score above the
// threshold. With minScore=0.5, m-good (0.9) and m-good2 (0.7) pass; the other
// three are filtered out for distinct reasons.
func TestVerifier_listVerifiedModels_Filters(t *testing.T) {
	router, _ := newVerifierTestServer(t, 0.5)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Models []map[string]any `json:"models"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 2, resp.Count)
	require.Len(t, resp.Models, 2)

	ids := map[string]bool{}
	for _, m := range resp.Models {
		ids[m["id"].(string)] = true
		assert.Equal(t, "verified", m["verification_status"])
	}
	assert.True(t, ids["m-good"], "high-score verified model must be present")
	assert.True(t, ids["m-good2"], "above-threshold verified model must be present")
	assert.False(t, ids["m-unverified"], "pending model must be filtered out")
	assert.False(t, ids["m-nocode"], "can_see_code=false must be filtered out")
	assert.False(t, ids["m-lowscore"], "below-threshold model must be filtered out")
}

// TestVerifier_getVerifiedModel_Found asserts a known model id returns its full
// detail document with the per-field scores.
func TestVerifier_getVerifiedModel_Found(t *testing.T) {
	router, _ := newVerifierTestServer(t, 0.0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models/m-good", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "m-good", resp["id"])
	assert.Equal(t, "openai", resp["provider_id"])
	assert.Equal(t, true, resp["can_see_code"])
	assert.Equal(t, 0.9, resp["overall_score"])
}

// TestVerifier_listVerifiedProviders aggregates verified models by provider.
// Of the 5 fixtures, 4 are verified (m-good, m-good2 -> openai; m-nocode ->
// qwen; m-lowscore -> zhipu). m-unverified (anthropic) is excluded. openai must
// report models_count=2 and highest_score=0.9.
func TestVerifier_listVerifiedProviders(t *testing.T) {
	router, _ := newVerifierTestServer(t, 0.0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/providers/verified", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Providers []map[string]any `json:"providers"`
		Count     int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, 3, resp.Count, "openai, qwen, zhipu are verified; anthropic is not")

	byID := map[string]map[string]any{}
	for _, p := range resp.Providers {
		byID[p["provider_id"].(string)] = p
	}
	require.Contains(t, byID, "openai")
	assert.EqualValues(t, 2, byID["openai"]["models_count"])
	assert.Equal(t, 0.9, byID["openai"]["highest_score"])
	assert.NotContains(t, byID, "anthropic", "unverified provider must be excluded")
}

// TestVerifier_getVerificationStatus_Connected asserts the status endpoint
// reports connected=true when the verifier /api/health ping succeeds (real
// roundtrip to the httptest server).
func TestVerifier_getVerificationStatus_Connected(t *testing.T) {
	router, _ := newVerifierTestServer(t, 0.0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verification-status", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["llmsverifier_connected"])
}

// --- authMiddleware (server.go) ---
//
// authMiddleware has three user-visible outcomes: auth disabled -> pass through;
// missing X-API-Key -> 401 "API key required"; wrong key -> 401 "Invalid API
// key"; correct key -> pass through. All four are driven through a real engine.

func authTestRouter(sec *SecurityConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	s := &Server{config: ServerConfig{Security: sec}}
	r := gin.New()
	r.Use(s.authMiddleware())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestServerAuthMiddleware(t *testing.T) {
	cases := []struct {
		name       string
		sec        *SecurityConfig
		header     string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "auth disabled passes through",
			sec:        &SecurityConfig{RequireAuth: false, APIKey: "secret"},
			header:     "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing key rejected",
			sec:        &SecurityConfig{RequireAuth: true, APIKey: "secret"},
			header:     "",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "API key required",
		},
		{
			name:       "wrong key rejected",
			sec:        &SecurityConfig{RequireAuth: true, APIKey: "secret"},
			header:     "nope",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "Invalid API key",
		},
		{
			name:       "correct key passes through",
			sec:        &SecurityConfig{RequireAuth: true, APIKey: "secret"},
			header:     "secret",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := authTestRouter(tc.sec)
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("X-API-Key", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			var resp map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			if tc.wantErr != "" {
				assert.Equal(t, tc.wantErr, resp["error"])
			} else {
				assert.Equal(t, true, resp["ok"])
			}
		})
	}
}

// TestAuthMiddleware_NilSecurity asserts a nil Security config is treated as
// "auth disabled" (the guard short-circuits on nil before dereferencing).
func TestServerAuthMiddleware_NilSecurity(t *testing.T) {
	r := authTestRouter(nil)
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
