package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive the version-monitoring HTTP handlers through a REAL gin
// engine (httptest) with no distributed backend wired in. Every handler in this
// family guards on `h.distributedManager == nil` and MUST return 503 with a
// specific JSON error body — that contract is fully unit-testable without
// Postgres/Redis/SSH workers and is exactly the end-user-visible behaviour an
// operator hits when distributed mode is disabled (the common default).
//
// Anti-bluff (§11.4.27 / §11.4.1): each test asserts the concrete status code
// AND the JSON error string the handler emits. A stubbed handler that returned
// 200 (or omitted the guard) fails these. See the mutation note in the package
// report.

// monitoringRouter wires the version-monitoring routes onto a fresh engine with
// a Handler that has NO distributed manager (the nil-guard path).
func monitoringRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Handler{} // distributedManager is nil
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.GET("/monitoring/version/metrics", h.getVersionMetrics)
	v1.GET("/monitoring/version/alerts", h.getVersionAlerts)
	v1.GET("/monitoring/version/health", h.getVersionHealth)
	v1.GET("/monitoring/version/dashboard", h.getVersionDashboard)
	v1.POST("/monitoring/version/drift-check", h.triggerVersionDriftCheck)
	v1.GET("/monitoring/version/alerts/history", h.getAlertHistory)
	v1.POST("/monitoring/version/alerts/:alert_id/acknowledge", h.acknowledgeAlert)
	v1.POST("/monitoring/version/alerts/channels/email", h.addEmailAlertChannel)
	v1.POST("/monitoring/version/alerts/channels/webhook", h.addWebhookAlertChannel)
	v1.POST("/monitoring/version/alerts/channels/slack", h.addSlackAlertChannel)
	v1.GET("/monitoring/version/dashboard.html", h.serveDashboard)
	return r
}

func doReq(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *strings.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	} else {
		rdr = strings.NewReader("")
	}
	req, err := http.NewRequest(method, path, rdr)
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestVersionMonitoring_NilManager_Returns503 asserts every read/write
// monitoring endpoint surfaces 503 + the documented error when distributed work
// is unavailable. This is the user-visible contract, not internal state.
func TestVersionMonitoring_NilManager_Returns503(t *testing.T) {
	r := monitoringRouter()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"metrics", http.MethodGet, "/api/v1/monitoring/version/metrics", ""},
		{"alerts", http.MethodGet, "/api/v1/monitoring/version/alerts", ""},
		{"health", http.MethodGet, "/api/v1/monitoring/version/health", ""},
		{"dashboard", http.MethodGet, "/api/v1/monitoring/version/dashboard", ""},
		{"drift-check", http.MethodPost, "/api/v1/monitoring/version/drift-check", ""},
		{"alerts-history", http.MethodGet, "/api/v1/monitoring/version/alerts/history", ""},
		{"email-channel", http.MethodPost, "/api/v1/monitoring/version/alerts/channels/email",
			`{"smtp_host":"h","smtp_port":25,"username":"u","password":"p","from_address":"f@x","to_addresses":["t@x"]}`},
		{"webhook-channel", http.MethodPost, "/api/v1/monitoring/version/alerts/channels/webhook",
			`{"url":"http://x"}`},
		{"slack-channel", http.MethodPost, "/api/v1/monitoring/version/alerts/channels/slack",
			`{"webhook_url":"http://x"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doReq(t, r, tc.method, tc.path, tc.body)
			assert.Equal(t, http.StatusServiceUnavailable, w.Code)

			var resp map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, "Distributed work not available", resp["error"])
		})
	}
}

// TestAcknowledgeAlert_NilManager_503 documents that the nil-guard fires before
// param/body validation for acknowledgeAlert.
func TestAcknowledgeAlert_NilManager_503(t *testing.T) {
	r := monitoringRouter()
	w := doReq(t, r, http.MethodPost,
		"/api/v1/monitoring/version/alerts/abc123/acknowledge",
		`{"acknowledged_by":"alice"}`)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Distributed work not available", resp["error"])
}

// TestServeDashboard_EmbeddedFallback asserts serveDashboard always serves HTML
// with the correct content-type. In the test working directory the on-disk
// dashboard.html relative path ("pkg/api/dashboard.html") does not resolve, so
// the embedded fallback HTML is served — this exercises getEmbeddedDashboardHTML
// end-to-end and asserts user-visible markers in the rendered page.
func TestServeDashboard_EmbeddedFallback(t *testing.T) {
	r := monitoringRouter()
	w := doReq(t, r, http.MethodGet, "/api/v1/monitoring/version/dashboard.html", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

	body := w.Body.String()
	assert.True(t, strings.HasPrefix(strings.TrimSpace(body), "<!DOCTYPE html>"),
		"dashboard must be served as an HTML document")
	// Marker unique to the embedded fallback page.
	assert.Contains(t, body, "Version Management Dashboard")
	assert.Contains(t, body, "/api/v1/monitoring/version/dashboard")
}

// TestGetEmbeddedDashboardHTML_NonDegenerate asserts the embedded HTML is a
// real, non-empty HTML document (anti-bluff: a stub returning "" fails).
func TestGetEmbeddedDashboardHTML_NonDegenerate(t *testing.T) {
	h := &Handler{}
	got := h.getEmbeddedDashboardHTML()
	assert.Greater(t, len(got), 500, "embedded dashboard HTML must be substantive")
	assert.Contains(t, got, "<!DOCTYPE html>")
	assert.Contains(t, got, "</html>")
	assert.Contains(t, got, "loadData") // the JS fetch hook the page relies on
}
