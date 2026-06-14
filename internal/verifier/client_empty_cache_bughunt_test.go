package verifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestGetVerifiedModels_EmptyResultIsCached is a REPRODUCE-FIRST RED test for a
// cache defect in GetVerifiedModels.
//
// FACT (root cause, client.go): the cache-hit guard is
//
//	if len(c.models) > 0 && time.Since(c.lastFetch) < c.cacheTTL { ... }
//
// When the LLMsVerifier API legitimately returns ZERO verified models (a
// documented, non-error state — see decodeModelsResponse null/empty-envelope
// handling), c.models stays length 0. The `len(c.models) > 0` guard is therefore
// never satisfied, so EVERY subsequent call re-hits the API within the TTL window.
// The cache + TTL are silently defeated for the empty case: a zero-verified-models
// deployment hammers the upstream on every GetVerifiedModels call instead of
// honouring CacheTTL.
//
// Contract: a successful fetch — even one yielding zero models — must populate the
// cache so that, within CacheTTL, a second call does NOT re-hit the server. The
// emptiness of the result is itself a cacheable fact.
func TestGetVerifiedModels_EmptyResultIsCached(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		// Valid envelope, zero verified models (the legitimate empty state).
		_, _ = w.Write([]byte(`{"count":0,"models":[]}`))
	}))
	defer server.Close()

	client := NewClient(&Config{APIURL: server.URL, APIKey: "test", CacheTTL: time.Hour})

	models, err := client.GetVerifiedModels(context.Background())
	if err != nil {
		t.Fatalf("first call must succeed, got: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expected 0 models, got %d", len(models))
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("first call must hit the server exactly once, got %d", got)
	}

	// Second call within the (1h) TTL window MUST be served from cache.
	_, err = client.GetVerifiedModels(context.Background())
	if err != nil {
		t.Fatalf("second call must succeed, got: %v", err)
	}
	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Fatalf("second call within TTL must be served from cache (expected 1 server hit total), got %d", got)
	}
}
