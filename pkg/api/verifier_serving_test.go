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

// These tests DEEPEN the verifier-serving coverage beyond
// verifier_realserver_test.go (which drives the forward-compatible bare
// []api.Model payload) and server_realhttp_integration_test.go. They exercise
// the REAL VerifierHandler over a REAL net/http/httptest LLMsVerifier upstream
// for the gaps those tests do not reach:
//
//   1. The REAL server ENVELOPE shape ({"count":N,"models":[{model_id,score,
//      status,provider_id,capabilities,...}]}) that the production server emits
//      (see docs/qa/llmsverifier_20260613_182600/api_models_head.json) — the
//      bare-array path is a forward-compat fallback, NOT what production sends.
//   2. Error/edge serving paths: upstream unreachable, empty upstream, malformed
//      upstream payload — each asserting a concrete user-visible outcome (an
//      exact status code, an empty-but-well-formed list, an honest error, never
//      a 200 with fabricated models, never a panic).
//
// Anti-bluff (§11.4 / §11.4.69): every PASS asserts the exact model set / status
// code / response field a client consumes — never "no error". §1.1 mutation note
// is on TestVerifierServing_RealServerEnvelope_FiltersAndServes below.

// realServerEnvelope mirrors the production LLMsVerifier /api/models envelope
// shape (per the captured fixture): integer ids, model_id, integer score,
// status, provider_id, capabilities[]. It mixes a high-score verified model, a
// below-threshold verified model, and a non-verified model so the filtering
// branches are exercised against the REAL envelope decoder + handler.
const realServerEnvelope = `{
  "count": 3,
  "models": [
    {"id":1,"model_id":"deepseek-v4-flash","name":"deepseek-v4-flash","provider":"DeepSeek","provider_id":1,"score":95,"status":"verified","capabilities":["text"],"version":"live"},
    {"id":50,"model_id":"weak-model","name":"weak-model","provider":"Groq","provider_id":3,"score":40,"status":"verified","capabilities":["text"],"version":"live"},
    {"id":77,"model_id":"pending-model","name":"pending-model","provider":"Mistral","provider_id":4,"score":99,"status":"pending","capabilities":["text"],"version":"live"}
  ]
}`

// newEnvelopeUpstream stands up a real httptest LLMsVerifier upstream serving the
// given /api/models body (and a 200 /api/health), wires a real VerifierHandler at
// /api/v1, and returns the router. minScore is the OverallScore threshold; the
// server envelope reports score on a 0-100 scale so the handler config threshold
// is set in the same scale by the caller.
func newEnvelopeUpstream(t *testing.T, modelsBody string, healthStatus, modelsStatus int, minScore float64) *gin.Engine {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(healthStatus)
		case "/api/models":
			if modelsStatus != http.StatusOK {
				w.WriteHeader(modelsStatus)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(modelsBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewVerifierHandler(&verifier.Config{
		APIURL:            srv.URL,
		APIKey:            "test",
		CacheTTL:          time.Hour,
		MinScoreThreshold: minScore,
	})
	h.RegisterVerifierRoutes(router.Group("/api/v1"))
	return router
}

func getVerifiedModels(t *testing.T, router *gin.Engine) (int, []map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		return w.Code, nil
	}
	var resp struct {
		Models []map[string]any `json:"models"`
		Count  int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "verified-models body is valid JSON")
	require.Equal(t, len(resp.Models), resp.Count, "count field matches the served model array length")
	return w.Code, resp.Models
}

// TestVerifierServing_RealServerEnvelope_FiltersAndServes is the headline test:
// against the REAL production envelope shape, the endpoint MUST serve the
// qualifying verified model(s) — not an empty list. This reproduces and guards
// the serving-layer regression where the envelope decoder leaves
// CanSeeCode/AffirmativeResponse false (the server schema does not carry them),
// and the handler's unconditional !CanSeeCode||!AffirmativeResponse gate then
// rejected EVERY real model, serving count=0 while N models were verified
// upstream.
//
// Threshold is set to 50 (server score scale 0-100): deepseek-v4-flash (95,
// verified) survives; weak-model (40, verified, below threshold) is filtered;
// pending-model (99, status=pending) is filtered. Exact surviving set asserted.
//
// §1.1 MUTATION (the assertion that makes this FAIL if the fix is removed):
// `require.Equal(t, 1, len(models))` together with
// `assert.Equal(t, "deepseek-v4-flash", models[0]["id"])`. If the production
// handler filter reverts to the unconditional `!m.CanSeeCode ||
// !m.AffirmativeResponse` reject (i.e. the fix is removed), every envelope model
// decodes with both flags false and is filtered out -> len(models)==0 -> this
// assertion FAILs. (Verified by re-introducing the old condition: this test
// reported `expected 1, got 0`.)
func TestVerifierServing_RealServerEnvelope_FiltersAndServes(t *testing.T) {
	router := newEnvelopeUpstream(t, realServerEnvelope, http.StatusOK, http.StatusOK, 50)

	code, models := getVerifiedModels(t, router)
	require.Equal(t, http.StatusOK, code, "verified-models returns 200 for a reachable upstream")

	require.Equal(t, 1, len(models),
		"exactly the verified, above-threshold model is served (envelope path must not filter all out)")

	m := models[0]
	// The response shape clients consume — real fields, real values.
	assert.Equal(t, "deepseek-v4-flash", m["id"], "the real model_id is surfaced as id")
	assert.Equal(t, "verified", m["verification_status"], "status maps to verification_status")
	assert.InDelta(t, 95.0, m["overall_score"], 0.0001, "the real numeric score round-trips")
	assert.Equal(t, "1", m["provider_id"], "the numeric provider_id is surfaced as a string id")
	// capabilities is the client-facing capability map decoded from the
	// server's capabilities[] array.
	caps, ok := m["capabilities"].(map[string]any)
	require.True(t, ok, "capabilities is an object the client can read")
	assert.Equal(t, true, caps["text"], "the 'text' capability from the envelope is present and true")

	// Negative assertions: the filtered-out models MUST NOT appear.
	for _, mm := range models {
		assert.NotEqual(t, "weak-model", mm["id"], "below-threshold verified model must be filtered out")
		assert.NotEqual(t, "pending-model", mm["id"], "non-verified model must be filtered out")
	}
}

// TestVerifierServing_EmptyUpstream_WellFormedEmptyList: an upstream with zero
// models yields an empty-but-well-formed list (count=0, models=[]), not an error
// and not a fabricated entry. A client can render "no verified models" safely.
func TestVerifierServing_EmptyUpstream_WellFormedEmptyList(t *testing.T) {
	router := newEnvelopeUpstream(t, `{"count":0,"models":[]}`, http.StatusOK, http.StatusOK, 0)

	code, models := getVerifiedModels(t, router)
	require.Equal(t, http.StatusOK, code, "empty upstream still returns 200")
	assert.Len(t, models, 0, "empty upstream yields an empty model list, never a fabricated entry")
}

// TestVerifierServing_UpstreamUnreachable_503: when the LLMsVerifier upstream
// cannot be reached, the endpoint MUST return a sane error status (503), NEVER a
// 200 carrying fake models. The handler maps the client's
// ErrLLMsVerifierUnreachable to 503 with an error body.
func TestVerifierServing_UpstreamUnreachable_503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Point at a closed port (no listener) so the HTTP roundtrip genuinely fails.
	h := NewVerifierHandler(&verifier.Config{
		APIURL:            "http://127.0.0.1:1", // port 1: connection refused
		APIKey:            "test",
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0,
	})
	h.RegisterVerifierRoutes(router.Group("/api/v1"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"unreachable upstream returns 503, not a 200 with fabricated models")
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "error body is valid JSON")
	assert.Equal(t, "failed to fetch verified models", resp["error"], "honest error surfaced to the client")
	assert.NotContains(t, resp, "models", "no models key on the error path — no fabricated list")
}

// TestVerifierServing_UpstreamNon200_503: an upstream that answers /api/models
// with a non-200 (e.g. 500) MUST surface as 503 to the client, never a 200 with
// fabricated models.
func TestVerifierServing_UpstreamNon200_503(t *testing.T) {
	router := newEnvelopeUpstream(t, "", http.StatusOK, http.StatusInternalServerError, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"upstream 500 surfaces as 503, never a 200 with fake models")
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "failed to fetch verified models", resp["error"])
}

// TestVerifierServing_MalformedPayload_HandledNoPanic: a malformed upstream
// payload (neither a valid envelope nor a valid []api.Model array) MUST be
// handled — the endpoint returns 503 with an honest decode error and does NOT
// panic, and does NOT return a 200 with fabricated models. Driving the real
// router catches a panic as a test failure (httptest recovers via gin.New()'s
// default — we assert the status instead).
func TestVerifierServing_MalformedPayload_HandledNoPanic(t *testing.T) {
	// `{"models": 12345}` — `models` is a number, not an array; both the
	// envelope decode and the bare-array fallback fail honestly.
	router := newEnvelopeUpstream(t, `{"models": 12345}`, http.StatusOK, http.StatusOK, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	require.NotPanics(t, func() { router.ServeHTTP(w, req) },
		"malformed upstream payload must not panic the handler")

	require.Equal(t, http.StatusServiceUnavailable, w.Code,
		"malformed payload surfaces as 503, never a 200 with fabricated models")
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "failed to fetch verified models", resp["error"], "honest decode error surfaced")
}

// TestVerifierServing_EnvelopeRespectsThreshold confirms the threshold filter is
// live on the envelope path (not only on the bare-array path): raising the
// threshold above the only verified model's score yields an empty list, proving
// the score comparison genuinely runs against the envelope-decoded OverallScore.
func TestVerifierServing_EnvelopeRespectsThreshold(t *testing.T) {
	// deepseek-v4-flash score=95; threshold 96 must exclude it.
	router := newEnvelopeUpstream(t, realServerEnvelope, http.StatusOK, http.StatusOK, 96)

	code, models := getVerifiedModels(t, router)
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, models, 0, "a threshold above every score yields an empty list (score filter is live on the envelope path)")
}
