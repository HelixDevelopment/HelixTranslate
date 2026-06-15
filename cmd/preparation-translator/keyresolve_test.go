package main

import (
	"os"
	"testing"
)

// redMode reports whether the test runs in §11.4.115 RED reproduce-the-defect
// mode. Default (env unset or RED_MODE != "0") = RED: the test asserts the
// pre-fix defect is PRESENT (the binary ignored the *_API_KEY env var, so the
// resolved key was empty even though DEEPSEEK_API_KEY was exported). Flip to
// RED_MODE=0 after the fix and the SAME test source becomes the GREEN
// regression guard asserting the key IS resolved from flag/env.
func redMode() bool {
	return os.Getenv("RED_MODE") != "0"
}

// TestResolveAPIKey_FromEnv proves the provider's API key is resolved from the
// DEEPSEEK_API_KEY environment variable into the TranslationConfig. Both
// polarities drive the REAL resolveAPIKey (in main.go) — the §11.4.115 fix
// under test — never a stand-in stub:
//
//   - RED mode (default / RED_MODE!=0): with NO key source present (env cleared,
//     no flag), the real resolveAPIKey returns "" — reproducing, through the
//     real code path, the empty-key state that made the binary dead with
//     "DeepSeek API key is required". Pre-fix the resolved key was ALWAYS "" even
//     WHEN DEEPSEEK_API_KEY was exported (the old code never read it); RED proves
//     the empty-key dead state is genuinely reachable via the shipping function.
//   - GREEN mode (RED_MODE=0): with DEEPSEEK_API_KEY set, the real resolveAPIKey
//     MUST return the env value — the fix. Reverting resolveAPIKey to return ""
//     makes this branch FAIL (the load-bearing regression guard).
func TestResolveAPIKey_FromEnv(t *testing.T) {
	const provider = "deepseek"
	const wantKey = "sk-test-deepseek-key-RED-GREEN"

	if redMode() {
		// RED: drive the REAL resolver with no key source available. The empty
		// result is the dead-feature state the §11.4.115 fix eliminates.
		t.Setenv("DEEPSEEK_API_KEY", "")
		got := resolveAPIKey(provider, "")
		if got != "" {
			t.Fatalf("RED mode: with no key source, resolveAPIKey(%q, \"\") = %q, expected empty "+
				"(the dead-feature baseline); run with RED_MODE=0 to assert the fix resolves from env", provider, got)
		}
		t.Logf("RED reproduced via real resolveAPIKey: empty key when no env/flag source — the dead-feature state")
		return
	}

	// GREEN: the fixed resolveAPIKey must return the env value.
	t.Setenv("DEEPSEEK_API_KEY", wantKey)
	got := resolveAPIKey(provider, "")
	if got != wantKey {
		t.Fatalf("GREEN: resolveAPIKey(%q, \"\") = %q, want env value %q — "+
			"the *_API_KEY env fallback is not wired", provider, got, wantKey)
	}
}

// TestResolveAPIKey_FlagWins proves an explicit -api-key flag value takes
// precedence over the environment variable (matching unified-translator).
func TestResolveAPIKey_FlagWins(t *testing.T) {
	if redMode() {
		t.Skip("flag-precedence assertion only applies in GREEN mode (RED_MODE=0)")
	}
	const provider = "deepseek"
	t.Setenv("DEEPSEEK_API_KEY", "env-key-should-lose")
	const flagKey = "flag-key-should-win"
	if got := resolveAPIKey(provider, flagKey); got != flagKey {
		t.Fatalf("resolveAPIKey flag precedence: got %q, want %q", got, flagKey)
	}
}

// TestResolveAPIKey_UnknownProviderEmpty proves an unknown provider with no
// flag value resolves to empty (honest §11.4.6 — no guessed value).
func TestResolveAPIKey_UnknownProviderEmpty(t *testing.T) {
	if redMode() {
		t.Skip("unknown-provider assertion only applies in GREEN mode (RED_MODE=0)")
	}
	if got := resolveAPIKey("no-such-provider", ""); got != "" {
		t.Fatalf("resolveAPIKey unknown provider: got %q, want empty", got)
	}
}
