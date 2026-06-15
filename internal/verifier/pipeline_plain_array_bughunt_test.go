package verifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCheckModelExistence_PlainArrayCatalog reproduces a real product defect in
// pipeline.go checkModelExistence: when a provider's /models endpoint returns a
// BARE JSON ARRAY ([{"id":"..."}]) instead of the OpenAI-style envelope
// ({"data":[...]}), the model-existence check must still find the model.
//
// The buggy implementation decoded the envelope form from resp.Body FIRST; on
// failure it attempted a "plain array fallback" by creating a SECOND
// json.Decoder over the SAME, already-consumed resp.Body. The body reader is
// exhausted after the first Decode, so the fallback decoded nothing, the model
// was never matched, and a model genuinely present in the catalog was reported
// "not found" (Passed=false, Score=0.3) — an end-user-visible
// false-negative that would block a perfectly good model from verification.
func TestCheckModelExistence_PlainArrayCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			// Bare array form (no {"data":...} envelope).
			_, _ = w.Write([]byte(`[{"id":"target-model"},{"id":"other-model"}]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	p := NewPipeline()
	provider := ProviderConfig{ID: "p", APIKey: "k", BaseURL: server.URL}

	res := p.checkModelExistence(context.Background(), provider, "target-model")

	if !res.Passed {
		t.Fatalf("model present in plain-array catalog reported NOT found: Passed=%v Score=%v Details=%v",
			res.Passed, res.Score, res.Details)
	}
	if res.Score != 1.0 {
		t.Fatalf("expected full score 1.0 for found model, got %v (Details=%v)", res.Score, res.Details)
	}
	if note, _ := res.Details["note"].(string); strings.Contains(note, "not found") {
		t.Fatalf("model found but result still says not found: %v", res.Details)
	}
}
