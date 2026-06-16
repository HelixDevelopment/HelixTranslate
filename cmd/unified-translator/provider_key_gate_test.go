package main

import "testing"

// TestRequiresProviderKey is the §11.4.115 reproduce-first guard for the
// release-critical default-translate path defect found by §11.4.153 video wave 2:
// a bare `unified-translator -i in -o out` (Provider left at its "openai" default,
// no -use-verifier) routes to the LLMsVerifier bridge, which keys its OWN verified
// models — yet the old key gate unconditionally demanded an OPENAI_API_KEY and
// exited 1, making the default path unusable on hosts without that (unused) key.
//
// requiresProviderKey mirrors executeAPITranslation's runtime switch EXACTLY:
// the ONLY runtime path that consumes a per-provider key is the -use-verifier
// (VerifierEnabled) path. The mock seam needs none, and an explicit `-provider X`
// WITHOUT -use-verifier still hits the default bridge case (there is no direct
// per-provider runtime path) — so the gate must NOT fire on it either.
//
// RED on the pre-fix code: the gate fired unconditionally for the default config.
// GREEN on the fix: only VerifierEnabled (non-mock) requires a key.
func TestRequiresProviderKey(t *testing.T) {
	cases := []struct {
		name string
		cfg  *UnifiedConfig
		want bool
	}{
		{
			name: "default bridge path (provider unset) requires NO key",
			cfg:  &UnifiedConfig{Provider: "openai", VerifierEnabled: false},
			want: false,
		},
		{
			name: "explicit -provider without -use-verifier still routes to the bridge → NO key",
			cfg:  &UnifiedConfig{Provider: "deepseek", VerifierEnabled: false},
			want: false,
		},
		{
			name: "-use-verifier requires a key (VerifiedFactory builds a per-provider client)",
			cfg:  &UnifiedConfig{Provider: "openai", VerifierEnabled: true},
			want: true,
		},
		{
			name: "-use-verifier with an explicit provider requires a key",
			cfg:  &UnifiedConfig{Provider: "deepseek", VerifierEnabled: true},
			want: true,
		},
		{
			name: "mock seam never requires a key",
			cfg:  &UnifiedConfig{Provider: "mock", VerifierEnabled: false},
			want: false,
		},
		{
			name: "mock seam never requires a key even with -use-verifier",
			cfg:  &UnifiedConfig{Provider: "mock", VerifierEnabled: true},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := requiresProviderKey(tc.cfg); got != tc.want {
				t.Fatalf("requiresProviderKey(%+v) = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}
