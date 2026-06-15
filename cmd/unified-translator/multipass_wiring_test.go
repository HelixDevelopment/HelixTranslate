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

// atomicString is a tiny lock-free string holder for capturing the last prompt.
type atomicString struct{ v atomic.Value }

func (a *atomicString) Store(s string) { a.v.Store(s) }
func (a *atomicString) Load() string {
	if s, ok := a.v.Load().(string); ok {
		return s
	}
	return ""
}
