package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"
	"github.com/stretchr/testify/require"
)

// TestHuggingFaceDecodeUsesCanonicalID is a RED regression test (Wave bug-hunt).
// FACT: the HuggingFace list API's canonical model identifier is the "id" field;
// "modelId" is a legacy alias the API contract does NOT guarantee. The decode
// path keyed registration on m.ModelID only, so a valid HF payload that supplies
// only "id" registered every model with ID="" — all colliding on the empty-string
// registry key, silently yielding ZERO usable models while the server is healthy.
func TestHuggingFaceDecodeUsesCanonicalID(t *testing.T) {
	// A valid HuggingFace-shaped payload using ONLY the canonical "id" field.
	const payload = `[
	  {"id":"bert-base-uncased","downloads":100,"tags":["pytorch"]},
	  {"id":"gpt2","downloads":200,"tags":["transformers"]},
	  {"id":"distilbert-base-uncased","downloads":50,"tags":["pytorch"]}
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	reg := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	svc := NewService(cfg, reg)
	svc.httpClient = &http.Client{Timeout: 2 * time.Second}
	svc.huggingFaceURL = srv.URL

	err := svc.discoverFromHuggingFace(context.Background())
	require.NoError(t, err)

	models := reg.ListModels()
	// Three distinct models supplied -> three must be registered with non-empty IDs.
	require.Len(t, models, 3, "all 3 HF models must register; empty ID collides them to 1 in the registry map")
	for _, m := range models {
		require.NotEmpty(t, m.ID, "HF model registered with empty ID (modelId absent, canonical id dropped)")
	}
}
