package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// §11.4.115 RED-baseline-on-the-broken-artifact + polarity-switch.
//
// This test proves the previously-DORMANT pkg/verification multi-pass polishing
// engine is genuinely INVOKED through the unified-translator `-multipass` wiring
// and that its LLM-produced polished text flows back into the result.
//
// RED (pre-wire): runMultiPass / markdownToSingleSectionBook did not exist, so
// this file does not compile -> the package test build FAILS. That compile
// failure IS the captured "defect present on the pre-fix artifact" evidence: the
// engine was unreachable from the CLI.
//
// GREEN (post-wire): runMultiPass exists, builds the verification engine from the
// CLI config, calls a real LLM (here a local httptest OpenAI-compatible server),
// parses the verification response, and applies POLISHED_TEXT. The assertions
// below fail if the engine is NOT actually invoked (server never hit) or if the
// polished text is NOT applied (output unchanged), so a no-op stub cannot pass.

// openAICompatHandler returns an httptest server that speaks the OpenAI
// chat-completions wire format (DeepSeek is OpenAI-compatible) and replies with a
// verification-formatted response whose POLISHED_TEXT differs from the input —
// so a correctly-wired engine MUST change the translation. hits counts requests.
func openAICompatHandler(t *testing.T, hits *int64, polishedText string, lastPrompt *atomicString) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(hits, 1)

		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Messages) > 0 {
			lastPrompt.Store(req.Messages[len(req.Messages)-1].Content)
		}

		// A full verification-format response the engine's parser understands.
		content := "SPIRIT_SCORE: 0.95\n" +
			"LANGUAGE_SCORE: 0.9\n" +
			"CONTEXT_SCORE: 0.92\n" +
			"VOCABULARY_SCORE: 0.88\n\n" +
			"ISSUES:\n" +
			"VOCABULARY: minor word-choice refinement\n\n" +
			"POLISHED_TEXT:\n" + polishedText + "\n\n" +
			"EXPLANATION:\nRefined vocabulary for naturalness."

		resp := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func TestRunMultiPass_InvokesEngineAndAppliesPolish(t *testing.T) {
	const inputTranslation = "Ovo je osnovni prevod koji treba poboljsati."
	const polished = "Ovo je doteran, prirodniji prevod posle poliranja."

	var hits int64
	var lastPrompt atomicString
	srv := httptest.NewServer(openAICompatHandler(t, &hits, polished, &lastPrompt))
	defer srv.Close()

	cfg := &UnifiedConfig{
		SourceLang:     "ru",
		TargetLang:     "sr",
		Provider:       "deepseek", // OpenAI-compatible, honors BaseURL
		Model:          "deepseek-chat",
		APIKey:         "test-key", // local server, no real secret
		BaseURL:        srv.URL,
		Temperature:    0.3,
		MaxTokens:      512,
		Timeout:        10 * time.Second,
		MultiPass:      true,
		MultiPassCount: 1,
	}
	session := &TranslationSession{
		ID:       "multipass-test-session",
		EventBus: events.NewEventBus(),
	}

	out, err := runMultiPass(context.Background(), cfg, session, "This is the basic translation to improve.", inputTranslation)
	if err != nil {
		t.Fatalf("runMultiPass returned error: %v", err)
	}

	// 1. The engine actually called the LLM (proves it is invoked, not a stub).
	if atomic.LoadInt64(&hits) == 0 {
		t.Fatalf("multipass engine never called the LLM provider — engine not invoked through wiring")
	}

	// 2. The LLM saw a verification/polishing prompt (proves the polishing path,
	//    not a plain translate call).
	if p := lastPrompt.Load(); !strings.Contains(p, "verify") && !strings.Contains(p, "polish") &&
		!strings.Contains(strings.ToLower(p), "translation") {
		t.Fatalf("LLM prompt was not a verification/polishing prompt: %q", p)
	}

	// 3. The polished text was applied back into the result (proves output is
	//    genuinely improved/changed through the engine — the dormant capability).
	if !strings.Contains(out, polished) {
		t.Fatalf("polished text not applied. got=%q want it to contain=%q", out, polished)
	}
	if out == inputTranslation {
		t.Fatalf("output unchanged — multipass had no effect")
	}
}

// TestRunMultiPass_PreservesBaseOnProviderFailure proves the guardrail: a multi-
// pass failure must NOT wipe the base translation (§11.4.1 no solve-A-create-B).
func TestRunMultiPass_PreservesBaseOnProviderFailure(t *testing.T) {
	const inputTranslation = "Osnovni prevod koji mora opstati i kad poliranje padne."

	// Server that always errors -> every provider call fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &UnifiedConfig{
		SourceLang: "ru", TargetLang: "sr",
		Provider: "deepseek", Model: "deepseek-chat",
		APIKey: "test-key", BaseURL: srv.URL,
		Timeout: 5 * time.Second, MultiPass: true, MultiPassCount: 1,
	}
	session := &TranslationSession{ID: "mp-fail-session", EventBus: events.NewEventBus()}

	out, err := runMultiPass(context.Background(), cfg, session, "original", inputTranslation)
	if err != nil {
		// The engine surfaces per-provider failures as warnings and falls back to
		// the existing translation; PolishBook itself returns the preserved text.
		// Either way the base translation must survive.
		t.Logf("runMultiPass error (expected, base must still be preserved): %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("multipass wiped the base translation on provider failure — guardrail broken")
	}
	if out != inputTranslation {
		t.Fatalf("on full provider failure the base translation must be returned unchanged, got=%q", out)
	}
}

// TestRunMultiPass_DefaultModelStillPolishes is the §11.4.115 RED-baseline for
// BUG-MULTIPASS-DEFAULT-MODEL. The unified-translator global -model default is
// "gpt-4" (valid ONLY for the openai provider). With any other provider (here
// deepseek), the multipass polish path historically built its LLM client with
// Model="gpt-4", which the ValidModels whitelist rejects ("model 'gpt-4' is not
// valid for provider 'deepseek'"), so the polish silently no-ops and the base
// translation is kept — a §11.4 PASS-bluff (the green "Multi-pass Polishing"
// step ran nothing).
//
// RED on the pre-fix artifact: the LLM is NEVER hit (whitelist reject happens at
// client construction before any request) AND the polished text is NOT applied —
// both assertions FAIL.
// GREEN after the fix: runMultiPass resolves a provider-valid model (DeepSeek's
// own default) when the configured model is invalid for the provider, so the
// engine genuinely runs and applies the polish even with the default -model.
func TestRunMultiPass_DefaultModelStillPolishes(t *testing.T) {
	const inputTranslation = "Ovo je osnovni prevod sa podrazumevanim modelom."
	const polished = "Ovo je doteran prevod iako je model podrazumevan."

	var hits int64
	var lastPrompt atomicString
	srv := httptest.NewServer(openAICompatHandler(t, &hits, polished, &lastPrompt))
	defer srv.Close()

	cfg := &UnifiedConfig{
		SourceLang:     "ru",
		TargetLang:     "sr",
		Provider:       "deepseek",
		Model:          "gpt-4", // the global -model DEFAULT, invalid for deepseek
		APIKey:         "test-key",
		BaseURL:        srv.URL,
		Temperature:    0.3,
		MaxTokens:      512,
		Timeout:        10 * time.Second,
		MultiPass:      true,
		MultiPassCount: 1,
	}
	session := &TranslationSession{
		ID:       "multipass-default-model-session",
		EventBus: events.NewEventBus(),
	}

	out, err := runMultiPass(context.Background(), cfg, session, "original english", inputTranslation)
	if err != nil {
		t.Fatalf("runMultiPass returned error with default model (should resolve a provider-valid model): %v", err)
	}

	// The engine MUST have actually called the LLM — proves the default-model
	// path was resolved to a valid model and the polish genuinely ran (not a
	// silent no-op).
	if atomic.LoadInt64(&hits) == 0 {
		t.Fatalf("BUG-MULTIPASS-DEFAULT-MODEL: engine never called the LLM with default -model gpt-4 on provider deepseek — silent no-op")
	}
	// The polished text MUST be applied — proves the default-model run produced
	// a genuinely polished output, not the kept-base bluff.
	if !strings.Contains(out, polished) {
		t.Fatalf("BUG-MULTIPASS-DEFAULT-MODEL: polished text not applied with default model. got=%q want contains=%q", out, polished)
	}
	if out == inputTranslation {
		t.Fatalf("BUG-MULTIPASS-DEFAULT-MODEL: output unchanged — default-model multipass silently no-opped")
	}
}

// TestResolvePolisherModel covers the full case-space of the BUG-MULTIPASS-DEFAULT-
// MODEL fix helper (§11.4.146 extend-to-all-cases).
func TestResolvePolisherModel(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		model    string
		// want is checked as: exact match if non-empty, else "must be valid for provider"
		want string
	}{
		{"deepseek + default gpt-4 -> deepseek default", "deepseek", "gpt-4", "deepseek-chat"},
		{"deepseek + valid explicit kept", "deepseek", "deepseek-coder", "deepseek-coder"},
		{"openai + gpt-4 kept (valid)", "openai", "gpt-4", "gpt-4"},
		{"deepseek + empty -> deepseek default", "deepseek", "", "deepseek-chat"},
		{"gemini + default gpt-4 -> gemini default", "gemini", "gpt-4", ""}, // any gemini-valid
		{"unknown provider passthrough", "no-such-provider", "gpt-4", "gpt-4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolvePolisherModel(c.provider, c.model)
			if c.want != "" {
				if got != c.want {
					t.Fatalf("resolvePolisherModel(%q,%q)=%q want %q", c.provider, c.model, got, c.want)
				}
				return
			}
			// want=="" means: got must be a provider-valid model (never the rejected input).
			if got == c.model {
				t.Fatalf("resolvePolisherModel(%q,%q) returned the invalid input unchanged", c.provider, c.model)
			}
			if got == "" {
				t.Fatalf("resolvePolisherModel(%q,%q) returned empty", c.provider, c.model)
			}
		})
	}
}

// atomicString is a tiny lock-free string holder for capturing the last prompt.
type atomicString struct{ v atomic.Value }

func (a *atomicString) Store(s string) { a.v.Store(s) }
func (a *atomicString) Load() string {
	if s, ok := a.v.Load().(string); ok {
		return s
	}
	return ""
}
