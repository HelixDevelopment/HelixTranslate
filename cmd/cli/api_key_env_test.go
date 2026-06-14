package main

import "testing"

// TestGetAPIKeyFromEnv_Gemini guards a silent-ignore bug: gemini is a
// first-class provider (pkg/translator/llm.ProviderGemini == "gemini",
// gemini.go GetProviderName() == "gemini"), and the primary CLI
// (cmd/unified-translator) maps it to GEMINI_API_KEY. The cmd/cli helper
// getAPIKeyFromEnv was missing the gemini entry, so a user running
// `translator -input book.epub -provider gemini` with GEMINI_API_KEY set
// (and no -api-key) got an empty key silently — the translator was built
// with "" and authentication failed despite the env var being present.
func TestGetAPIKeyFromEnv_Gemini(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "gem-secret-123")

	got := getAPIKeyFromEnv("gemini")
	if got != "gem-secret-123" {
		t.Fatalf("getAPIKeyFromEnv(%q) = %q, want %q (GEMINI_API_KEY silently ignored)",
			"gemini", got, "gem-secret-123")
	}
}

// TestGetAPIKeyFromEnv_KnownProviders is a regression guard for the existing
// provider->env mappings so the fix does not disturb them.
func TestGetAPIKeyFromEnv_KnownProviders(t *testing.T) {
	cases := []struct {
		provider string
		envVar   string
		value    string
	}{
		{"openai", "OPENAI_API_KEY", "oa-1"},
		{"anthropic", "ANTHROPIC_API_KEY", "an-1"},
		{"zhipu", "ZHIPU_API_KEY", "zp-1"},
		{"deepseek", "DEEPSEEK_API_KEY", "ds-1"},
		{"qwen", "QWEN_API_KEY", "qw-1"},
		{"gemini", "GEMINI_API_KEY", "gm-1"},
	}
	for _, c := range cases {
		t.Run(c.provider, func(t *testing.T) {
			t.Setenv(c.envVar, c.value)
			if got := getAPIKeyFromEnv(c.provider); got != c.value {
				t.Fatalf("getAPIKeyFromEnv(%q) = %q, want %q", c.provider, got, c.value)
			}
		})
	}

	// Unknown provider must return empty.
	if got := getAPIKeyFromEnv("nope"); got != "" {
		t.Fatalf("getAPIKeyFromEnv(unknown) = %q, want empty", got)
	}
}
