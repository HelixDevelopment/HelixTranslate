package main

import (
	"reflect"
	"testing"
)

// TestParseProviders_HonorsFlag is the regression guard for the bug where the
// -providers CLI flag was parsed/logged but silently ignored: PreparationConfig
// hardcoded Providers to []string{"deepseek","zhipu"} ("// Fixed for now"), so a
// user passing -providers openai,anthropic still got deepseek+zhipu.
//
// §11.4.115 polarity: with the fix wired (parseProviders feeds PreparationConfig)
// this is the GREEN regression-guard. On the pre-fix code the helper did not
// exist and the wiring discarded the flag — the package would not compile / the
// flag value would never reach the config, so this test fails by construction.
func TestParseProviders_HonorsFlag(t *testing.T) {
	fallback := []string{"deepseek", "zhipu"}

	tests := []struct {
		name     string
		raw      string
		fallback []string
		want     []string
	}{
		{
			name:     "single provider from flag",
			raw:      "openai",
			fallback: fallback,
			want:     []string{"openai"},
		},
		{
			name:     "multiple providers from flag are all honored",
			raw:      "openai,anthropic,gemini",
			fallback: fallback,
			want:     []string{"openai", "anthropic", "gemini"},
		},
		{
			name:     "spaces around entries are trimmed",
			raw:      "openai, anthropic ,  gemini",
			fallback: fallback,
			want:     []string{"openai", "anthropic", "gemini"},
		},
		{
			name:     "empty entries are dropped",
			raw:      "openai,,anthropic,",
			fallback: fallback,
			want:     []string{"openai", "anthropic"},
		},
		{
			name:     "empty flag falls back to default",
			raw:      "",
			fallback: fallback,
			want:     fallback,
		},
		{
			name:     "only separators/whitespace falls back to default",
			raw:      " , ,  ",
			fallback: fallback,
			want:     fallback,
		},
		{
			name:     "the default flag value deepseek,zhipu round-trips",
			raw:      "deepseek,zhipu",
			fallback: fallback,
			want:     []string{"deepseek", "zhipu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseProviders(tt.raw, tt.fallback)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseProviders(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
