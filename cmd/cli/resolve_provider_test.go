package main

import "testing"

// TestResolveProvider guards the provider precedence bug: the CLI -provider flag
// must win over the config DefaultProvider. The original code used
// `providerName == "openai"` (the flag default) as the proxy for "unset", so an
// EXPLICIT `-provider openai` was silently overridden by a config default like
// "deepseek". The fix tracks whether -provider was explicitly set (flag.Visit)
// and only adopts the config default when it was NOT set.
func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name          string
		cliProvider   string
		explicitlySet bool
		configDefault string
		want          string
	}{
		{
			name:          "explicit openai wins over config deepseek (the regression)",
			cliProvider:   "openai",
			explicitlySet: true,
			configDefault: "deepseek",
			want:          "openai",
		},
		{
			name:          "explicit anthropic wins over config deepseek",
			cliProvider:   "anthropic",
			explicitlySet: true,
			configDefault: "deepseek",
			want:          "anthropic",
		},
		{
			name:          "unset falls back to config default",
			cliProvider:   "openai", // flag default value, not explicitly set
			explicitlySet: false,
			configDefault: "deepseek",
			want:          "deepseek",
		},
		{
			name:          "unset with empty config keeps cli default",
			cliProvider:   "openai",
			explicitlySet: false,
			configDefault: "",
			want:          "openai",
		},
		{
			name:          "explicit with empty config keeps cli value",
			cliProvider:   "openai",
			explicitlySet: true,
			configDefault: "",
			want:          "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveProvider(tt.cliProvider, tt.explicitlySet, tt.configDefault)
			if got != tt.want {
				t.Fatalf("resolveProvider(%q, %v, %q) = %q, want %q",
					tt.cliProvider, tt.explicitlySet, tt.configDefault, got, tt.want)
			}
		})
	}
}
