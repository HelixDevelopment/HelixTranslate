package api

import (
	"bytes"
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

// newVerifierServerWithBackend builds a VerifierHandler router whose upstream
// LLMsVerifier behaves per the supplied /api/models handler, so a test can make
// the upstream healthy (return models) or broken (return 5xx) on demand.
func newVerifierServerWithBackend(t *testing.T, modelsHandler http.HandlerFunc) *gin.Engine {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusOK)
		case "/api/models":
			modelsHandler(w, r)
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
		MinScoreThreshold: 0.0,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed: 0.2, CostEffectiveness: 0.3, ModelEfficiency: 0.25,
			Capability: 0.2, Recency: 0.05,
		},
	}
	h := NewVerifierHandler(cfg)
	h.RegisterVerifierRoutes(router.Group("/api/v1"))
	return router
}

const oneVerifiedModelJSON = `[
  {"id":"m-good","provider_id":"openai","name":"Good","verification_status":"verified","can_see_code":true,"affirmative_response":true,"overall_score":0.9}
]`

// TestVerifier_getVerifiedModel_UpstreamDown_Is503 — when the LLMsVerifier
// upstream is UNREACHABLE/erroring, GetModel returns a transport error (NOT a
// not-found). Reporting 404 ("model not found") in that case is a wrong
// error-to-status mapping: it tells the client the specific model is absent when
// the truth is the whole upstream is down. The correct code is 503.
// Pre-fix the handler maps every GetModel error to 404 → this FAILs.
func TestVerifier_getVerifiedModel_UpstreamDown_Is503(t *testing.T) {
	router := newVerifierServerWithBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // upstream broken
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models/m-good", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code,
		"upstream-down GetModel failure must map to 503, not 404")
}

// TestVerifier_getVerifiedModel_NotFound_Is404 — when the upstream is healthy
// but the requested model id is genuinely not present, 404 is correct. This is
// the regression guard for the 503 fix (a real not-found must NOT become 503).
func TestVerifier_getVerifiedModel_NotFound_Is404(t *testing.T) {
	router := newVerifierServerWithBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oneVerifiedModelJSON))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models/m-DOES-NOT-EXIST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"genuinely-absent model must map to 404")
}

// TestVerifier_translateWithVerification_MissingModel_Is404 — requesting
// translation with an explicit model_id that does not exist must be 404
// (resource not found), NOT 400 (malformed request — the request is well-formed,
// the model simply isn't there). Pre-fix the handler maps the GetModel error to
// 400 → this FAILs. Consistency with getVerifiedModel's not-found mapping.
func TestVerifier_translateWithVerification_MissingModel_Is404(t *testing.T) {
	router := newVerifierServerWithBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oneVerifiedModelJSON))
	})

	body, _ := json.Marshal(map[string]string{
		"text":        "hello",
		"source_lang": "en",
		"target_lang": "sr",
		"model_id":    "m-DOES-NOT-EXIST",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate-with-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"translate with a missing model_id must map to 404, not 400")
}

// TestVerifier_translateWithVerification_UpstreamDown_Is503 — explicit model_id
// but upstream broken must be 503, not 400/404.
func TestVerifier_translateWithVerification_UpstreamDown_Is503(t *testing.T) {
	router := newVerifierServerWithBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	body, _ := json.Marshal(map[string]string{
		"text":        "hello",
		"source_lang": "en",
		"target_lang": "sr",
		"model_id":    "m-good",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate-with-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code,
		"translate with upstream down must map to 503, not 400")
}

// TestVerifier_translateWithVerification_ValidModel_Is200 — regression guard:
// a real model_id on a healthy upstream still succeeds.
func TestVerifier_translateWithVerification_ValidModel_Is200(t *testing.T) {
	router := newVerifierServerWithBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(oneVerifiedModelJSON))
	})

	body, _ := json.Marshal(map[string]string{
		"text":        "hello",
		"source_lang": "en",
		"target_lang": "sr",
		"model_id":    "m-good",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate-with-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "m-good", resp["model_id"])
}
