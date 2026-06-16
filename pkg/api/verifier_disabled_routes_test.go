package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
)

// §11.4.69 FEATURE: boot_service
//
// RED (pre-fix): the verifier routes were registered in cmd/server/main.go ONLY
// inside `if cfg.LLMsVerifier.Enabled`. On the default-config nezha deploy
// (Enabled=false) the routes were never added to the router, so the documented
// endpoints (HTQ-API-003/004/006) returned 404 — falsely telling clients the API
// does not exist. The fix ALWAYS registers the routes and answers with an honest
// 503 "llmsverifier_disabled" when the integration is off.
//
// This test drives a DISABLED handler through the real gin router and asserts the
// routes EXIST (never 404) and return 503 with the machine-readable reason. It
// FAILS on the pre-fix behaviour (routes absent => 404) and PASSES on the fix.
func newDisabledVerifierRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &verifier.Config{
		APIURL:   "http://127.0.0.1:1", // never contacted while disabled
		CacheTTL: time.Hour,
	}
	h := NewVerifierHandler(cfg)
	h.SetEnabled(false)
	h.RegisterVerifierRoutes(router.Group("/api/v1"))
	return router
}

func TestVerifier_DisabledRoutes_Return503NotFound(t *testing.T) {
	router := newDisabledVerifierRouter(t)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/verified-models", ""},
		{http.MethodGet, "/api/v1/verified-models/m-good", ""},
		{http.MethodGet, "/api/v1/verification-status", ""},
		{http.MethodGet, "/api/v1/providers/verified", ""},
		{http.MethodPost, "/api/v1/verification/refresh", ""},
		{http.MethodPost, "/api/v1/translate-with-verification",
			`{"text":"hi","source_lang":"en","target_lang":"sr"}`},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			router.ServeHTTP(w, req)

			require.NotEqual(t, http.StatusNotFound, w.Code,
				"DEAD-ROUTE BUG: %s %s returned 404 — route must exist even when disabled", tc.method, tc.path)
			require.Equal(t, http.StatusServiceUnavailable, w.Code,
				"disabled route must return honest 503, got %d", w.Code)

			var resp map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, "llmsverifier_disabled", resp["reason"],
				"disabled route must report machine-readable reason")
		})
	}
}

// GREEN guard for the enabled path: an ENABLED handler against a real verifier
// httptest backend serves real data (200) — proving SetEnabled(true) does NOT
// short-circuit. (The full filtering assertions live in verifier_realserver_test.go;
// here we only assert the guard lets the request through.)
func TestVerifier_EnabledRoutes_PassThroughGuard(t *testing.T) {
	router, _ := newVerifierTestServer(t, 0.5) // NewVerifierHandler defaults enabled=true

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "enabled route must serve real data, not 503")
}
