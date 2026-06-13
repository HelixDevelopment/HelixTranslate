package discovery

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// sentinelTier2 points a Service's Tier-2 registry endpoints (OpenRouter,
// HuggingFace) at a closed local TCP port so they fail FAST instead of dialing
// the live public registries. Combined with a short httpClient timeout this
// keeps the Discover() pipeline exercising its real logic + Tier-1/Tier-3
// assertions while guaranteeing the suite never hangs on internet reachability
// (§11.4.3 topology-aware dispatch — the live Tier-2 registries are absent in
// CI/sandbox, so we drive the pipeline against a deterministic local fail).
//
// It returns the URL of a guaranteed-unreachable endpoint (a server that is
// started then immediately closed, so the port is reserved-then-refused).
func sentinelTier2(t *testing.T, s *Service) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately → connections to url are refused quickly
	s.httpClient = &http.Client{Timeout: 2 * time.Second}
	s.openRouterURL = url
	s.huggingFaceURL = url
	return url
}

// tier2Reachable probes whether the live public Tier-2 registry is reachable
// within a 2s budget. Tests that genuinely require the live registry call this
// and t.Skip when it returns false (§11.4.3 honest SKIP, never a fail-open
// PASS and never a 30s-per-call hang).
func tier2Reachable(host string) bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", net.JoinHostPort(host, "443"))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
