//go:build integration

// Package llm — REAL provider translation proof (anti-bluff, §11.4 / §11.4.136).
//
// This test makes an ACTUAL network call to a real LLM provider (DeepSeek or
// Groq) using a real API key sourced from the environment, and asserts the
// returned string is genuine translated target-language text — not a
// placeholder, not a session-id, not an error echo. A mock/stub is explicitly
// NOT acceptable for this proof; mock.go is for unit tests elsewhere.
//
// It is gated behind the `integration` build tag, so a plain `go test ./...`
// (no tag) compiles and runs NOTHING here — zero network traffic by default.
//
// Run it with a real key in the environment:
//
//	source "$HOME/api_keys.sh" && \
//	  go test -tags=integration -run TestRealTranslation -v ./pkg/translator/llm/
//
// SKIP-OK (§11.4.3): when neither DEEPSEEK_API_KEY nor GROQ_API_KEY is present
// in the environment, the test SKIPs with a reason rather than failing — the
// required topology (a live provider credential) is genuinely absent.
package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/translator"
)

// realProvider describes one candidate real provider to attempt, in priority
// order. Each maps directly onto the real factory constructor for that
// provider's LLMClient (NewDeepSeekClient / NewGroqClient).
type realProvider struct {
	name     string
	envKey   string
	model    string
	newC     func(translator.TranslationConfig) (LLMClient, error)
}

// candidateProviders returns the ordered set of real providers we will try,
// reading keys from the environment only (§11.4.10 — never hardcoded, never
// logged). The key VALUE is never printed anywhere in this test.
func candidateProviders() []realProvider {
	return []realProvider{
		{
			name:   "deepseek",
			envKey: "DEEPSEEK_API_KEY",
			model:  "deepseek-chat", // a valid DeepSeek model per ValidModels
			newC: func(c translator.TranslationConfig) (LLMClient, error) {
				return NewDeepSeekClient(c)
			},
		},
		{
			name:   "groq",
			envKey: "GROQ_API_KEY",
			model:  "llama-3.1-8b-instant", // a valid Groq model per ValidModels
			newC: func(c translator.TranslationConfig) (LLMClient, error) {
				return NewGroqClient(c)
			},
		},
	}
}

// TestRealTranslation_RealProvider_ProducesRealTargetLanguageText is the
// anti-bluff gold-standard proof: a real provider call yields real Spanish text.
//
// English input  : "The sky is blue today."
// Target language: Spanish (a concrete, assertable expected token: "azul" = blue)
//
// Assertions on the returned string:
//   - non-empty
//   - different from the input
//   - not an obvious placeholder / error / session-id echo
//   - contains real Spanish content — the expected token "azul" (robust:
//     case-insensitive substring, tolerant of surrounding sentence variation).
func TestRealTranslation_RealProvider_ProducesRealTargetLanguageText(t *testing.T) {
	const (
		inputText      = "The sky is blue today."
		expectedToken  = "azul" // Spanish for "blue" — the load-bearing target-language token
		minLen         = 4
		maxLen         = 400 // a single short sentence; guards against runaway / dumped errors
	)

	// Explicit English->Spanish prompt. We drive the LLMClient interface
	// directly (Translate(ctx, text, prompt)) with our own prompt so the
	// target language and the assertable token are under our control, rather
	// than the factory's hardcoded Russian->Serbian literary prompt.
	prompt := "Translate the following English sentence into Spanish. " +
		"Respond with ONLY the Spanish translation, no quotes, no explanation.\n\n" +
		"English: " + inputText + "\nSpanish:"

	candidates := candidateProviders()

	// SKIP cleanly if no provider key is present at all (§11.4.3 topology absent).
	anyKey := false
	for _, p := range candidates {
		if strings.TrimSpace(os.Getenv(p.envKey)) != "" {
			anyKey = true
			break
		}
	}
	if !anyKey {
		t.Skip("SKIP-OK: no real provider key present (set DEEPSEEK_API_KEY or GROQ_API_KEY); " +
			"this test requires a live provider credential (§11.4.3)")
	}

	var lastErr error
	for _, p := range candidates {
		key := strings.TrimSpace(os.Getenv(p.envKey))
		if key == "" {
			t.Logf("provider %s: no %s in env — trying next", p.name, p.envKey)
			continue
		}

		cfg := translator.TranslationConfig{
			Provider:   p.name,
			Model:      p.model,
			APIKey:     key, // VALUE never logged
			SourceLang: "en",
			TargetLang: "es",
			Options: map[string]interface{}{
				"temperature": 0.0, // low temperature => deterministic-enough output
				"max_tokens":  64,
			},
		}

		client, err := p.newC(cfg)
		if err != nil {
			lastErr = err
			t.Logf("provider %s: construction failed: %v — trying next", p.name, err)
			continue
		}

		// Sanity: this is the real provider client, never the mock.
		if got := client.GetProviderName(); got != p.name {
			t.Fatalf("provider %s: GetProviderName() = %q, want %q (wrong/mock client wired)",
				p.name, got, p.name)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		out, terr := client.Translate(ctx, inputText, prompt)
		cancel()
		if terr != nil {
			lastErr = terr
			// Honest report of the real error; try the next provider with a key.
			t.Logf("provider %s (model %s): real Translate call failed: %v — trying next",
				p.name, p.model, terr)
			continue
		}

		out = strings.TrimSpace(out)

		// ---- ANTI-BLUFF ASSERTIONS on REAL output ----------------------------

		// 1) Non-empty.
		if out == "" {
			t.Fatalf("provider %s: real translation is EMPTY — not a usable result", p.name)
		}

		// 2) Different from the input (a passthrough is not a translation).
		if strings.EqualFold(out, inputText) {
			t.Fatalf("provider %s: output equals input %q — no actual translation occurred",
				p.name, inputText)
		}

		// 3) Length sanity — a short sentence, not a placeholder fragment nor a dumped blob.
		if len(out) < minLen || len(out) > maxLen {
			t.Fatalf("provider %s: output length %d out of expected range [%d,%d]: %q",
				p.name, len(out), minLen, maxLen, out)
		}

		// 4) Not an obvious placeholder / error / session-id echo.
		lower := strings.ToLower(out)
		for _, bad := range []string{
			"placeholder", "todo", "lorem ipsum", "error", "session_id",
			"session id", "n/a", "null", "<", "api key", "unauthorized",
		} {
			if strings.Contains(lower, bad) {
				t.Fatalf("provider %s: output looks like a placeholder/error (contains %q): %q",
					p.name, bad, out)
			}
		}

		// 5) Real target-language (Spanish) content — the load-bearing token.
		//    "blue" -> "azul". Robust: case-insensitive substring match.
		if !strings.Contains(lower, expectedToken) {
			t.Fatalf("provider %s: output does NOT contain expected Spanish token %q "+
				"(translation of 'blue') — got: %q", p.name, expectedToken, out)
		}

		// PASS — capture the real input->output for the anti-bluff evidence trail.
		// (No key value is ever printed; only the model and the translated text.)
		t.Logf("ANTI-BLUFF PROOF — real provider call succeeded\n"+
			"  provider : %s\n"+
			"  model    : %s\n"+
			"  input    : %q\n"+
			"  output   : %q\n"+
			"  asserted : non-empty, != input, len in [%d,%d], no placeholder/error, contains %q",
			p.name, p.model, inputText, out, minLen, maxLen, expectedToken)
		return // success on this provider — done.
	}

	// Every provider with a key failed — report the real last error honestly.
	if lastErr != nil {
		t.Fatalf("all candidate providers with a key failed; last real error: %v", lastErr)
	}
	t.Skip("SKIP-OK: no provider key was usable (§11.4.3)")
}
