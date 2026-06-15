package bridge

import (
	"context"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/selection"
)

// verifiedModel builds an api.Model that passes the registry's FilterVerified
// criteria (VerificationStatus=="verified" && CanSeeCode && AffirmativeResponse
// && OverallScore > minScore).
func verifiedModel(id, providerID, name string, score float64) verifier.Model {
	return verifier.Model{
		ID:                  id,
		ProviderID:          providerID,
		Name:                name,
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        score,
	}
}

func keyForAll() func(string) string {
	return func(k string) string {
		if strings.HasSuffix(k, "_API_KEY") {
			return "key-" + k
		}
		return ""
	}
}

// TestBridge_ListVerified_StrongestFirst proves ListVerified ranks verified
// models strongest-first (score descending) and assigns FallbackOrder 1..N.
func TestBridge_ListVerified_StrongestFirst(t *testing.T) {
	b := newTestBridge(keyForAll(),
		verifiedModel("deepseek-chat", "deepseek", "DeepSeek Chat", 0.71),
		verifiedModel("gpt-4o", "openai", "GPT-4o", 0.93),
		verifiedModel("groq-llama", "groq", "Groq Llama", 0.55),
	)

	models, err := b.ListVerified(context.Background())
	if err != nil {
		t.Fatalf("ListVerified error: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("got %d models, want 3", len(models))
	}
	// Strongest first.
	if models[0].ModelID != "gpt-4o" {
		t.Errorf("strongest = %q, want gpt-4o", models[0].ModelID)
	}
	if models[2].ModelID != "groq-llama" {
		t.Errorf("weakest = %q, want groq-llama", models[2].ModelID)
	}
	// FallbackOrder is 1..N monotonic.
	for i, m := range models {
		if m.FallbackOrder != i+1 {
			t.Errorf("models[%d].FallbackOrder = %d, want %d", i, m.FallbackOrder, i+1)
		}
	}
	// Each resolved its base_url + factory provider via the resolver.
	if models[0].FactoryName != "openai" {
		t.Errorf("FactoryName = %q, want openai", models[0].FactoryName)
	}
	if models[0].BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want openai v1", models[0].BaseURL)
	}
	// No API key leaks into the public ModelInfo (§11.4.10) — verified by the
	// struct having no key field; this asserts the metadata is the safe subset.
	for _, m := range models {
		if strings.Contains(m.BaseURL, "key-") {
			t.Errorf("API key leaked into BaseURL: %q", m.BaseURL)
		}
	}
}

// TestBridge_BestModel_PicksTopScore proves BestModel returns the single
// highest-scored verified model.
func TestBridge_BestModel_PicksTopScore(t *testing.T) {
	b := newTestBridge(keyForAll(),
		verifiedModel("a", "deepseek", "A", 0.40),
		verifiedModel("b", "openai", "B", 0.88),
	)
	best, err := b.BestModel(context.Background(), selection.TaskRequirements{})
	if err != nil {
		t.Fatalf("BestModel error: %v", err)
	}
	if best.ModelID != "b" {
		t.Errorf("BestModel = %q, want b (top score)", best.ModelID)
	}
	if best.FallbackOrder != 1 {
		t.Errorf("strongest FallbackOrder = %d, want 1", best.FallbackOrder)
	}
}

// TestBridge_FallbackChain_ScoreDescending proves BestTranslator's fallback
// chain (and the ranked list it derives from) is deterministic score-descending —
// a fallback model exists when the top one is unreachable.
func TestBridge_FallbackChain_ScoreDescending(t *testing.T) {
	b := newTestBridge(keyForAll(),
		verifiedModel("m-high", "openai", "High", 0.90),
		verifiedModel("m-mid", "deepseek", "Mid", 0.70),
		verifiedModel("m-low", "groq", "Low", 0.50),
	)
	models, err := b.ListVerified(context.Background())
	if err != nil {
		t.Fatalf("ListVerified error: %v", err)
	}
	wantOrder := []string{"m-high", "m-mid", "m-low"}
	for i, want := range wantOrder {
		if models[i].ModelID != want {
			t.Errorf("rank %d = %q, want %q", i+1, models[i].ModelID, want)
		}
	}
}

// TestBridge_NumericProviderID_HTTPPath proves a verified model carrying a
// NUMERIC ProviderID (the HTTP server path emits fmt.Sprintf("%d", ...))
// materializes through the bridge — the §3.3 part-1 load-bearing bug, exercised
// end-to-end at the bridge layer (not just the resolver unit).
func TestBridge_NumericProviderID_HTTPPath(t *testing.T) {
	// envProviderSpecs[1] is deepseek per the canonical table.
	b := newTestBridge(func(k string) string {
		if k == "DEEPSEEK_API_KEY" {
			return "sk-numeric"
		}
		return ""
	}, verifiedModel("some-model", "1", "Numeric-Provider Model", 0.80))

	models, err := b.ListVerified(context.Background())
	if err != nil {
		t.Fatalf("ListVerified error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
	if models[0].FactoryName == "" {
		t.Errorf("numeric ProviderID resolved with empty FactoryName — not materializable")
	}
	if models[0].BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("numeric ProviderID \"1\" → BaseURL %q, want deepseek v1", models[0].BaseURL)
	}
}

// TestBridge_ListVerified_EmptyIsHonestError proves an empty registry yields an
// honest error, never an empty silent success.
func TestBridge_ListVerified_EmptyIsHonestError(t *testing.T) {
	b := newTestBridge(keyForAll()) // no models
	_, err := b.ListVerified(context.Background())
	if err == nil {
		t.Fatal("ListVerified with no models should error, got nil")
	}
}

// TestBridge_Invoke_RoutesToBestModel is the §1.1 paired-mutation routing guard
// for Review Finding C (commit ab1bed3). It proves that Invoke routes the raw
// completion to EXACTLY the (provider, model) that BestModel selected — not merely
// that it returns "some non-empty result" while silently routing elsewhere.
//
// The dispatch is intercepted via the package-level invokeDispatch seam (no live
// network call), capturing the (providerMarker, modelID) the dispatch is asked to
// route to. The guard then asserts they equal what BestModel independently
// returned for the same bridge.
//
// Polarity switch (§11.4.115): RED_MODE=1 forces a deliberate MIS-ROUTE inside the
// seam (the dispatch routes to a DIFFERENT verified model than BestModel selected),
// proving the guard FAILs on a broken route. RED_MODE=0 (default) is the standing
// GREEN regression guard asserting the route is correct.
func TestBridge_Invoke_RoutesToBestModel(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	// Two distinct verified models with distinct providers + distinct scores so
	// "the model selected" is unambiguous: gpt-4o (openai, 0.93) is strongest;
	// deepseek-chat (deepseek, 0.71) is the wrong route the mutation forces.
	const (
		bestModelID    = "gpt-4o"
		bestProviderID = "openai"
		wrongModelID   = "deepseek-chat"
		wrongProvider  = "deepseek"
	)
	b := newTestBridge(keyForAll(),
		verifiedModel(bestModelID, bestProviderID, "GPT-4o", 0.93),
		verifiedModel(wrongModelID, wrongProvider, "DeepSeek Chat", 0.71),
	)

	// Independently establish what BestModel selects (the oracle for the route).
	best, err := b.BestModel(context.Background(), selection.TaskRequirements{})
	if err != nil {
		t.Fatalf("BestModel error: %v", err)
	}
	if best.ModelID != bestModelID {
		t.Fatalf("test setup: BestModel = %q, want %q", best.ModelID, bestModelID)
	}

	// Intercept the dispatch: record the (providerMarker, modelID) Invoke routes
	// to. NO network call is made — this is the unit-test capability seam.
	var gotProviderMarker, gotModelID string
	const sentinel = "ROUTED-OK"

	orig := invokeDispatch
	t.Cleanup(func() { invokeDispatch = orig })
	invokeDispatch = func(ctx context.Context, providerMarker, modelID string, rp *verifier.ResolvedProvider, full string) (string, error) {
		if redMode {
			// §1.1 MUTATION: simulate a broken route — Invoke's selection is ignored
			// and the dispatch goes to the WRONG model. The guard below MUST catch it.
			modelID = wrongModelID
			providerMarker = wrongProvider
		}
		gotProviderMarker = providerMarker
		gotModelID = modelID
		return sentinel, nil
	}

	out, err := b.Invoke(context.Background(), "you are a router test", "ping")
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if out != sentinel {
		t.Fatalf("Invoke output = %q, want sentinel %q (dispatch result must propagate)", out, sentinel)
	}

	// THE ROUTING GUARD: Invoke must dispatch to EXACTLY the BestModel-selected
	// model. In RED_MODE the mutation rerouted to the wrong model, so this fails.
	if gotModelID != best.ModelID {
		t.Errorf("Invoke routed to model %q, want BestModel-selected %q", gotModelID, best.ModelID)
	}
	// The provider marker must materialize from the SAME selected model's provider
	// (openai → genuine "openai" marker per the delegation rule).
	if gotProviderMarker != bestProviderID {
		t.Errorf("Invoke routed via provider marker %q, want %q", gotProviderMarker, bestProviderID)
	}
}

// TestOpen_NoKeys_HonestHardError proves the OOTB in-process path returns an
// honest hard error (NOT a silent fallback) when no provider key is present —
// the R2 require-keys mandate.
func TestOpen_NoKeys_HonestHardError(t *testing.T) {
	_, err := Open(context.Background(), Options{
		Getenv: func(string) string { return "" }, // no keys, no LLMSVERIFIER_API_URL
	})
	if err == nil {
		t.Fatal("Open with no keys should return an honest hard error, got nil")
	}
	if !strings.Contains(err.Error(), "no provider API keys") {
		t.Errorf("error = %q, want it to name the missing-keys condition", err.Error())
	}
	// MUST NOT mention any local runtime fallback (forbidden by mandate).
	if strings.Contains(strings.ToLower(err.Error()), "llama") {
		// the message intentionally states llama.cpp is NOT permitted; ensure it
		// is framed as a prohibition, not an offered fallback.
		if !strings.Contains(err.Error(), "not permitted") {
			t.Errorf("error must frame llama.cpp as forbidden, got %q", err.Error())
		}
	}
}
